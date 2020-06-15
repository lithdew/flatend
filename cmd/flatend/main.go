package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	jsoniter "github.com/json-iterator/go"
	"github.com/julienschmidt/httprouter"
	"github.com/lithdew/flatend"
	"github.com/spf13/pflag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var Methods = map[string]struct{}{
	http.MethodGet:    {},
	http.MethodPost:   {},
	http.MethodPut:    {},
	http.MethodDelete: {},
	http.MethodPatch:  {},
}

func main() {
	var configPath string
	var bindHost net.IP
	var bindPort uint16

	pflag.StringVarP(&configPath, "config", "c", "config.toml", "path to config file")
	pflag.IPVarP(&bindHost, "host", "h", net.ParseIP("0.0.0.0"), "bind host")
	pflag.Uint16VarP(&bindPort, "port", "p", 9000, "bind port")
	pflag.Parse()

	buf, err := ioutil.ReadFile(configPath)
	check(err)

	var cfg Config
	check(toml.Unmarshal(buf, &cfg))
	check(cfg.Validate())

	addr := flatend.Addr(bindHost, bindPort)

	node := &flatend.Node{PublicAddr: addr}
	check(node.Start())

	fmt.Printf("Listening for microservices on %s.\n", addr)

	defer node.Shutdown()

	for _, cfg := range cfg.HTTP {
		router := httprouter.New()

		if cfg.RedirectTrailingSlash != nil {
			router.RedirectTrailingSlash = *cfg.RedirectTrailingSlash
		}

		if cfg.RedirectFixedPath != nil {
			router.RedirectFixedPath = *cfg.RedirectFixedPath
		}

		for _, route := range cfg.Routes {
			fields := strings.Fields(route.Path)

			var handler http.Handler

			switch {
			case route.Static != "":
				static := route.Static

				info, err := os.Lstat(static)
				check(err)

				if info.IsDir() {
					if fields[1] == "/" {
						router.NotFound = http.FileServer(http.Dir(static))
					} else {
						if info.IsDir() && !strings.HasPrefix(fields[1], "/*filepath") {
							fields[1] = filepath.Join(fields[1], "/*filepath")
						}
						router.ServeFiles(fields[1], http.Dir(static))
					}
				} else {
					handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						http.ServeFile(w, r, static)
					})
				}
			case route.Service != "" || len(route.Services) > 0:
				services := route.Services
				if route.Service != "" {
					services = append(services, route.Service)
				}

				handler = HandleService(node, services)
			}

			if handler != nil {
				if route.NoCache {
					handler = NoCache(handler)
				}
				router.Handler(fields[0], fields[1], handler)
			}
		}

		srv := &http.Server{
			Handler:           router,
			ReadTimeout:       cfg.Timeout.Read.Duration,
			ReadHeaderTimeout: cfg.Timeout.ReadHeader.Duration,
			IdleTimeout:       cfg.Timeout.Idle.Duration,
			WriteTimeout:      cfg.Timeout.Write.Duration,
			MaxHeaderBytes:    cfg.Max.HeaderSize,
		}

		defer func() {
			check(srv.Close())
		}()

		for _, addr := range cfg.GetAddrs() {
			addr := addr
			go func() {
				ln, err := net.Listen("tcp", addr)
				check(err)

				err = srv.Serve(ln)
				if !errors.Is(err, http.ErrServerClosed) {
					check(err)
				}
			}()
		}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
