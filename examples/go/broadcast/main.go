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
