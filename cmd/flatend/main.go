package main

import (
	"errors"
	"github.com/lithdew/flatend"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	ln, err := net.Listen("tcp", ":44444")
	check(err)
	defer wrap(ln.Close)

	db, err := dburl.Open("sqlite://:memory:")
	check(err)
	defer wrap(db.Close)

	stmt, err := db.Prepare("SELECT sqlite_version()")
	check(err)
	defer wrap(stmt.Close)

	srv := &flatend.Server{}
	srv.Register(
		&flatend.ContentType{},
		&flatend.ContentLength{Max: 10 * 1024 * 1024},
		&flatend.ContentDecode{},
		&flatend.QuerySQL{MaxNumRows: 1000, Stmt: stmt},
		&flatend.ContentEncode{},
	)

	go func() { check(srv.Serve(ln)) }()
	defer wrap(srv.Shutdown)

	log.Printf("Listening for connections on %q.", ln.Addr())

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}

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
