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
	"testing"
)

func TestGoBuildWasm(t *testing.T) {
	prjDir := "/Users/tschinke/git/github.com/golangee/forms-example/www/"
	hash, err := HashFileTree(prjDir)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", hex.EncodeToString(hash))

	root, err := GoEnv("GOROOT")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", root)

	err = GoBuildWasm(prjDir, "bla.wasm")
	if err != nil {
		t.Fatal(err)
	}
}
