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

package builder

import (
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	wasmFilename       = "app.wasm"
	goRootJsBridge     = "misc/wasm/wasm_exec.js"
	wasmBridgeFilename = "wasm_exec.js"
	indexHtml          = "index.gohtml"
)

type BuildErr struct {
	delegate error
}

func (b BuildErr) Error() string {
	return b.delegate.Error()
}

func (b BuildErr) Unwrap() error {
	return b.delegate
}




// BuildProject builds an entire golangee wasm project from src to dst.
func BuildProject(srcDir, dstDir string) error {
	wasmHash, err := HashFileTree(srcDir)
	if err != nil {
		return fmt.Errorf("unable to calculate hash version: %w", err)
	}

	goRoot, err := GoEnv("GOROOT")
	if err != nil || goRoot == "" {
		return fmt.Errorf("unable to determine GOROOT: %w", err)
	}

	if err := CopyFile(filepath.Join(dstDir, wasmBridgeFilename), filepath.Join(goRoot, goRootJsBridge)); err != nil {
		return fmt.Errorf("unable to provide wasm-js-bridge: %w", err)
	}

	allDeps, err := dependencyDirectories(srcDir)
	if err != nil {
		return fmt.Errorf("unable to find dependencies: %w", err)
	}

	// TODO this is inefficient, we should ever read the source and target directory once and compare and update only the different files
	for i := len(allDeps) - 1; i >= 0; i-- {
		staticDepDir := filepath.Join(allDeps[i], "static")
		if _, err := os.Stat(staticDepDir); err != nil {
			continue // no static folder
		}

		fmt.Println("copy", staticDepDir)
		if err := CopyDir(dstDir, staticDepDir); err != nil {
			return fmt.Errorf("unable to copy static: %w", err)
		}
	}

	bridgeHash, err := HashFile(filepath.Join(goRoot, goRootJsBridge))
	if err != nil {
		return fmt.Errorf("unable to hash bridge js: %w", err)
	}

	idxDat := IndexData{
		WasmVersion:       hex.EncodeToString(wasmHash),
		WasmBridgeVersion: hex.EncodeToString(bridgeHash),
		HotReload:         true,
	}

	buildErr := GoBuildWasm(srcDir, filepath.Join(dstDir, wasmFilename))

	if buildErr != nil {
		idxDat.Body = buildErrAsHtml(buildErr.Error())
		idxDat.LoadWasm = false
	} else {
		idxDat.LoadWasm = true
	}

	if err := RewriteHTML(filepath.Join(dstDir, indexHtml), idxDat); err != nil {
		return fmt.Errorf("unable to create index html: %w", err)
	}

	_ = os.Remove(filepath.Join(dstDir, indexHtml))

	if buildErr != nil {
		return BuildErr{buildErr}
	}

	return nil
}

func buildErrAsHtml(str string) string {
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
