package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/caddyserver/certmagic"
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
			services := route.GetServices()

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
			case len(services) > 0:
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

		addrs := cfg.GetAddrs()

		if cfg.HTTPS {
			magic := certmagic.NewDefault()
			check(magic.ManageSync(addrs))

			acme := certmagic.NewACMEManager(magic, certmagic.DefaultACME)
			srv.Handler = acme.HTTPChallengeHandler(srv.Handler)

			for _, addr := range addrs {
				addr := addr
				go func() {
					ln, err := tls.Listen("tcp", net.JoinHostPort(addr, "443"), magic.TLSConfig())
					check(err)

					err = srv.Serve(ln)
					if !errors.Is(err, http.ErrServerClosed) {
						check(err)
					}
				}()
			}
		} else {
			for _, addr := range addrs {
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
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
