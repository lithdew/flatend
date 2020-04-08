package main

import (
	"fmt"
	"github.com/lithdew/flatend"
	flag "github.com/spf13/pflag"
	"github.com/valyala/fasthttp/reuseport"
	"net"
	"os"
	"os/signal"
	"strconv"
)

const program = `
http_get("hello_world")
`

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	bindHost string
	bindPort uint16
)

func main() {
	flag.StringVarP(&bindHost, "host", "h", "", "binding host")
	flag.Uint16VarP(&bindPort, "port", "p", 0, "binding port")

	flag.Parse()

	addr := net.JoinHostPort(bindHost, strconv.FormatUint(uint64(bindPort), 10))

	fmt.Printf("Bind address: %q\n", addr)

	ln, err := reuseport.Listen("tcp4", addr)
	check(err)

	_, err = flatend.LoadConfig(program)
	check(err)

	srv := flatend.NewHTTP()
	check(srv.Listen(ln))

	defer func() {
		check(srv.Close())
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	println()
}
