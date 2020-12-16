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

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/golangee/log"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Debug is a global flag, which is only used by the command line program to track errors down.
var Debug = false

// File should represent a real physical file with the given meta data. It still virtual, as the file may not exist.
type File struct {
	Prefix   string // Prefix is a constant
	Filename string // Filename is a relative but full file name
	Node     *Node
}

// A Node is an element in a merkle tree. This one represents a part of the real filesystem. Using a hash tree,
// we can efficiently decide and find changes in very large and complex trees.
type Node struct {
	Hash     [32]byte
	Name     string
	Mode     os.FileMode
	ModTime  time.Time
	Children []*Node
}

func NewNode() *Node {
	return &Node{}
}

// Flatten returns hashtree files with absolute file names according to the given root. The array is sorted ascending.
func (n *Node) Flatten(prefix string) []File {
	return n.flatten(prefix, "")
}

func (n *Node) flatten(prefix, root string) []File {
	res := make([]File, 0, len(n.Children)+1)

	res = append(res, File{
		Prefix:   prefix,
		Filename: filepath.Join(root, n.Name),
		Node:     n,
	})

	for _, child := range n.Children {
		flatChildren := child.flatten(prefix, filepath.Join(root, n.Name))
		for _, flatChild := range flatChildren {
			res = append(res, flatChild)
		}
	}

	return res
}

// IndexOf returns the found index or nil in log(n), because children are sorted ascending by name.
func (n *Node) IndexOf(name string) int {
	idx := sort.Search(len(n.Children), func(i int) bool {
		return n.Children[i].Name >= name
	})

	if idx >= len(n.Children) || n.Children[idx].Name != name {
		return -1
	}

	return idx
}

// Find returns the first Node with the given name or nil.
func (n *Node) Find(name string) *Node {
	idx := n.IndexOf(name)
	if idx >= 0 {
		return n.Children[idx]
	}

	return nil
}

// RemoveAt deletes the node at the given index
func (n *Node) RemoveAt(i int) {
	copy(n.Children[i:], n.Children[i+1:])
	n.Children[len(n.Children)-1] = nil
	n.Children = n.Children[:len(n.Children)-1]
}

// Remove deletes the first child with that name
func (n *Node) Remove(name string) {
	idx := n.IndexOf(name)
	if idx < 0 {
		return
	}

	n.RemoveAt(idx)
}

// Add inserts the given child and keeps a sorted order.
func (n *Node) Add(child *Node) {
	n.Remove(child.Name)
	n.Children = append(n.Children, child)

	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Name < n.Children[j].Name
	})
}

// Read just calculates the sha256 value for a single file.
func Read(fname string) (r [32]byte, err error) {
	f, err := os.OpenFile(fname, os.O_RDONLY, 0)
	if err != nil {
		return r, err
	}

	defer try(f.Close, &err)
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		_ = f.Close() // read-only err not of interest
		return r, err
	}

	tmp := h.Sum(nil)
	copy(r[:], tmp)

	return r, nil
}

// ReadDir walks in sorted order from root to any leaf. It ignores anything starting with a dot.
// If a directory matches that name, it is ignored entirely. To improve performance, it will only
// ever read leaf-files if they are unknown or if the ModTime is different. Extra in-memory nodes are
// removed, if they are not present in the filesystem anymore. Note that this performance improvement will
// fail on systems where the ModTime is not updated or the timer resolution is not small enough.
func ReadDir(rootDir string, parent *Node) error {
	files, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return fmt.Errorf("unable to list directory: '%s': %w", rootDir, err)
	}

	hasher := sha256.New()
	var currentFiles []string
	for _, file := range files {
		if fileIgnored(file.Name()) {
			continue
		}

		currentFiles = append(currentFiles, file.Name())
		absolutePath := filepath.Join(rootDir, file.Name())
		node := parent.Find(file.Name())

		// check if we already know that file
		if node != nil && node.Mode.IsRegular() && node.Mode == file.Mode() && node.ModTime == file.ModTime() {
			if Debug {
				log.Println(fmt.Sprintf("hashtree: %s: file not changed, do not read file: %s", rootDir, file.Name()))
			}

			continue
		}

		// if it is a directory or changed, descend
		if node == nil || node.Mode != file.Mode() {
			node = &Node{
				Name:    file.Name(),
				Mode:    file.Mode(),
				ModTime: file.ModTime(),
			}
		}

		if file.Mode().IsRegular() {
			h, err := Read(absolutePath)
			if err != nil {
				return fmt.Errorf("unable to calculate file hash sum")
			}

			if Debug {
				log.Println(fmt.Sprintf("hashtree: %s: file %s => %s", rootDir, file.Name(), hex.EncodeToString(h[:])))
			}

			node.Hash = h
		} else if file.IsDir() {
			if err := ReadDir(absolutePath, node); err != nil {
				return fmt.Errorf("unable to read node dir: %w", err)
			}
		}

		parent.Add(node)

		if _, err := hasher.Write(node.Hash[:]); err != nil {
			return fmt.Errorf("unable to hash node: %w", err)
		}

	}

	// purge files, which are absent
	sort.Strings(currentFiles)
	childCopy := append([]*Node{}, parent.Children...)
	for _, child := range childCopy {
		idx := sort.SearchStrings(currentFiles, child.Name)
		if idx >= len(currentFiles) || currentFiles[idx] != child.Name {
			parent.Remove(child.Name)
			if Debug {
				log.Println(fmt.Sprintf("hashtree: %s: found extra child, removing: %s", rootDir, child.Name))
			}
		}
	}

	// update merkle root hash
	tmp := hasher.Sum(nil)
	copy(parent.Hash[:], tmp)

	if Debug {
		log.Println(fmt.Sprintf("hashtree: dir %s => %s", rootDir, hex.EncodeToString(parent.Hash[:])))
	}

	return nil
}

// fileIgnored currently only returns false for dotted names (. prefix).
func fileIgnored(name string) bool {
	if len(name) == 0 || name[0] == '.' {
		return true
	}

	return false
}
