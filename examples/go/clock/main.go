package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/lithdew/flatend"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func clock(ctx *flatend.Context) {
	latest := time.Now()
	ours := latest.Format(time.Stamp)

	timestamp, err := ioutil.ReadAll(ctx.Body)
	if err != nil {
		return
	}

	fmt.Printf("Got someone's time ('%s')! Sent back ours ('%s').\n", string(timestamp), ours)

	ctx.Write([]byte(ours))
}

func main() {
	flag.Parse()

	node := &flatend.Node{
		SecretKey: flatend.GenerateSecretKey(),
		Services: map[string]flatend.Handler{
			"clock": clock,
		},
	}
	defer node.Shutdown()

	if flag.Arg(0) == "" {
		node.PublicAddr = "127.0.0.1:9000"
		check(node.Start())
	} else {
		check(node.Start("127.0.0.1:9000"))
	}

	exit := make(chan struct{})
	defer close(exit)

	go func() {
		for i := 0; ; i++ {
			select {
			case <-exit:
				return
			case <-time.After(time.Duration(rand.Intn(1000)) * time.Millisecond):
				timestamp := time.Now().Format(time.Stamp)
				body := ioutil.NopCloser(strings.NewReader(timestamp))

				stream, err := node.Push([]string{"clock"}, nil, body)
				if err != nil {
					if strings.Contains(err.Error(), "no nodes were able to process") {
						continue
					}
					if errors.Is(err, io.EOF) {
						continue
					}
					check(err)
				}

				fmt.Printf("[%d] Asked someone for their current time. Ours is '%s'.\n", i, timestamp)

				res, err := ioutil.ReadAll(stream.Reader)
				if !errors.Is(err, io.ErrClosedPipe) {
					check(err)
				}

				fmt.Printf("[%d] Got a response! Their current time is: '%s'\n", i, string(res))
			}
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
