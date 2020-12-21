package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/golangee/gotrino-make/internal/git"
	"github.com/golangee/gotrino-make/internal/gotool"
	"github.com/golangee/gotrino-make/internal/hashtree"
	"github.com/golangee/gotrino-make/internal/io"
	"github.com/golangee/log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	wasmFilename       = "app.wasm"
	goRootJsBridge     = "misc/wasm/wasm_exec.js"
	wasmBridgeFilename = "wasm_exec.js"
	staticFolder       = "static"
)

// Debug is a global flag, which is only used by the command line program to track errors down.
var Debug = false

// Options to use for building.
type Options struct {
	Force            bool
	HotReload        bool
	TemplatePatterns []string
	Extra            interface{}
	Debug            bool
	GoGenerate       bool
}

// A Part of a Project.
type Part struct {
	mod gotool.Module
	src *hashtree.Node // the file tree of mod.Dir
}

// refresh reads the src it represents the current state of the filesystem.
// If the force flag is true, the entire directory content is hashed again, instead of using the ModTime as
// a delta indicator. The directory is mod.Dir+static
func (p *Part) refresh(force bool, subDir string) error {
	exists := true
	dir := filepath.Join(p.mod.Dir, subDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		exists = false
	}

	if p.src == nil || force || !exists {
		p.src = hashtree.NewNode()
		p.src.Mode = os.ModeDir
	}

	if !exists {
		return nil
	}

	if err := hashtree.ReadDir(dir, p.src); err != nil {
		return fmt.Errorf("unable to hash src: %w", err)
	}

	return nil
}

// A Project is kept usually in-memory to efficiently (re-)build a Go module with dependent other modules.
type Project struct {
	srcPath       string // srcPath contains the source go module.
	main          *Part
	mods          []*Part // modules contains at least 1 module. The first module is always the main module.
	dst           *hashtree.Node
	dstPath       string   // the actual target directory to merge everything into.
	extraDstFiles []string // absolute file names in dstPath which must/need not to be deleted.
	lastBuildHash [32]byte
}

// NewProject allocates a new project and setups one-time things.
func NewProject(dstPath, srcPath string) (*Project, error) {
	p := &Project{
		srcPath: srcPath,
		dstPath: dstPath,
	}

	if err := p.copyWasmBridge(); err != nil {
		return nil, fmt.Errorf("unable to provide the current Go WASM bridge: %w", err)
	}

	return p, nil
}

func (p *Project) copyWasmBridge() error {
	if err := os.MkdirAll(p.dstPath, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create build directory: %s: %w", p.dstPath, err)
	}

	goRoot, err := gotool.Env("GOROOT")
	if err != nil || goRoot == "" {
		return fmt.Errorf("unable to determine GOROOT: %w", err)
	}

	wasmDstFile := filepath.Join(p.dstPath, wasmBridgeFilename)
	if err := io.CopyFile(wasmDstFile, filepath.Join(goRoot, goRootJsBridge)); err != nil {
		return fmt.Errorf("unable to provide wasm-js-bridge: %w", err)
	}

	p.extraDstFiles = append(p.extraDstFiles, wasmDstFile)

	return nil
}

// loadMods refreshes the modules. It tries to avoid resetting modules, to keep their state in-memory and allow delta
// updates.
func (p *Project) loadMods() error {
	str, err := gotool.ModTidy(p.srcPath) // otherwise the Dir folders may be empty, because no sources have been loaded
	if err != nil {
		return fmt.Errorf("unable to go mod tidy: %w", err)
	}

	if Debug {
		log.Println(str)
	}

	mods, err := gotool.ModList(p.srcPath)
	if err != nil {
		return fmt.Errorf("unable to list modules: %w", err)
	}

	if len(mods) == 0 || !mods[0].Main {
		return fmt.Errorf("no main module found: %s", p.srcPath)
	}

	rebuild := false

	if len(mods) != len(p.mods) {
		rebuild = true
	} else {
		for i := range mods {
			if mods[i].Dir != p.mods[i].mod.Dir || mods[i].Version != p.mods[i].mod.Version {
				if Debug {
					log.Println(fmt.Sprintf("modules at index %d are different: \n%+v\n%+v", i, p.mods[i].mod, mods[i]))
				}

				rebuild = true
				break
			}
		}
	}

	if rebuild {
		if Debug {
			log.Println("modules have changed, reloading all modules")
		}

		parts := make([]*Part, 0, len(mods))
		for _, mod := range mods {
			parts = append(parts, &Part{mod: mod})
		}

		p.mods = parts
		p.main = &Part{mod: mods[0]}
	}

	return nil
}

// refresh syncs all internal hashtree.Node roots to be equal to the filesystem (which may race logically). Force
// will calculates all hashes, instead of re-using already calculated ones.
func (p *Project) refresh(force bool) error {
	for _, mod := range p.mods {
		if err := mod.refresh(force, staticFolder); err != nil {
			return fmt.Errorf("unable to refresh module: %w", err)
		}
	}

	if err := p.main.refresh(force, ""); err != nil {
		return fmt.Errorf("unable to refresh main root: %w", err)
	}

	if p.dst == nil || force {
		p.dst = hashtree.NewNode()
		p.dst.Mode = os.ModeDir
	}

	if err := hashtree.ReadDir(p.dstPath, p.dst); err != nil {
		return fmt.Errorf("unable to hash dst: %w", err)
	}

	return nil
}

// sync writes only different files from src to dst based on the current meta data.
// Actually we assemble a virtual overlay, so that we can determine which files are shadowed and need to be actually
// copied and written over (only once) and which files are extra.
func (p *Project) sync() error {

	var srcTree []hashtree.File

	// reverse order: the natural order is, that at index 0, we have the main module
	for i := len(p.mods) - 1; i >= 0; i-- {
		mod := p.mods[i]
		srcTree = hashtree.PutTop(srcTree, mod.src.Flatten(filepath.Join(mod.mod.Dir, staticFolder)))
	}

	dstTree := p.dst.Flatten(p.dstPath)

	// copy only files which are different in content or do not exist at all
	for _, file := range srcTree {
		idx := hashtree.FindFile(dstTree, file.Filename)
		if idx == -1 || file.Node.Hash != dstTree[idx].Node.Hash {
			from := filepath.Join(file.Prefix, file.Filename)
			to := filepath.Join(p.dstPath, file.Filename)

			if file.Node.Mode.IsDir() {
				if Debug {
					log.Println(fmt.Sprintf("mkdir folder %s -> %s", from, to))
				}

				if err := os.MkdirAll(to, os.ModePerm); err != nil {
					return fmt.Errorf("unable to create target folder: %w", err)
				}

				continue
			}

			if err := os.MkdirAll(filepath.Dir(from), os.ModePerm); err != nil {
				return fmt.Errorf("unable to create copy-folder: %w", err)
			}

			if Debug {
				log.Println(fmt.Sprintf("copy modified file %s -> %s", from, to))
			}

			if err := io.CopyFile(to, from); err != nil {
				return fmt.Errorf("fail to copy file: %w", err)
			}
		} else {
			if Debug {
				log.Println(fmt.Sprintf("sync: unmodified %s", file.Filename))
			}
		}
	}

	// remove extra files
NextFile:
	for _, file := range dstTree {
		idx := hashtree.FindFile(srcTree, file.Filename)
		if idx == -1 {
			to := filepath.Join(file.Prefix, file.Filename)

			for _, dstFile := range p.extraDstFiles {
				if to == dstFile {
					continue NextFile
				}
			}

			if Debug {
				log.Println(fmt.Sprintf("removing extra file file %s", to))
			}

			if err := os.RemoveAll(to); err != nil {
				return fmt.Errorf("failed to remove extra file: %w", err)
			}
		}
	}

	return nil
}

// srcHash calculates an uber hash from all source modules.
func (p *Project) srcHash() [32]byte {
	hasher := sha256.New()
	for _, mod := range p.mods {
		hasher.Write(mod.src.Hash[:])
	}

	hasher.Write(p.main.src.Hash[:])

	var r [32]byte
	tmp := hasher.Sum(nil)
	copy(r[:], tmp)

	return r
}

// Build syncs the file tree of all modules into the build destination directory and compiles the web assembly.
// Returns the unique hash of the last build.
func (p *Project) Build(opts Options) ([32]byte, error) {
	start := time.Now()
	defer func() {
		log.Println(fmt.Sprintf("build duration: %v", time.Now().Sub(start)))
	}()

	if err := os.MkdirAll(p.dstPath, os.ModePerm); err != nil {
		return p.lastBuildHash, fmt.Errorf("unable to create build directory: %s: %w", p.dstPath, err)
	}

	if err := p.loadMods(); err != nil {
		return p.lastBuildHash, fmt.Errorf("unable to load modules: %w", err)
	}

	if err := p.refresh(opts.Force); err != nil {
		return p.lastBuildHash, fmt.Errorf("unable to refresh file hashes: %w", err)
	}

	// only compare originally synced hashes, to avoid any other copy work, which just creates invalid
	// intermediate builder states
	uberHash := p.srcHash()
	if uberHash == p.lastBuildHash {
		if Debug {
			log.Println(fmt.Sprintf("hash unchanged, no build required: %s", hex.EncodeToString(uberHash[:])))
		}

		return p.lastBuildHash, nil
	}

	if opts.GoGenerate {
		if Debug {
			log.Println("invoking go generate ./...")
		}

		genPrints, err := gotool.Generate(p.srcPath)
		if err != nil {
			return p.lastBuildHash, fmt.Errorf("failed to go generate: %w", err)
		}

		if Debug {
			log.Println(genPrints)
		}

		// need to refresh again
		if err := p.refresh(opts.Force); err != nil {
			return p.lastBuildHash, fmt.Errorf("unable to refresh file hashes: %w", err)
		}
	}

	// reset our last build hash, otherwise we may get weired build/bug/revert/non-build inconsistencies
	for i := range p.lastBuildHash {
		p.lastBuildHash[i] = 0
	}

	if Debug {
		log.Println(fmt.Sprintf("build hash changed, old: %s new: %s", hex.EncodeToString(p.lastBuildHash[:]), hex.EncodeToString(uberHash[:])))
	}

	// copy all original stuff over, sync also deletes generated extra files like wasm and templates
	if err := p.sync(); err != nil {
		return p.lastBuildHash, fmt.Errorf("cannot sync file trees: %w", err)
	}

	// try to actually build, every other error until now was critical
	buildInfo := BuildInfo{
		Time:      time.Now(),
		Version:   hex.EncodeToString(uberHash[:]),
		HotReload: opts.HotReload,
		Extra:     opts.Extra,
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Println("unable to read hostname", err)
	}

	buildInfo.Host = hostname

	gitCommit, err := git.Head(p.srcPath)
	if err != nil {
		log.Println("unable to read git head", err)
	}

	buildInfo.Commit = gitCommit

	goVersion, err := gotool.Version()
	if err != nil {
		log.Println("unable to get go compiler version", err)
	}

	buildInfo.Compiler = goVersion

	if err := gotool.BuildWasm(p.mods[0].mod, filepath.Join(p.dstPath, wasmFilename)); err != nil {
		buildInfo.CompileError = err
		if Debug {
			log.Println("wasm build failed", err)
		}
	} else {
		buildInfo.Wasm = true
		if Debug {
			log.Println("wasm build successful")
		}
	}

	// apply all templates to files like *.gocss or *.gohtml
	allFiles, err := listAllFiles(p.dstPath)
	if err != nil {
		return p.lastBuildHash, err
	}

GoTemplateLoop:
	for _, file := range allFiles {
		ext := strings.ToLower(filepath.Ext(file))
		for _, pattern := range opts.TemplatePatterns {
			if pattern == ext {
				if Debug {
					log.Println(fmt.Sprintf("found template file: %s", file))
				}

				_, err := buildInfo.applyTemplate(file)
				if err != nil {
					log.Println("template error", err)
				}

				if err != nil && buildInfo.CompileError == nil {
					buildInfo.CompileError = err
					break GoTemplateLoop
				}

			}
		}
	}

	if buildInfo.HasError() {
		if Debug {
			log.Println("build has errors")
		}
		return p.lastBuildHash, CompileErr{delegate: buildInfo.CompileError}
	}

	p.lastBuildHash = uberHash

	if Debug {
		log.Println(fmt.Sprintf("build completed: %s", hex.EncodeToString(p.lastBuildHash[:])))
	}

	return p.lastBuildHash, nil
}

func listAllFiles(root string) ([]string, error) {
	var res []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if info.Mode().IsRegular() {
			res = append(res, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("cannot list files: %w", err)
	}

	return res, nil
}
