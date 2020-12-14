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

package app

import (
	"errors"
	"fmt"
	builder2 "github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/http"
	"github.com/golangee/gotrino-make/internal/livebuilder"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

type Application struct {
	server  *http.Server
	logger  log.Logger
	builder *livebuilder.Builder
	tmpDir  string
}

func NewApplication(host string, port int, wwwDir, buildDir string) (*Application, error) {
	tmpDir := buildDir
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, err
	}

	a := &Application{}
	a.initCloseListener()
	a.logger = log.NewLogger(ecs.Log("application"))

	a.logger.Println(ecs.Msg("build dir " + tmpDir))
	wwwBuildDir := filepath.Join(tmpDir, "www")

	if err := os.MkdirAll(wwwBuildDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("unable to create www build dir")
	}

	a.server = http.NewServer(log.WithFields(a.logger, ecs.Log("httpserver")), host, port, wwwBuildDir)
	builder, err := livebuilder.NewBuilder(wwwBuildDir, wwwDir, func(hash string) {
		a.server.NotifyChanged(hash)
	})
	if err != nil {
		return nil, err
	}
	a.builder = builder
	if err := a.builder.Build(); err != nil {
		buildErr := builder2.BuildErr{}
		if errors.As(err, &buildErr) {
			a.logger.Println(ecs.ErrMsg(err))
		} else {
			return nil, fmt.Errorf("unable to create initial build: %w", err)
		}
	}

	return a, nil
}

func (a *Application) initCloseListener() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		a.server.Stop()
	}()
}

func (a *Application) Run() error {
	defer func() {
		a.logger.Println(ecs.Msg("exiting"))
	}()

	return a.server.Run()
}

func (a *Application) Close() error {
	a.server.Stop()
	return os.RemoveAll(a.tmpDir)
}
