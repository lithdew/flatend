package main

import (
	"errors"
	"github.com/lithdew/flatend"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

func isEOF(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}

	if errors.Is(err, http.ErrServerClosed) {
		return true
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) && netErr.Err.Error() == "use of closed network connection" {
		return true
	}

	return false
}

func check(err error) {
	if err != nil && !isEOF(err) {
		log.Panic(err)
	}
}

func wrap(fn func() error) {
	check(fn())
}

func main() {
	ln, err := net.Listen("tcp", ":44444")
	check(err)
	defer wrap(ln.Close)

	srv := &flatend.Server{}
	go func() { check(srv.Serve(ln)) }()
	defer wrap(srv.Shutdown)

	log.Printf("Listening for connections on %q.", ln.Addr())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
