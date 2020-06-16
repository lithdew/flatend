# file

Here's something trippy: a service that responds with its own source code.

```
$ flatend
2020/06/17 02:36:30 Listening for microservices on '127.0.0.1:9000'.
2020/06/17 02:36:30 Listening for HTTP requests on '[::]:3000'.
2020/06/17 02:36:41 ??? has connected to you. Services: [file]

$ go run main.go 
2020/06/17 02:36:41 You are now connected to 127.0.0.1:9000. Services: []

$ http://localhost:3000/
package main

import (
	"github.com/lithdew/flatend"
	"io"
	"os"
	"os/signal"
)

func check(err error)  {
	if err != nil {
		panic(err)
	}
}

func main() {
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"file": func(ctx *flatend.Context) {
				f, err := os.Open("main.go")
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}
				_, _ = io.Copy(ctx, f)
				f.Close()
			},
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /"
service = "file"
```

```go
package main

import (
	"github.com/lithdew/flatend"
	"io"
	"os"
	"os/signal"
)

func check(err error)  {
	if err != nil {
		panic(err)
	}
}

func main() {
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"file": func(ctx *flatend.Context) {
				f, err := os.Open("main.go")
				if err != nil {
					ctx.Write([]byte(err.Error()))
					return
				}
				_, _ = io.Copy(ctx, f)
				f.Close()
			},
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
```