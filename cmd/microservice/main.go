package main

import (
	"github.com/lithdew/flatend"
	"strconv"
	"sync/atomic"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var counter uint64 = 0

func handleAllTodos(_ *flatend.Context) []byte {
	return strconv.AppendUint(nil, atomic.AddUint64(&counter, 1), 10)
}

func handleGetTodos(ctx *flatend.Context) []byte {
	return ctx.Body()
}

func main() {
	service := &flatend.Service{Addr: "127.0.0.1:9000"}
	service.Register("all_todos", handleAllTodos)
	service.Register("get_todos", handleGetTodos)
	check(service.Start())
}
