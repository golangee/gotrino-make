// Copyright 2020 Torben Schinke
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gotool

import (
	"encoding/json"
	"fmt"
	"github.com/golangee/log"
	"os"
	"os/exec"
	"strings"
)

// Debug is a global flag, which is only used by the command line program to track errors down.
var Debug = false

// A Module describes the anatomy of Go Module.
type Module struct {
	Path    string // Path is the module import path or the module name
	Main    bool   // Main tells whether this module is the actual main module
	Dir     string // Dir is the local folder, where the actual source code resides
	Version string // Version is usually something like v1.2.3 or v0.0.0-20201210172659-1ccebcf04a20
	Replace struct {
		Dir string // Dir is the actual local replacement directory
	}
}

// ModTidy invokes go mod tidy in the given directory. It will clean up deps and download their source.
// See also https://golang.org/ref/mod#go-mod-tidy.
func ModTidy(dir string) (string, error) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Env = os.Environ()
	cmd.Dir = dir

	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot go generate: %s: %w", string(res), err)
	}

	return strings.TrimSpace(string(res)), nil
}

// Generate invokes go generate ./... in the given directory.
func Generate(dir string) (string, error) {
	cmd := exec.Command("go", "generate", "./...")
	cmd.Env = os.Environ()
	cmd.Dir = dir

	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot go generate: %s: %w", string(res), err)
	}

	return strings.TrimSpace(string(res)), nil
}

// Version returns the go version.
func Version() (string, error) {
	cmd := exec.Command("go", "version")
	cmd.Env = os.Environ()

	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to 'go version': %w", err)
	}

	return strings.TrimSpace(string(res)), nil
}

// ModList returns all local folders to each correct dependency version. The first
// returned directory is the main directory.
func ModList(moduleDir string) ([]Module, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = moduleDir
	cmd.Env = os.Environ()

	res, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("unable to grab dependencies: %w", err)
	}

	str := "[" + strings.ReplaceAll(string(res), "}\n{", "},\n{") + "]"

	var tmp []Module
	if err := json.Unmarshal([]byte(str), &tmp); err != nil {
		return nil, fmt.Errorf("unable grab results: %w", err)
	}

	modules := make([]Module, 0, len(tmp))

	// unclear what an empty Dir means, but we cannot work with it properly. There is no documentation
	// for it: https://github.com/golang/go/blob/master/src/cmd/go/internal/list/list.go
	for _, module := range tmp {
		if module.Dir == "" {
			if Debug {
				log.Println("modList: ignoring module without directory: " + module.Path)
			}
		} else {
			modules = append(modules, module)
		}
	}

	return modules, nil
}

// BuildWasm builds an idiomatic wasm go module. The wasm main entry point must be defined at cmd/wasm. The
// output file is forwarded.
func BuildWasm(mod Module, outFile string) error {
	err := Build(Options{
		GOOS:       "js",
		GOARCH:     "wasm",
		WorkingDir: mod.Dir,
		Output:     outFile,
		Packages:   []string{mod.Path + "/cmd/wasm"}, // this is our convention
		LDFLAGS: LDFLAGS{

		},
	})

	if err != nil {
		return err
	}

	return nil
}

// Options represent the various build options for the go build command.
type Options struct {
	GOOS       string
	GOARCH     string
	WorkingDir string
	Output     string
	Packages   []string
	Env        []string
	LDFLAGS    LDFLAGS
}

// LDFLAGS represent the go linker flags.
type LDFLAGS struct {
	// X is to inject variables at compilation/linking time.
	X []string
}

// String returns the linker flags.
func (f *LDFLAGS) String() string {
	sb := &strings.Builder{}
	for _, x := range f.X {
		sb.WriteString("-X ")
		sb.WriteString(x)
		sb.WriteString(" ")
	}

	return sb.String()
}

// Build just issues the go build command.
func Build(opts Options) error {
	args := []string{"build"}
	ldflags := opts.LDFLAGS.String()
	if ldflags != "" {
		args = append(args, "-ldflags", "\""+ldflags+"\"")
	}

	if opts.Output != "" {
		args = append(args, "-o", opts.Output)
	}

	for _, p := range opts.Packages {
		args = append(args, p)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = opts.WorkingDir
	cmd.Env = opts.Env
	if len(cmd.Env) == 0 {
		cmd.Env = append(cmd.Env, os.Environ()...)
	}

	if opts.GOOS != "" {
		cmd.Env = append(cmd.Env, "GOOS="+opts.GOOS)
	}

	if opts.GOARCH != "" {
		cmd.Env = append(cmd.Env, "GOARCH="+opts.GOARCH)
	}

	res, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(res))
	}

	return nil
}

// Env requests the given parameter name.
func Env(name string) (string, error) {
	cmd := exec.Command("go", "env", name)
	cmd.Env = os.Environ()
	res, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, string(res))
	}

	for _, s := range strings.Split(string(res), "\n") {
		s = strings.TrimSpace(s)
		if s != "" {
			return s, nil
		}
	}

	return "", nil
}
