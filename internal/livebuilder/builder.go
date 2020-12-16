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
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/fsnotify"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"sync"
)

// Builder provides an automatic live builder which rebuilds an idiomatic golangee wasm project any time it
// recognizes a change.
type Builder struct {
	logger         log.Logger
	srcDir, dstDir string
	buildLock      sync.Mutex
	watcher        *fsnotify.Watcher
	buildFinished  func(hash string)
	opts           builder.Options
	project        *builder.Project
}

func NewBuilder(dstDir, srcDir string, buildFinished func(hash string), opts builder.Options) (*Builder, error) {
	b := &Builder{
		srcDir:        srcDir,
		dstDir:        dstDir,
		buildFinished: buildFinished,
		opts:          opts,
	}

	prj, err := builder.NewProject(dstDir, srcDir)
	if err != nil {
		return nil, fmt.Errorf("unable to setup project builder: %w", err)
	}

	b.project = prj
	b.logger = log.NewLogger(ecs.Log("livebuilder"))

	w, err := fsnotify.NewWatcher(srcDir, func() {
		if err := b.Build(); err != nil {
			b.logger.Println("failed to build", err)
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

	if b.opts.Debug {
		b.logger.Println("building started...")
	}

	hash, err := b.project.Build(b.opts)
	if err != nil {
		var buildErr builder.CompileErr
		if !errors.As(err, &buildErr) {
			return fmt.Errorf("unable to build wasm project: %w", err)
		}
	}

	if b.buildFinished != nil {
		b.buildFinished(hex.EncodeToString(hash[:]))
	}

	return err
}

func (b *Builder) Close() error {
	return b.watcher.Close()
}
