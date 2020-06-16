# hello_world

Create a service `hello_world` that replies with "Hello world!".

```
$ flatend
2020/06/17 00:44:34 Listening for microservices on '127.0.0.1:9000'.
2020/06/17 00:44:34 Listening for HTTP requests on '[::]:3000'.
2020/06/17 00:44:37 ??? has connected to you. Services: [hello_world]
2020/06/17 00:44:56 ??? has disconnected from you. Services: [hello_world]

$ http://localhost:3000/hello
no nodes were able to process your request for service(s): [hello_world]

$ go run main.go
2020/06/17 00:44:37 You are now connected to 127.0.0.1:9000. Services: []

$ http://localhost:3000/hello
Hello world!
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /hello"
service = "hello_world"
```

```go
package main

import (
	"github.com/lithdew/flatend"
	"os"
	"os/signal"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func helloWorld(ctx *flatend.Context) {
	ctx.WriteHeader("Content-Type", "text/plain; charset=utf-8")
	ctx.Write([]byte("Hello world!"))
}

func main() {
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"hello_world": helloWorld,
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
```