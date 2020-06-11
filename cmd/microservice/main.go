package main

import (
	"github.com/lithdew/flatend"
	"github.com/lithdew/kademlia"
	"log"
	"net"
	"os"
	"os/signal"
)

func main() {
	Register("127.0.0.1:9000")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func Register(hub string) {
	_, priv, err := kademlia.GenerateKeys(nil)
	check(err)

	addr := "127.0.0.1:12000"

	node, err := flatend.NewNode(priv, addr)
	check(err)

	//counter := uint64(0)
	//
	//node.Handle(func(service string, buf []byte) []byte {
	//	return strconv.AppendUint(nil, atomic.AddUint64(&counter, 1), 10)
	//})

	handleGetTodos := func(ctx *flatend.Context) []byte {
		return ctx.Body()
	}

	node.Register("get_todos", handleGetTodos)

	ln, err := net.Listen("tcp", addr)
	check(err)

	go func() {
		check(node.Serve(ln))
	}()

	defer func() {
		node.Shutdown()
		check(ln.Close())
	}()

	check(node.Dial(hub))

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	log.Println("Done.")
}
