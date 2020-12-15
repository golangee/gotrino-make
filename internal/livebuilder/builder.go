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

package livebuilder

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/fsnotify"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"sync"
	"time"
)

// Builder provides an automatic live builder which rebuilds an idiomatic golangee wasm project any time it
// recognizes a change.
type Builder struct {
	logger         log.Logger
	lastBuildHash  []byte
	srcDir, dstDir string
	buildLock      sync.Mutex
	watcher        *fsnotify.Watcher
	buildFinished  func(hash string)
}

func NewBuilder(dstDir, srcDir string, buildFinished func(hash string)) (*Builder, error) {
	b := &Builder{
		srcDir:        srcDir,
		dstDir:        dstDir,
		buildFinished: buildFinished,
	}

	b.logger = log.NewLogger(ecs.Log("livebuilder"))

	w, err := fsnotify.NewWatcher(srcDir, func() {
		hash, err := builder.HashFileTree(srcDir)
		if err != nil {
			b.logger.Println(ecs.Msg("failed to calculate file tree hash"), ecs.ErrMsg(err))
			return
		}

		b.buildLock.Lock()
		hashCopy := append([]byte{}, b.lastBuildHash...)
		b.buildLock.Unlock()

		if bytes.Compare(hashCopy, hash) != 0 {
			if err := b.Build(); err != nil {
				b.logger.Println(ecs.Msg("failed to build project"), ecs.ErrMsg(err))
				return
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init fsnotify watcher: %w", err)
	}

	b.watcher = w
	b.logger.Println(ecs.Msg("start watching " + srcDir))

	return b, nil
}

// Build triggers a build now
func (b *Builder) Build() error {
	b.buildLock.Lock()
	defer b.buildLock.Unlock()

	start := time.Now()
	defer func() {
		b.logger.Println(ecs.Msg("build duration " + time.Now().Sub(start).String()))
	}()
	hash, err := builder.HashFileTree(b.srcDir)
	if err != nil {
		return err
	}
	b.logger.Println(ecs.Msg("building " + hex.EncodeToString(hash)))

	err = builder.BuildProject(b.srcDir, b.dstDir)
	if err != nil {
		var buildErr builder.CompileErr
		if !errors.As(err, &buildErr) {
			return fmt.Errorf("unable to build wasm project: %w", err)
		}
	}

	b.lastBuildHash = hash

	if b.buildFinished != nil {
		b.buildFinished(hex.EncodeToString(hash))
	}

	return err
}

func (b *Builder) Close() error {
	return b.watcher.Close()
}
