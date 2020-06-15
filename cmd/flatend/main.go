package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	jsoniter "github.com/json-iterator/go"
	"github.com/julienschmidt/httprouter"
	"github.com/lithdew/flatend"
	"github.com/spf13/pflag"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Config struct {
	HTTP []ConfigHTTP
}

func (c Config) Validate() error {
	for _, srv := range c.HTTP {
		err := srv.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type ConfigHTTP struct {
	Addr  string
	Addrs []string

	RedirectTrailingSlash *bool `toml:"redirect_trailing_slash"`
	RedirectFixedPath     *bool `toml:"redirect_fixed_path"`

	Timeout struct {
		Read       Duration
		ReadHeader Duration
		Idle       Duration
		Write      Duration
		Shutdown   Duration
	}

	Min struct {
		BodySize *int `toml:"body_size"`
	}

	Max struct {
		HeaderSize int  `toml:"header_size"`
		BodySize   *int `toml:"body_size"`
	}

	Routes []ConfigRoute
}

func (h ConfigHTTP) GetAddrs() []string {
	if h.Addr != "" {
		return []string{h.Addr}
	}
	return h.Addrs
}

func (h ConfigHTTP) Validate() error {
	if h.Addr != "" && h.Addrs != nil {
		return errors.New("'addr' and 'addrs' cannot both be non-nil at the same time")
	}

	for _, route := range h.Routes {
		err := route.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type ConfigRoute struct {
	Path     string
	Dispatch string
	Service  string
	Services []string

	Min struct {
		BodySize *int `toml:"body_size"`
	}

	Max struct {
		BodySize *int `toml:"body_size"`
	}
}

func (r ConfigRoute) Validate() error {
	if r.Service != "" && r.Services != nil {
		return errors.New("'service' and 'services' cannot both be non-nil at the same time")
	}

	fields := strings.Fields(r.Path)
	if len(fields) != 2 {
		return fmt.Errorf("invalid number of fields in route path '%s' (format: 'HTTP_METHOD /path/here')",
			r.Path)
	}

	method := strings.ToUpper(fields[0])
	_, exists := Methods[method]
	if !exists {
		return fmt.Errorf("unknown http method '%s'", method)
	}

	if len(fields[1]) < 1 || fields[1][0] != '/' {
		return fmt.Errorf("path must begin with '/' in path '%s'", fields[1])
	}

	_, err := url.ParseRequestURI(fields[1])
	if err != nil {
		return fmt.Errorf("invalid http path '%s': %w", fields[1], err)
	}

	return nil
}

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

			services := route.Services
			if route.Service != "" {
				services = append(services, route.Service)
			}

			handler := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
				headers := make(map[string]string)
				for key := range r.Header {
					headers[strings.ToLower(key)] = r.Header.Get(key)
				}

				for key := range r.URL.Query() {
					headers["query."+strings.ToLower(key)] = r.URL.Query().Get(key)
				}

				for _, param := range params {
					headers["params."+strings.ToLower(param.Key)] = param.Value
				}

				stream, err := node.Push(services, headers, r.Body)
				if err != nil {
					w.Write([]byte(err.Error()))
					return
				}

				for name, val := range stream.Header.Headers {
					w.Header().Set(name, val)
				}

				_, _ = io.Copy(w, stream.Reader)
			}

			router.Handle(fields[0], fields[1], handler)
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
