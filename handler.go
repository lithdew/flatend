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

type Handler interface {
	Serve(ctx *Context, w http.ResponseWriter, r *http.Request)
}

type Context struct {
	Config *Config
	In     Values
	Out    Values
}

type ContentType struct{}

func (h *ContentType) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) {
	accept := NegotiateCodec(r, ctx.Config.Codecs, ctx.Config.DefaultCodec)

	codec, available := ctx.Config.Codecs[accept]
	if !available {
		httpError(
			w, codec,
			http.StatusNotAcceptable,
			fmt.Errorf("only able to accept %v", ctx.Config.Codecs),
		)
		return
	}

	w.Header().Set(HeaderContentType, accept)
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

type ContentLength struct {
	Min int64
	Max int64
}

func (c *ContentLength) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) {
	var codec *Codec
	if ctx.Config.Codecs != nil {
		codec = ctx.Config.Codecs[w.Header().Get(HeaderContentType)]
	}

	switch {
	case r.ContentLength == 0:
		httpError(w, codec, http.StatusNoContent, nil)
	case r.ContentLength < c.Min:
		httpError(
			w, codec,
			http.StatusBadRequest,
			fmt.Errorf("payload too small: expected %d byte(s) min, got %d byte(s)", c.Min, r.ContentLength),
		)
	case r.ContentLength > c.Max:
		httpError(
			w, codec,
			http.StatusRequestEntityTooLarge,
			fmt.Errorf("payload too large: expected %d byte(s) max, got %d byte(s)", c.Max, r.ContentLength),
		)
	}
}

type ContentDecode struct{}

func (h *ContentDecode) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) {
	var codec *Codec
	if ctx.Config.Codecs != nil {
		codec = ctx.Config.Codecs[w.Header().Get(HeaderContentType)]
	}

	err := getHeaderParams(r, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}

	err = getQueryParams(r, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}

	err = getBodyParams(r, codec, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		httpError(w, codec, http.StatusBadRequest, err)
		return
	}
}

func getHeaderParams(r *http.Request, values map[string]interface{}) error {
	for k, v := range r.Header {
		switch len(v) {
		case 0:
		case 1:
			values[k] = v[0]
		default:
			values[k] = v
		}
	}
	return nil
}

func getQueryParams(r *http.Request, values map[string]interface{}) error {
	for k, v := range r.URL.Query() {
		switch len(v) {
		case 0:
		case 1:
			values[k] = v[0]
		default:
			values[k] = v
		}
	}
	return nil
}

func getBodyParams(r *http.Request, codec *Codec, values map[string]interface{}) error {
	if r.ContentLength == 0 || r.Method == http.MethodTrace {
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

	err = codec.Decode(buf, values)
	if err != nil {
		return fmt.Errorf("failed to decode body: %w", err)
	}

	return nil
}
