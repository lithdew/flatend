package flatend

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lithdew/flatend/httputil"
	"io"
	"net/http"
	"strconv"
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
	_ Handler = (*QuerySQL)(nil)
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

func (c *ContentLength) Serve(_ *Context, _ http.ResponseWriter, r *http.Request) error {
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

func (h *ContentDecode) Serve(ctx *Context, _ http.ResponseWriter, r *http.Request) error {
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
	if len(ctx.Out) == 0 {
		return nil
	}

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

func HandlePagination(ctx *Context, min, max int) (offset, limit int, ok bool, err error) {
	var (
		offsetProvided bool
		limitProvided  bool
	)

	switch val := ctx.In["offset"].(type) {
	case string:
		parsed, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return min, max, false, err
		}
		offset, offsetProvided = int(parsed), true
	case int:
		offset, offsetProvided = val, true
	}

	if offset < 0 || !offsetProvided {
		offset = 0
	}

	switch val := ctx.In["limit"].(type) {
	case string:
		parsed, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return min, max, false, err
		}
		limit, limitProvided = int(parsed), true
	case int:
		limit, limitProvided = val, true
	}

	switch {
	case limit < min || !limitProvided:
		limit = min
	case limit > max:
		limit = max
	}

	return offset, limit, offsetProvided || limitProvided, nil
}

type QuerySQL struct {
	MinNumRows int
	MaxNumRows int

	Stmt   *sql.Stmt
	Params []string
}

func (h *QuerySQL) Serve(ctx *Context, _ http.ResponseWriter, _ *http.Request) error {
	offset, limit, paginated, err := HandlePagination(ctx, h.MinNumRows, h.MaxNumRows)
	if err != nil {
		return &Error{
			Status: http.StatusBadRequest,
			Err:    fmt.Errorf("failed to parse pagination params: %w", err),
		}
	}

	params := acquireValueBuffer(len(h.Params))
	for _, param := range h.Params {
		params = append(params, ctx.In[param])
	}
	rows, err := h.Stmt.Query(params...)
	releaseValueBuffer(params)

	if err != nil {
		return &Error{
			Status: http.StatusInternalServerError,
			Err:    fmt.Errorf("failed to execute query: %w", err),
		}
	}

	defer rows.Close()

	keys, err := rows.Columns()
	if err != nil {
		return &Error{
			Status: http.StatusInternalServerError,
			Err:    fmt.Errorf("failed to fetch columns: %w", err),
		}
	}

	if len(keys) == 0 {
		return &Error{
			Status: http.StatusInternalServerError,
			Err:    errors.New("zero columns resultant from sql query"),
		}
	}

	vals := acquireValueBuffer(len(keys))
	defer releaseValueBuffer(vals)

	var results []map[string]interface{}

	for i := 0; rows.Next() && i < h.MaxNumRows; i++ {
		err := rows.Scan(vals...)
		if err != nil {
			return &Error{
				Status: http.StatusInternalServerError,
				Err:    fmt.Errorf("got an error while scanning: %w", err),
			}
		}

		result := make(map[string]interface{}, len(keys))
		for i := range keys {
			result[keys[i]] = vals[i]
		}

		results = append(results, result)
	}

	ctx.Out["status"] = http.StatusOK
	ctx.Out["results"] = results

	if paginated {
		ctx.Out["offset"] = offset
		ctx.Out["limit"] = limit
	}

	return nil
}
