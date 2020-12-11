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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// dependencyDirectories returns all local folders to each correct dependency version. The first
// returned directory is the main directory.
func dependencyDirectories(moduleDir string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	cmd.Dir = moduleDir
	cmd.Env = os.Environ()

	res, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("unable to grab dependencies: %w", err)
	}

	str := "[" + strings.ReplaceAll(string(res), "}\n{", "},\n{") + "]"

	type meta struct {
		Dir string
	}

	var metas []meta
	var list []string
	if err := json.Unmarshal([]byte(str), &metas); err != nil {
		return nil, fmt.Errorf("unable grab results: %w", err)
	}

	for _, r := range metas {
		list = append(list, r.Dir)
	}

	return list, nil
}
