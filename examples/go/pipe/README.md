# pipe

Whatever comes in must come out. Simple piping example: upload a file to POST /pipe.

```
$ flatend
2020/06/17 01:07:12 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 01:07:12 Listening for HTTP requests on '[::]:3000'.
2020/06/17 01:07:17 ??? has connected to you. Services: [pipe]

$ go run main.go
2020/06/17 01:07:17 You are now connected to 127.0.0.1:9000. Services: []

$ POST /pipe (1.6MiB GIF)
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "POST /pipe"
service = "pipe"
```

```go
package main

import (
	"github.com/lithdew/flatend"
	"io"
	"os"
	"os/signal"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func pipe(ctx *flatend.Context) {
	_, _ = io.Copy(ctx, ctx.Body)
}

func main() {
	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"pipe": pipe,
		},
	}
	check(node.Start("127.0.0.1:9000"))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	node.Shutdown()
}
```