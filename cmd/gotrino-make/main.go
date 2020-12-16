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

package main

import (
	"flag"
	"fmt"
	"github.com/golangee/gotrino-make/internal/app"
	"github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/gotool"
	"github.com/golangee/gotrino-make/internal/hashtree"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current working directory: %w", err)
	}

	host := flag.String("host", "localhost", "the host to bind on.")
	port := flag.Int("port", 8080, "the port to bind to.")
	wwwDir := flag.String("www", "", "the directory which contains the wasm module.")
	buildDir := flag.String("dir", "", "the target build directory")
	debug := flag.Bool("debug", false, "enable debug logging output")

	flag.Parse()

	builder.Debug = *debug
	hashtree.Debug = *debug
	gotool.Debug = *debug

	if *buildDir == "" {
		*buildDir = filepath.Join(os.TempDir(), "gotrino-livebuilder")
	}

	if strings.HasPrefix(*buildDir, ".") {
		*buildDir = filepath.Join(cwd, *buildDir)
	}

	if *wwwDir == "" || strings.HasPrefix(*wwwDir, ".") {
		*wwwDir = filepath.Join(cwd, *wwwDir)
	}

	if len(flag.Args()) == 1 {
		action := flag.Args()[len(flag.Args())-1]

		a, err := app.NewApplication(*host, *port, *wwwDir, *buildDir, *debug)
		if err != nil {
			return err
		}

		defer a.Close()

		switch action {
		case "serve":
			return a.Run()
		case "build":
		// already builds on construction
		case "clean":
			if err := os.RemoveAll(*buildDir); err != nil {
				log.Fatalf("cannot clean build dir: %w", err)
			}
		default:
			log.Fatalf("invalid action: %s", action)
		}

	} else {
		log.Fatalf("you must provide an action: serve | build | clean")
	}

	return nil
}
