# clock

A demo of peer discovery and bidirectional streaming, and Flatend by itself without its built-in HTTP server.

Run `go run main.go` on one terminal. Run `go run main.go clock` on several other terminals.

Watch nodes randomly query and respond to each others requests regarding their current systems time.

```
[terminal 1] $ go run main.go
2020/06/18 00:06:56 Listening for Flatend nodes on '127.0.0.1:9000'.
2020/06/18 00:06:57 [::]:44369 has connected. Services: [clock]
Got someone's time ('Jun 18 00:06:57')! Sent back ours ('Jun 18 00:06:57').
2020/06/18 00:06:57 [::]:45309 has connected. Services: [clock]
Got someone's time ('Jun 18 00:06:57')! Sent back ours ('Jun 18 00:06:57').
[1] Asked someone for their current time. Ours is 'Jun 18 00:06:57'.
[1] Got a response! Their current time is: 'Jun 18 00:06:57'.
[2] Asked someone for their current time. Ours is 'Jun 18 00:06:57'.
[2] Got a response! Their current time is: 'Jun 18 00:06:57'.
[3] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.

[terminal 2] $ go run main.go clock
2020/06/18 00:06:57 Listening for Flatend nodes on '[::]:44369'.go clock
2020/06/18 00:06:57 You are now connected to 127.0.0.1:9000. Services: [clock]
2020/06/18 00:06:57 Re-probed 127.0.0.1:9000. Services: [clock]
2020/06/18 00:06:57 Discovered 0 peer(s).
[0] Asked someone for their current time. Ours is 'Jun 18 00:06:57'.
[0] Got a response! Their current time is: 'Jun 18 00:06:57'.
2020/06/18 00:06:57 [::]:45309 has connected. Services: [clock]
Got someone's time ('Jun 18 00:06:57')! Sent back ours ('Jun 18 00:06:57').
[1] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.
[1] Got a response! Their current time is: 'Jun 18 00:06:58'.
[2] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.
[2] Got a response! Their current time is: 'Jun 18 00:06:58'.
Got someone's time ('Jun 18 00:06:58')! Sent back ours ('Jun 18 00:06:58').
Got someone's time ('Jun 18 00:06:58')! Sent back ours ('Jun 18 00:06:58').
[3] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.
[3] Got a response! Their current time is: 'Jun 18 00:06:58'.

[terminal 3] $ go run main.go clock
2020/06/18 00:06:57 Listening for Flatend nodes on '[::]:45309'.go clock
2020/06/18 00:06:57 You are now connected to 127.0.0.1:9000. Services: [clock]
2020/06/18 00:06:57 Re-probed 127.0.0.1:9000. Services: [clock]
2020/06/18 00:06:57 You are now connected to [::]:44369. Services: [clock]
2020/06/18 00:06:57 Discovered 1 peer(s).
[0] Asked someone for their current time. Ours is 'Jun 18 00:06:57'.
[0] Got a response! Their current time is: 'Jun 18 00:06:57'.
Got someone's time ('Jun 18 00:06:57')! Sent back ours ('Jun 18 00:06:57').
Got someone's time ('Jun 18 00:06:58')! Sent back ours ('Jun 18 00:06:58').
Got someone's time ('Jun 18 00:06:58')! Sent back ours ('Jun 18 00:06:58').
[1] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.
[1] Got a response! Their current time is: 'Jun 18 00:06:58'.
[2] Asked someone for their current time. Ours is 'Jun 18 00:06:58'.
[2] Got a response! Their current time is: 'Jun 18 00:06:58'.
Got someone's time ('Jun 18 00:06:58')! Sent back ours ('Jun 18 00:06:58').
```

```go
package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/lithdew/flatend"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func clock(ctx *flatend.Context) {
	latest := time.Now()
	ours := latest.Format(time.Stamp)

	timestamp, err := ioutil.ReadAll(ctx.Body)
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return
	}

	fmt.Printf("Got someone's time ('%s')! Sent back ours ('%s').\n", string(timestamp), ours)

	ctx.Write([]byte(ours))
}

func main() {
	flag.Parse()

	node := &flatend.Node{
		SecretKey: flatend.GenerateSecretKey(),
		Services: map[string]flatend.Handler{
			"clock": clock,
		},
	}
	defer node.Shutdown()

	if flag.Arg(0) == "" {
		node.PublicAddr = "127.0.0.1:9000"
		check(node.Start())
	} else {
		check(node.Start("127.0.0.1:9000"))
	}

	exit := make(chan struct{})
	defer close(exit)

	go func() {
		for i := 0; ; i++{
			select {
			case <-exit:
				return
			case <-time.After(time.Duration(rand.Intn(1000)) * time.Millisecond):
				timestamp := time.Now().Format(time.Stamp)
				body := ioutil.NopCloser(strings.NewReader(timestamp))

				stream, err := node.Push([]string{"clock"}, nil, body)
				if err != nil {
					if strings.Contains(err.Error(), "no nodes were able to process") {
						continue
					}
					if errors.Is(err, io.EOF) {
						continue
					}
					check(err)
				}

				fmt.Printf("[%d] Asked someone for their current time. Ours is '%s'.\n", i, timestamp)

				res, err := ioutil.ReadAll(stream.Reader)
				if !errors.Is(err, io.ErrClosedPipe) {
					check(err)
				}

				fmt.Printf("[%d] Got a response! Their current time is: '%s'\n", i, string(res))
			}
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
```