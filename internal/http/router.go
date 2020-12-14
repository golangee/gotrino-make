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
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// newRouter creates a router and connects the endpoints with the given server and its methods.
func (s *Server) newRouter(fileServerDir string) *httprouter.Router {
	logMe := func(p string) string {
		s.logger.Println(ecs.Msg("registered endpoint"), log.V("url.path", p))
		return p
	}

	router := httprouter.New()
	router.HandlerFunc(http.MethodGet, logMe("/blub"), func(writer http.ResponseWriter, request *http.Request) {
		s.logger.Println(ecs.Msg("hello world"))
	})
	router.HandlerFunc(http.MethodGet, logMe("/api/v1/poll/version"), s.pollVersion)

	if fileServerDir != "" {
		router.NotFound = http.FileServer(http.Dir(logMe(fileServerDir)))
	}

	return router
}
