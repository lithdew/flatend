package flatend

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

var _ http.Handler = (*Server)(nil)

type Config struct {
	Codecs       map[string]*Codec
	DefaultCodec string

	Handlers []Handler
}

func NewDefaultConfig() *Config {
	return &Config{
		Codecs:       Codecs,
		DefaultCodec: "json",

		Handlers: []Handler{
			&ContentType{},
			&ContentLength{Max: 10 * 1024 * 1024},
			&ContentDecode{},
		},
	}
}

type Server struct {
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration

	MaxHeaderBytes int

	TLS    *tls.Config
	Config *Config

	once   sync.Once
	http   *http.Server
	routes map[string]http.HandlerFunc
}

func (s *Server) init() {
	s.http = &http.Server{
		Handler: s,

		TLSConfig: s.TLS,

		ReadTimeout:       s.ReadTimeout,
		WriteTimeout:      s.WriteTimeout,
		IdleTimeout:       s.IdleTimeout,
		ReadHeaderTimeout: s.ReadHeaderTimeout,

		MaxHeaderBytes: s.MaxHeaderBytes,
	}

	if s.Config == nil {
		s.Config = NewDefaultConfig()
	}
}

func (s *Server) Serve(ln net.Listener) error {
	s.once.Do(s.init)
	return s.http.Serve(ln)
}

func (s *Server) ServeTLS(addr, certFile, keyFile string) error {
	s.once.Do(s.init)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.http.ServeTLS(ln, certFile, keyFile)
}

func (s *Server) Shutdown() error {
	s.once.Do(s.init)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return s.http.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	in := acquireValues()
	out := acquireValues()

	defer releaseValues(in)
	defer releaseValues(out)

	ctx := &Context{
		Config: s.Config,
		In:     in,
		Out:    out,
	}

	for _, handler := range s.Config.Handlers {
		handler.Serve(ctx, w, r)
	}
}
