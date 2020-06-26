# broadcast

A demo of broadcasting a message from one node to several nodes providing the service named `'chat'`.

```
[terminal 1] $ go run main.go -l :9000
2020/06/27 02:37:52 Listening for Flatend nodes on '[::]:9000'.
2020/06/27 02:37:53 0.0.0.0:9001 has connected. Services: [chat]
2020/06/27 02:37:56 0.0.0.0:9002 has connected. Services: [chat]
hello
Got 'world' from 0.0.0.0:9001!
Got 'test' from 0.0.0.0:9002!
2020/06/27 02:39:09 0.0.0.0:9002 has disconnected from you. Services: [chat]
2020/06/27 02:39:10 0.0.0.0:9001 has disconnected from you. Services: [chat]

[terminal 2] $ go run main.go -l :9001 :9000
2020/06/27 02:37:53 Listening for Flatend nodes on '[::]:9001'.
2020/06/27 02:37:53 You are now connected to 0.0.0.0:9000. Services: [chat]
2020/06/27 02:37:53 Re-probed 0.0.0.0:9000. Services: [chat]
2020/06/27 02:37:53 Discovered 0 peer(s).
2020/06/27 02:37:56 0.0.0.0:9002 has connected. Services: [chat]
Got 'hello' from 0.0.0.0:9000!
world
Got 'test' from 0.0.0.0:9002!
2020/06/27 02:39:09 0.0.0.0:9002 has disconnected from you. Services: [chat]

[terminal 3] $ go run main.go -l :9002 :9000
2020/06/27 02:37:56 Listening for Flatend nodes on '[::]:9002'.
2020/06/27 02:37:56 You are now connected to 0.0.0.0:9000. Services: [chat]
2020/06/27 02:37:56 Re-probed 0.0.0.0:9000. Services: [chat]
2020/06/27 02:37:56 You are now connected to 0.0.0.0:9001. Services: [chat]
2020/06/27 02:37:56 Discovered 1 peer(s).
Got 'hello' from 0.0.0.0:9000!
Got 'world' from 0.0.0.0:9001!
```

```go
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/lithdew/flatend"
	"io/ioutil"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "l", ":9000", "address to listen for peers on")
	flag.Parse()

	node := &flatend.Node{
		PublicAddr: listenAddr,
		BindAddrs:  []string{listenAddr},
		SecretKey:  flatend.GenerateSecretKey(),
		Services: map[string]flatend.Handler{
			"chat": func(ctx *flatend.Context) {
				buf, err := ioutil.ReadAll(ctx.Body)
				if err != nil {
					return
				}
				fmt.Printf("Got '%s' from %s:%d!\n", string(buf), ctx.ID.Host.String(), ctx.ID.Port)
			},
		},
	}
	defer node.Shutdown()

	check(node.Start(flag.Args()...))

	br := bufio.NewReader(os.Stdin)
	for {
		line, _, err := br.ReadLine()
		if err != nil {
			break
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		providers := node.ProvidersFor("chat")
		for _, provider := range providers {
			_, err := provider.Push([]string{"chat"}, nil, ioutil.NopCloser(bytes.NewReader(line)))
			if err != nil {
				fmt.Printf("Unable to broadcast to %s: %s\n", provider.Addr(), err)
			}
		}
	}
}
```
