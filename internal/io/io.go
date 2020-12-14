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

package io

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// try executes the given func and updates the error,
// but only if it has not been set yet.
func try(f func() error, err *error) {
	newErr := f()
	if *err == nil {
		*err = newErr
	}
}

// CopyFile copies a file from src to dst
func CopyFile(dst, src string) (err error) {
	// delete target file first, ensure that the FS looses all meta data.
	if err = os.RemoveAll(dst); err != nil {
		return err
	}

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open dst file: %w", err)
	}
	defer try(df.Close, &err)

	sf, err := os.OpenFile(src, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open src file: %w", err)
	}
	defer try(sf.Close, &err)

	if _, err := io.Copy(df, sf); err != nil {
		return fmt.Errorf("unable to copy file bytes: %w", err)
	}

	return nil
}

// CopyDir copies from source to dst overwriting any existing files. Extra files are not removed.
// Hidden files are ignored.
func CopyDir(dst, src string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		srcPath := filepath.Join(src, file.Name())
		dstPath := filepath.Join(dst, file.Name())
		if file.IsDir() {
			if err := os.MkdirAll(dstPath, os.ModePerm); err != nil {
				return err
			}

			if err := CopyDir(dstPath, srcPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(dstPath, srcPath); err != nil {
				return err
			}
		}
	}

	return nil
}
