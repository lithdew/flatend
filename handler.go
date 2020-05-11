package flatend

import (
	"errors"
	"fmt"
	"github.com/lithdew/flatend/httputil"
	"io"
	"net/http"
	"strings"
)

const (
	HeaderContentType        = "Content-Type"
	HeaderContentTypeOptions = "X-Content-Type-Options"
)

var (
	_ http.Handler = (*ContextHandler)(nil)
	_ http.Handler = (*ContentTypeHandler)(nil)
	_ http.Handler = (*ContentLengthHandler)(nil)
	_ http.Handler = (*ContentDecodeHandler)(nil)

	_ ContextLoader = (*ContentTypeHandler)(nil)
	_ ContextLoader = (*ContentLengthHandler)(nil)
	_ ContextLoader = (*ContentDecodeHandler)(nil)
)

type ContextLoader interface {
	LoadContext(ctx *Context)
}

type Context struct {
	Codecs           map[string]*Codec
	DefaultCodec     string
	MinContentLength int64
	MaxContentLength int64

	In  Values
	Out Values
}

type ContextHandler struct {
	Codecs           map[string]*Codec
	DefaultCodec     string
	MinContentLength int64
	MaxContentLength int64

	Handlers []interface {
		http.Handler
		ContextLoader
	}
}

func (h *ContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	in := acquireValues()
	out := acquireValues()

	defer releaseValues(in)
	defer releaseValues(out)

	ctx := &Context{
		Codecs:           h.Codecs,
		DefaultCodec:     h.DefaultCodec,
		MinContentLength: h.MinContentLength,
		MaxContentLength: h.MaxContentLength,

		In:  in,
		Out: out,
	}

	for _, handler := range h.Handlers {
		handler.LoadContext(ctx)
		handler.ServeHTTP(w, r)
	}
}

type ContentTypeHandler struct {
	Codecs       map[string]*Codec
	DefaultCodec string
}

func (h *ContentTypeHandler) LoadContext(ctx *Context) {
	h.Codecs = ctx.Codecs
	h.DefaultCodec = ctx.DefaultCodec
}

func (h *ContentTypeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	accepts := NegotiateCodec(r, h.Codecs, h.DefaultCodec)

	codec, available := h.Codecs[accepts]
	if !available {
		httpError(
			w, codec,
			http.StatusNotAcceptable,
			fmt.Errorf("only able to accept %v", h.Codecs),
		)
		return
	}

	w.Header().Set(HeaderContentType, accepts)
}

func NegotiateCodec(r *http.Request, codecs map[string]*Codec, defaultCodec string) string {
	specs := httputil.ParseAccept(r.Header, "Accept")
	bestCodec, bestQ, bestWild := defaultCodec, -1.0, 3

	for codec := range codecs {
		for _, spec := range specs {
			switch {
			case spec.Q == 0.0:
				// ignore
			case spec.Q < bestQ:
				// better match found
			case spec.Value == "*/*":
				if spec.Q > bestQ || bestWild > 2 {
					bestQ = spec.Q
					bestWild = 2
					bestCodec = codec
				}
			case strings.HasSuffix(spec.Value, "/*"):
				if strings.HasPrefix(codec, spec.Value[:len(spec.Value)-1]) && (spec.Q > bestQ || bestWild > 1) {
					bestQ = spec.Q
					bestWild = 1
					bestCodec = codec
				}
			default:
				if spec.Value == codec && (spec.Q > bestQ || bestWild > 0) {
					bestQ = spec.Q
					bestWild = 0
					bestCodec = codec
				}
			}
		}
	}
	return bestCodec
}

type ContentLengthHandler struct {
	Codecs map[string]*Codec
	Min    int64
	Max    int64
}

func (h *ContentLengthHandler) LoadContext(ctx *Context) {
	h.Codecs = ctx.Codecs
	h.Min = ctx.MinContentLength
	h.Max = ctx.MaxContentLength
}

func (h *ContentLengthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var codec *Codec
	if h.Codecs != nil {
		codec = h.Codecs[w.Header().Get(HeaderContentType)]
	}

	switch {
	case r.ContentLength == 0:
		httpError(w, codec, http.StatusNoContent, nil)
	case r.ContentLength < h.Max:
		httpError(
			w, codec,
			http.StatusBadRequest,
			fmt.Errorf("payload too small: expected %d byte(s) min, got %d byte(s)", h.Min, r.ContentLength),
		)
	case r.ContentLength > h.Max:
		httpError(
			w, codec,
			http.StatusRequestEntityTooLarge,
			fmt.Errorf("payload too large: expected %d byte(s) max, got %d byte(s)", h.Max, r.ContentLength),
		)
	}
}

type ContentDecodeHandler struct {
	Codecs map[string]*Codec
	Values Values
}

func (h *ContentDecodeHandler) LoadContext(ctx *Context) {
	h.Codecs = ctx.Codecs
	h.Values = ctx.In
}

func (h *ContentDecodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var codec *Codec
	if h.Codecs != nil {
		codec = h.Codecs[w.Header().Get(HeaderContentType)]
	}

	err := h.getHeaderParams(r)
	if !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}

	err = h.getQueryParams(r)
	if !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}

	err = h.getBodyParams(r, codec)
	if !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}
}

func (h *ContentDecodeHandler) getHeaderParams(r *http.Request) error {
	for k, v := range r.Header {
		switch len(v) {
		case 0:
		case 1:
			h.Values[k] = v[0]
		default:
			h.Values[k] = v
		}
	}
	return nil
}

func (h *ContentDecodeHandler) getQueryParams(r *http.Request) error {
	for k, v := range r.URL.Query() {
		switch len(v) {
		case 0:
		case 1:
			h.Values[k] = v[0]
		default:
			h.Values[k] = v
		}
	}
	return nil
}

func (h *ContentDecodeHandler) getBodyParams(r *http.Request, codec *Codec) error {
	if r.Method == http.MethodTrace {
		return fmt.Errorf("no body is expected: %w", http.ErrBodyNotAllowed)
	}

	buf := make([]byte, r.ContentLength)

	n, err := io.ReadFull(r.Body, buf)
	if err == nil && int64(n) != r.ContentLength {
		err = fmt.Errorf(
			"read only %d byte(s), expected exactly %d byte(s): %w",
			n, r.ContentLength, io.ErrUnexpectedEOF,
		)
	}
	if err != nil {
		return err
	}

	if codec == nil {
		return errors.New("unable to negotiate a codec to decode the body of this request")
	}

	err = codec.Decode(buf, h.Values)
	if err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	return nil
}
