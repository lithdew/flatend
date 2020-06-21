# count

Count from zero.

```
$ flatend
2020/06/17 02:12:53 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/17 02:12:53 Listening for HTTP requests on '[::]:3000'.
2020/06/17 02:12:59 <anon> has connected to you. Services: [count]

$ go run main.go
2020/06/17 02:13:00 You are now connected to 127.0.0.1:9000. Services: []

$ http://localhost:3000
0

$ http://localhost:3000
1

$ http://localhost:3000
2
```

```toml
addr = "127.0.0.1:9000"

[[http]]
addr = ":3000"

[[http.routes]]
path = "GET /"
service = "count"
```

```go
package main

import (
	"github.com/lithdew/flatend"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	counter := uint64(0)

	node := &flatend.Node{
		Services: map[string]flatend.Handler{
			"count": func(ctx *flatend.Context) {
				current := atomic.AddUint64(&counter, 1) - 1
				ctx.Write(strconv.AppendUint(nil, current, 10))
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
