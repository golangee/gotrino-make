package builder

import (
	"bytes"
	"fmt"
	"github.com/golangee/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// A CompileErr denotes a special build error, which is solely related to applying templates or the go compiler.
type CompileErr struct {
	delegate error
}

func (b CompileErr) Error() string {
	return b.delegate.Error()
}

func (b CompileErr) Unwrap() error {
	return b.delegate
}

// BuildInfo provides some basic information about a gotrino build.
type BuildInfo struct {
	// Time of this build.
	Time time.Time
	// Version contains a hash or something else which uniquely identifies this build.
	Version string
	// CompileError is nil or contains a compile error.
	CompileError error
	// HotReload is true, if the server should be polled at /api/v1/poll/version.
	HotReload bool
	// Wasm is true, if the web assembly (app.wasm) is available.
	Wasm bool
	// Commit may be empty, if the project is not contained in a git repository.
	Commit string
	// Host name.
	Host string
	// Compiler denotes the compiler which has created the wasm build.
	Compiler string
	// Extra may be nil or injected by user.
	Extra interface{}
}

// HasError returns true, if something went wrong while building.
func (b BuildInfo) HasError() bool {
	return b.CompileError != nil
}

// Error returns an html formatted error description. Check HasError before.
func (b BuildInfo) Error() string {
	str := ""
	if b.CompileError != nil {
		str = b.CompileError.Error()
	}

	sb := &strings.Builder{}
	sb.WriteString("<div class=\"h-screen bg-gray-600 p-10\">")
	sb.WriteString("<div class=\"bg-white max-w-6xl p-1 rounded overflow-hidden shadow-lg dark:bg-gray-800\">\n")
	sb.WriteString("<p class=\"text-xl text-red-600\">build error</p>")
	for _, line := range strings.Split(str, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "exit status") {
			sb.WriteString("<p class=\"text-base medium\">")
		} else {
			sb.WriteString("<p class=\"text-base text-red-600 medium\">")
		}
		sb.WriteString(line)
		sb.WriteString("</p>\n")
	}
	sb.WriteString("</div>\n")
	sb.WriteString("</div>\n")
	return sb.String()
}

// applyTemplate reads the given file, applies it as a text/template and writes it back again. If file name contains
// a *.go<ext> pattern, the 'go' part is removed, also like the original file as well. The (new) written file name
// returned.
func (b BuildInfo) applyTemplate(fname string) (string, error) {
	rawText, err := ioutil.ReadFile(fname)
	if err != nil {
		return "", fmt.Errorf("unable to read template file: %w", err)
	}

	text := string(rawText)

	tpl, err := template.New(fname).Parse(text)
	if err != nil {
		return "", fmt.Errorf("unable to parse text template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, b); err != nil {
		return "", fmt.Errorf("unable to execute BuildInfo template: %w", err)
	}

	dstFile := fname
	myExt := filepath.Ext(fname)
	if strings.HasPrefix(myExt, ".go") {
		dstFile = fname[0:len(dstFile)-len(myExt)] + "." + myExt[3:]
	}

	if Debug {
		log.Println(fmt.Sprintf("BuildInfo: wrote template file to: %s", dstFile))
	}

	if err := ioutil.WriteFile(dstFile, buf.Bytes(), os.ModePerm); err != nil {
		return "", fmt.Errorf("unable to write target file: %w", err)
	}

	if dstFile != fname {
		if Debug {
			log.Println(fmt.Sprintf("BuildInfo: remove extra file: %s", fname))
		}

		if err := os.RemoveAll(fname); err != nil {
			return "", fmt.Errorf("cannot remove source file: %w", err)
		}
	}

	return dstFile, nil
}
