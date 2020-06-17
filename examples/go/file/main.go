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
