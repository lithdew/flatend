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
