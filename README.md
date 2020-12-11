# gotrino-make

Package gotrino-make/cmd/gotrino-make contains a program to build or serve with hot deployment a gotrino wasm project.

## usage and example

```bash
# install into ~/go/bin
GOPROXY=direct go get -u github.com/golangee/gotrino-make/cmd/gotrino-make

# create a new wasm project
mkdir -p ~/tmp/gotrino-test
cd ~tmp/gotrino-test
go mod init mycompany.com/myproject

# by convention, there must be a main package in cmd/wasm as an entry point
cat > cmd/wasm/main.go << EOL
package main

import (
	"github.com/golangee/dom"
	"github.com/golangee/gotrino"
	"github.com/golangee/gotrino-tailwind/button"
)

func main() {
    // start your actual application, better refactor it into a call to internal/app
    run()
    
    // keep wasm alive, e.g. for click listeners
	select {} 
}

func run(){
    // show error, if run fails with panic
    defer dom.GlobalPanicHandler() 
    
    // render some component or html
    gotrino.RenderBody(button.NewTextButton("hello world",func(){
        panic("not yet implemented")
      })
    )
}

EOL

# make nice
gofmt -w cmd/wasm/main.go

# build for productive deployment
gotrino-make -dir=./dist

# serve and rebuild automatically. Use 0.0.0.0 to be able to connect with your smartphone (security concern). 
# Now change your file and note that the browser will automatically reload the page.
gotrino-make -host=0.0.0.0 -www=. serve
```