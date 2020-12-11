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

package fsnotify

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Watcher is a recursive fsnotify implementation.
type Watcher struct {
	fsw                *fsnotify.Watcher
	watchedDirectories []string
	watchedDirLock     sync.Mutex
	lastMod            int64
	lastModRebuild     int64
	dir                string
	logger             log.Logger
	onNotify           func()
}

// NewWatcher creates a new recursive fsnotify watch on all directories.
// If something is added or renamed, that watch tree is re-created.
// The given callback is not called for each change, but aggregated
// within a time window of second. It gets only called, as soon as
// all changes within a second have been applied, so an ever-changing
// directory will cause the callback to be never called.
func NewWatcher(root string, onNotifyCallback func()) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("no fsnotify support")
	}

	w := &Watcher{
		fsw:      watcher,
		dir:      root,
		onNotify: onNotifyCallback,
		logger:   log.NewLogger(ecs.Log("fsnotify"), ecs.URLPath(root)),
	}

	go func() {

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if log.Debug {
					w.logger.Print(ecs.Msg(event.String()))
				}

				if event.Op&fsnotify.Create == fsnotify.Create {
					if stat, err := os.Stat(event.Name); err == nil {
						if stat.IsDir() {
							w.notifyDelayedChange(event.Name, true)
							continue
						}
					}
				}

				w.notifyDelayedChange(event.Name, false)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				w.logger.Print(ecs.ErrMsg(err))
			}
		}
	}()

	if err := w.updateRecursiveWatch(root); err != nil {
		return nil, err
	}

	return w, nil
}

// notifyDelayedChange post-pones events, so that massive changes
// won't overload the system. It is fine to miss events, as long
// as we are still "dirty".
func (w *Watcher) notifyDelayedChange(fname string, rebuild bool) {
	atomic.StoreInt64(&w.lastMod, time.Now().UnixNano())
	if rebuild {
		atomic.StoreInt64(&w.lastModRebuild, 1)
	}

	w.checkLater()
}

func (w *Watcher) checkLater() {
	myGen := atomic.LoadInt64(&w.lastMod)

	time.AfterFunc(1*time.Second, func() {
		actualGen := atomic.LoadInt64(&w.lastMod)

		if myGen != actualGen {
			return
		}

		rebuild := atomic.LoadInt64(&w.lastModRebuild) == 1
		if rebuild {
			if err := w.updateRecursiveWatch(w.dir); err != nil {
				w.logger.Print(ecs.Msg("unable to update recursive watch"), ecs.ErrMsg(err))
			}
		}

		if w.onNotify != nil {
			w.onNotify()
		}
	})
}

// updateRecursiveWatch cleans up all ever registered file watches
// and attaches new watches to all non-hidden folders.
func (w *Watcher) updateRecursiveWatch(root string) error {
	w.watchedDirLock.Lock()
	defer w.watchedDirLock.Unlock()

	atomic.StoreInt64(&w.lastModRebuild, 0)

	for _, directory := range w.watchedDirectories {
		_ = w.fsw.Remove(directory)
	}

	w.watchedDirectories = w.watchedDirectories[:0]

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		w.watchedDirectories = append(w.watchedDirectories, path)
		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to search %s: %w", root, err)
	}

	for _, directory := range w.watchedDirectories {
		if err := w.fsw.Add(directory); err != nil {
			return fmt.Errorf("unable to attach watch %s: %w", directory, err)
		}
	}

	return nil
}

// Close removes all watchers.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}
