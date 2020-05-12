package flatend

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lithdew/flatend/httputil"
	"io"
	"net/http"
)

const (
	HeaderContentType        = "Content-Type"
	HeaderContentTypeOptions = "X-Content-Type-Options"
)

var (
	_ Handler = (*ContentType)(nil)
	_ Handler = (*ContentLength)(nil)
	_ Handler = (*ContentDecode)(nil)
	_ Handler = (*ContentEncode)(nil)
	_ Handler = (*ExecuteSQL)(nil)
)

type Context struct {
	Config *Config
	In     Values
	Out    Values
}

type Handler interface {
	Serve(ctx *Context, w http.ResponseWriter, r *http.Request) error
}

type ContentType struct{}

func (h *ContentType) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	accept := httputil.NegotiateContentType(r, ctx.Config.CodecTypes, ctx.Config.DefaultCodec)

	if _, available := ctx.Config.Codecs[accept]; !available {
		return &Error{
			Status: http.StatusNotAcceptable,
			Err:    fmt.Errorf("only able to accept %q", ctx.Config.CodecTypes),
		}
	}

	w.Header().Set(HeaderContentType, accept)

	return nil
}

type ContentLength struct {
	Min int64
	Max int64
}

func (c *ContentLength) Serve(_ *Context, w http.ResponseWriter, r *http.Request) error {
	switch {
	case r.ContentLength < c.Min:
		return &Error{
			Status: http.StatusBadRequest,
			Err:    fmt.Errorf("payload too small: expected %d byte(s) min, got %d byte(s)", c.Min, r.ContentLength),
		}
	case r.ContentLength > c.Max:
		return &Error{
			Status: http.StatusRequestEntityTooLarge,
			Err:    fmt.Errorf("payload too large: expected %d byte(s) max, got %d byte(s)", c.Max, r.ContentLength),
		}
	}

	return nil
}

type ContentDecode struct{}

func (h *ContentDecode) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	codec := ctx.Config.Codecs[r.Header.Get(HeaderContentType)]

	err := getHeaderParams(r, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return &Error{Status: http.StatusBadRequest, Err: err}
	}

	err = getQueryParams(r, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return &Error{Status: http.StatusBadRequest, Err: err}
	}

	err = getBodyParams(r, codec, ctx.In)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return &Error{Status: http.StatusBadRequest, Err: err}
	}

	return nil
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

type ContentEncode struct{}

func (h *ContentEncode) Serve(ctx *Context, w http.ResponseWriter, _ *http.Request) error {
	ctx.Out = ctx.In

	codec := ctx.Config.Codecs[w.Header().Get(HeaderContentType)]

	if codec == nil {
		return &Error{
			Status: http.StatusNotAcceptable,
			Err:    fmt.Errorf("only able to accept %q", ctx.Config.CodecTypes),
		}
	}

	buf, err := codec.Encode(ctx.Out)
	if err != nil && !errors.Is(err, http.ErrBodyNotAllowed) {
		return &Error{
			Status: http.StatusInternalServerError,
			Err:    err,
		}
	}

	w.Write(buf)

	return nil
}

type ExecuteSQL struct {
	Stmt *sql.Stmt
}

func (h *ExecuteSQL) Serve(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	rows, err := h.Stmt.Query()
	if err != nil {
		return &Error{Status: http.StatusInternalServerError, Err: fmt.Errorf("failed to execute query: %w", err)}
	}
	_ = rows
	return nil
}
