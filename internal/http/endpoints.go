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
	"net/http"
	"time"
)

func (s *Server) pollVersion(w http.ResponseWriter, r *http.Request) {
	log.FromContext(r.Context()).Println(ecs.Msg("registered long poll"))

	c := s.await()
	select {
	case version := <-c:
		type Version struct {
			Version string
		}
		log.FromContext(r.Context()).Println(ecs.Msg("returning " + version))
		writeJson(w, r, Version{Version: version})
	case _ = <-time.After(50 * time.Second):
		w.WriteHeader(http.StatusResetContent)
	}
}
