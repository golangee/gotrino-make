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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

type IndexData struct {
	WasmVersion       string
	WasmBridgeVersion string
	Body              string
	HotReload         bool
	LoadWasm          bool
}

// RewriteTemplate reads the given file, applies it as a template and writes it back again (as *.html).
func RewriteHTML(file string, indexData IndexData) error {

	rawHtml, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("unable to read .gohtml file: %w", err)
	}

	html := string(rawHtml)

	tpl, err := template.New("index.html").Parse(html)
	if err != nil {
		return fmt.Errorf("unable to parse html template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, indexData); err != nil {
		return fmt.Errorf("unable to apply index template: %w", err)
	}

	myExt := filepath.Ext(file)
	dstFile := file[0:len(file)-len(myExt)] + ".html"

	return ioutil.WriteFile(dstFile, buf.Bytes(), os.ModePerm)
}
