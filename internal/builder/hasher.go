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
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// HashFile just calculates the hash for a single file.
func HashFile(fname string) ([]byte, error) {
	f, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		_ = f.Close()
		return nil, err
	}

	return h.Sum(nil), nil
}

// HashFileTree walks in sorted order from root to any leaf. It ignores anything starting with a dot.
// It a directory matches that name, it is ignored entirely.
func HashFileTree(root string) ([]byte, error) {
	ignores := []string{"."}
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		for _, s := range ignores {
			if strings.HasPrefix(info.Name(), s) {
				if info.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}
		}

		if info.Mode().IsRegular() {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	h := sha256.New()
	for _, file := range files {
		f, err := os.OpenFile(file, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}

		if _, err = io.Copy(h, f); err != nil {
			_ = f.Close()
			return nil, err
		}

		_ = f.Close()
	}

	return h.Sum(nil), nil
}
