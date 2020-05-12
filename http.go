package flatend

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"github.com/lithdew/bytesutil"
	"net"
	"net/http"
	"sync"
	"time"
)

var _ http.Handler = (*Server)(nil)

type Server struct {
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	MaxHeaderBytes    int
	ReadHeaderTimeout time.Duration

	once sync.Once

	config *Config

	srv    *http.Server
	router *httprouter.Router
}

func (s *Server) init() {
	if s.config == nil {
		s.config = NewDefaultConfig()
	}

	s.router = httprouter.New()

	s.srv = &http.Server{
		Handler: s,

		IdleTimeout:  s.IdleTimeout,
		ReadTimeout:  s.ReadTimeout,
		WriteTimeout: s.WriteTimeout,

		MaxHeaderBytes:    s.MaxHeaderBytes,
		ReadHeaderTimeout: s.ReadHeaderTimeout,
	}
}

func (s *Server) Serve(ln net.Listener) error {
	s.once.Do(s.init)
	return s.srv.Serve(ln)
}

func (s *Server) ServeTLS(addr, certFile, keyFile string) error {
	s.once.Do(s.init)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.srv.ServeTLS(ln, certFile, keyFile)
}

func (s *Server) Shutdown() error {
	s.once.Do(s.init)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//s.router.ServeHTTP(w, r)

	in := acquireValues()
	out := acquireValues()

	defer releaseValues(in)
	defer releaseValues(out)

	ctx := &Context{
		Config: s.config,
		In:     in,
		Out:    out,
	}

	for _, handler := range s.config.Handlers {
		err := handler.Serve(ctx, w, r)
		if err != nil {
			s.writeError(w, err.(*Error))
			break
		}
	}
}

func (s *Server) writeError(w http.ResponseWriter, err *Error) {
	if err.Status == http.StatusNoContent {
		w.WriteHeader(err.Status)
		return
	}

	values := acquireValues()
	defer releaseValues(values)

	values["status"] = err.Status
	values["error"] = err.Error()

	var (
		buf      []byte
		codecErr error
	)

	codec := s.config.Codecs[w.Header().Get(HeaderContentType)]
	if codec != nil {
		buf, codecErr = codec.Encode(values)
	}

	if codec == nil || codecErr != nil {
		w.Header().Set(HeaderContentType, "text/plain; charset=utf-8")
		w.Header().Set(HeaderContentTypeOptions, "nosniff")
		w.WriteHeader(err.Status)

		buf = bytesutil.Slice(err.Error())
		w.Write(buf)

		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(err.Status)
	w.Write(buf)
}
