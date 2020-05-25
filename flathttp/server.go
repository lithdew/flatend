package flathttp

import (
	"context"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var DefaultShutdownTimeout = 5 * time.Second

type Server struct {
	cfg Config

	router *httprouter.Router
	srv    *http.Server

	lns  map[string]net.Listener // all listeners
	wg   sync.WaitGroup          // all served goroutines
	lm   sync.Mutex              // lifecycle mutex
	em   sync.Mutex              // errors mutex
	errs []string                // errors
}

func NewServer(cfg *Config) *Server {
	if cfg == nil {
		cfg = &Config{}
	}
	srv := &Server{
		cfg: *cfg,
		lns: make(map[string]net.Listener),
	}
	return srv
}

func (s *Server) Start() error {
	s.lm.Lock()
	defer s.lm.Unlock()

	err := s.start()
	if err != nil {
		return s.stop()
	}
	return nil
}

func (s *Server) Stop() error {
	s.lm.Lock()
	defer s.lm.Unlock()

	return s.stop()
}

func (s *Server) Update(cfg *Config) {
	s.lm.Lock()
	defer s.lm.Unlock()

	if cfg == nil {
		cfg = &Config{}
	}
	s.cfg = *cfg
}

func (s *Server) start() error {
	if err := s.init(); err != nil {
		return err
	}

	for _, a := range s.cfg.addrs {
		err := s.listen(a)
		if err != nil {
			if errors.Is(err, ErrAlreadyListening) {
				continue
			}
			return err
		}
	}

	return nil
}

func (s *Server) listen(a Addr) error {
	srv := s.srv

	_, exists := s.lns[a.Addr]
	if exists {
		return fmt.Errorf("flathttp [%s]: %w", a.Addr, ErrAlreadyListening)
	}

	ln, err := net.Listen(a.Scheme, a.Host)
	if err != nil {
		return fmt.Errorf("flathttp: failed to start listener for %q: %w", a, err)
	}

	fmt.Printf("Listening %q.\n", ln.Addr())

	s.lns[a.Addr] = ln

	// Start serving the listener.

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := eof(srv.Serve(ln))
		if err != nil {
			err := fmt.Sprintf("unexpected error finishing serving %s (err=%q)", a.Addr, err)
			s.em.Lock()
			s.errs = append(s.errs, err)
			s.em.Unlock()
		}
	}()

	return nil
}

func (s *Server) stop() error {
	var errs []string

	// Gracefully shutdown the server.

	timeout := s.cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	err := s.srv.Shutdown(ctx)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to gracefully shutdown the server (err=%q)", err.Error()))
	}

	<-ctx.Done()

	for addr, ln := range s.lns {
		err := eof(ln.Close())
		if err != nil {
			err := fmt.Sprintf("[%s] failed to close listener (err=%q)", addr, err)
			errs = append(errs, err)
		}
	}

	if cancel != nil {
		cancel()
	}

	// Wait until all listeners are closed, and if errors rise up, include them in the result.

	s.wg.Wait()

	errs = append(errs, s.errs...)

	s.cleanup()

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf("flathttp: failed to stop server: %s", strings.Join(errs, ", "))
}

func (s *Server) init() error {
	if err := s.cfg.Parse(); err != nil {
		return err
	}

	s.router = httprouter.New()

	s.srv = &http.Server{
		Handler:           s.router,
		MaxHeaderBytes:    s.cfg.MaxHeaderBytes,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
	}

	return nil
}

func (s *Server) cleanup() {
	s.cfg.Reset()

	s.srv = nil
	s.errs = nil

	for addr := range s.lns {
		delete(s.lns, addr)
	}
}
