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
