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
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golangee/gotrino-make/internal/app"
	"github.com/golangee/gotrino-make/internal/builder"
	"github.com/golangee/gotrino-make/internal/deploy"
	"github.com/golangee/gotrino-make/internal/gotool"
	"github.com/golangee/gotrino-make/internal/hashtree"
	"io/ioutil"
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
	port := flag.Int("port", 8080, "the port to bind to for the serve mode.")
	wwwDir := flag.String("www", "", "the directory which contains the go wasm module to build.")
	buildDir := flag.String("dir", "", "the target output build directory. If empty a temporary folder is picked automatically.")
	debug := flag.Bool("debug", false, "enable debug logging output for gotrino-make.")
	templatePatterns := flag.String("templatePatterns", ".gohtml,.gocss,.gojs,.gojson,.goxml", "file extensions which should be processed as text/template with BuildInfo.")
	extra := flag.String("extra", "", "filename to a local json file, which contains extra BuildInfo values. Accessible in templates by {{.Extra}}")
	forceRefresh := flag.Bool("forceRefresh", false, "if set to true, all file hashes are always recalculated for each build instead of relying on ModTime.")
	goGenerate := flag.Bool("generate", false, "if set to true, 'go generate' is invoked everytime before building.")
	deployHost := flag.String("deploy-host", "", "the host to deploy to")
	deployPwd := flag.String("deploy-password", "", "the host password to deploy to")
	deployUser := flag.String("deploy-user", "", "the host user to deploy to")
	deploySrc := flag.String("deploy-src", "", "the local folder to upload")
	deployDst := flag.String("deploy-dst", ".", "the remote folder to upload")
	deployPrt := flag.Int("deploy-port", 22, "the remote port (e.g. ftp is usually 21 and sftp (SSH file Transfer Protocol) is 22)")
	//deploySkipVerify := flag.Bool("deploy-skip-verify", false, "accept invalid certificates")

	flag.Parse()

	builder.Debug = *debug
	hashtree.Debug = *debug
	gotool.Debug = *debug
	deploy.Debug = *debug

	action := ""
	if len(flag.Args()) == 1 {
		action = flag.Args()[len(flag.Args())-1]
	}

	opts := builder.Options{}
	opts.TemplatePatterns = strings.Split(*templatePatterns, ",")
	opts.Force = *forceRefresh
	opts.HotReload = action == "serve"
	opts.Debug = *debug
	opts.GoGenerate = *goGenerate

	if *extra != "" {
		buf, err := ioutil.ReadFile(*extra)
		if err != nil {
			return fmt.Errorf("unable to open extra file: %w", err)
		}

		err = json.Unmarshal(buf, &opts.Extra)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json from extra file: %w", err)
		}
	}

	if *buildDir == "" {
		*buildDir = filepath.Join(os.TempDir(), "gotrino-livebuilder")
	}

	if strings.HasPrefix(*buildDir, ".") {
		*buildDir = filepath.Join(cwd, *buildDir)
	}

	if strings.HasPrefix(*deploySrc, ".") {
		*deploySrc = filepath.Join(cwd, *deploySrc)
	}

	if *wwwDir == "" || strings.HasPrefix(*wwwDir, ".") {
		*wwwDir = filepath.Join(cwd, *wwwDir)
	}

	if len(flag.Args()) == 1 {

		switch action {
		case "deploy-ftp":
			/*err := ftp.Upload(*deployHost, *deployUser, *deployPwd, *deploySrc, *deployDst, *deployPrt, *debug, *deploySkipVerify)
			if err != nil {
				return fmt.Errorf("unable to deploy-ftp: %w", err)
			}*/
			panic("implement me")
		case "deploy-sftp":
			err := deploy.SyncSFTP(*deployDst, *deploySrc, *deployHost, *deployUser, *deployPwd, *deployPrt)
			if err != nil {
				return fmt.Errorf("unable to deploy-ftp: %w", err)
			}
		case "serve":
			a, err := app.NewApplication(*host, *port, *wwwDir, *buildDir, opts)
			if err != nil {
				return err
			}

			defer a.Close()

			return a.Run()
		case "build":
			a, err := app.NewApplication(*host, *port, *wwwDir, *buildDir, opts)
			if err != nil {
				return err
			}

			defer a.Close()
		case "clean":
			if err := os.RemoveAll(*buildDir); err != nil {
				log.Fatalf("cannot clean build dir: %w", err)
			}
		default:
			log.Fatalf("you must provide an action: serve | build | clean | deploy-sftp")
		}

	}

	return nil
}

func buildAndApp() {

}
