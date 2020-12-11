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
	"encoding/json"
	"github.com/golangee/log"
	"github.com/golangee/log/ecs"
	"net/http"
	"reflect"
)

// writeJson is a helper and just tries to serialize the response as json.
func writeJson(w http.ResponseWriter, r *http.Request, obj interface{}) {
	buf, err := json.Marshal(obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.FromContext(r.Context()).Print(ecs.Msg("failed to marshal json response"), ecs.ErrMsg(err), log.V("type", reflect.TypeOf(obj).String()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(buf); err != nil {
		log.FromContext(r.Context()).Print(ecs.Msg("failed to write Json response"), ecs.ErrMsg(err))
	}
}
