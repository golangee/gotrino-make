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

package hashtree

import "sort"

// try executes the given func and updates the error,
// but only if it has not been set yet.
func try(f func() error, err *error) {
	newErr := f()
	if *err == nil {
		*err = newErr
	}
}

// PutTop inserts or updates all entries from src on dst. The result is returned.
func PutTop(dst, src []File) []File {
	tmp := map[string]File{} // that is expensive, we surely may want to use a slice with memcpy instead
	for _, file := range dst {
		tmp[file.Filename] = file
	}

	for _, file := range src {
		tmp[file.Filename] = file
	}

	res := make([]File, 0, len(tmp))

	for _, file := range tmp {
		res = append(res, file)
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Filename < res[j].Filename
	})

	return res
}

// FindFile returns the first entry index or -1 if not found. Expects that s is sorted ascending by name.
func FindFile(s []File, name string) int {
	idx := sort.Search(len(s), func(i int) bool {
		return s[i].Filename >= name
	})

	if idx < len(s) && s[idx].Filename == name {
		return idx
	}

	return -1
}
