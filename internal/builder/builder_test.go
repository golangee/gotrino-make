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

package builder_test

import (
	"github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/gotool"
	"github.com/golangee/gotrino-make/internal/hashtree"
	"os"
	"path/filepath"
	"testing"
)

func TestGoBuildWasm(t *testing.T) {
	builder.Debug = true
	hashtree.Debug = true
	gotool.Debug = true

	tmpDir := filepath.Join(os.TempDir(), "gotrino-make")
	prjDir := "/Users/tschinke/git/github.com/golangee/forms-example/www/"
	prj, err := builder.NewProject(prjDir, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := prj.Build(false, true); err != nil {
			t.Fatal(err)
		}
	}

}
