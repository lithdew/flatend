package flatend

import (
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"net"
	"sync"
	"time"
)

type HTTP struct {
	srv *fasthttp.Server

	wg sync.WaitGroup
	ch chan error

	mu     sync.RWMutex
	routes map[string]func()
}

func NewHTTP() *HTTP {
	s := &HTTP{ch: make(chan error, 1)}
	return s
}

func (s *HTTP) Load(cfg *Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.routes = make(map[string]func())

	for route, fn := range cfg.handlers {
		s.routes[route] = fn
	}
}

func (s *HTTP) Listen(ln net.Listener) error {
	if ln == nil {
		return errors.New("net.Listener must not be nil")
	}

	s.wg.Add(1)

	s.srv = &fasthttp.Server{
		Name: "flatend",
		Handler: func(ctx *fasthttp.RequestCtx) {
			s.mu.RLock()
			defer s.mu.RUnlock()

			handler, exists := s.routes[string(ctx.Path())]
			if !exists {
				return
			}

			handler()
		},
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	go func() {
		defer s.wg.Done()
		s.ch <- s.srv.Serve(ln)
	}()

	return nil
}

func (s *HTTP) Close() error {
	if err := s.srv.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown http server: %w", err)
	}

	s.wg.Wait()

	if err := <-s.ch; err != nil {
		return fmt.Errorf("failed to close http server listener: %w", err)
	}

	return nil
}
