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

package http

import (
	"context"
	"fmt"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"net/http"
	"time"
)

// Server is the rest service.
type Server struct {
	host     string
	port     int
	httpSrv  *http.Server
	dir      string
	logger   log.Logger
	awaiting chan chan string
}

// NewServer prepares a new Server instance.
func NewServer(logger log.Logger, host string, port int, dir string) *Server {
	s := &Server{
		host:     host,
		port:     port,
		logger:   logger,
		dir:      dir,
		awaiting: make(chan chan string, 10_000), // TODO await will stop working when capacity reached
	}

	return s
}

func (s *Server) NotifyChanged(version string) {
	// drain entire awaiting channels
	// TODO if clients re-connect to fast we have an endless loop here
	for {
		select {
		case c := <-s.awaiting:
			c <- version
		default:
			return
		}
	}
}

func (s *Server) await() chan string {
	c := make(chan string, 1)
	s.awaiting <- c
	return c
}

// Run launches the server
func (s *Server) Run() error {
	router := s.newRouter(s.dir)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.host, s.port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      router,
	}

	s.logger.Println(ecs.Msg("starting"), ecs.ServerAddress(s.host), ecs.ServerPort(s.port))
	err := s.httpSrv.ListenAndServe()

	if err == http.ErrServerClosed {
		s.logger.Println(ecs.Msg("stopped"))
		return nil
	}

	return err
}

// Stop signals the server to halt gracefully.
func (s *Server) Stop() {
	// normal if never run
	if s.httpSrv == nil {
		return
	}

	if err := s.httpSrv.Shutdown(context.Background()); err != nil {
		s.logger.Println(ecs.Msg("failed to shutdown"), ecs.ErrMsg(err))
	}
}
