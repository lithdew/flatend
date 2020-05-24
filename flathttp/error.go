package flathttp

import (
	"errors"
	"io"
	"net"
	"net/http"
)

var ErrAlreadyListening = errors.New("already listening")

func eof(err error) error {
	var op *net.OpError

	switch {
	case errors.As(err, &op) && op.Err.Error() == "use of closed network connection":
		return nil
	case errors.Is(err, http.ErrServerClosed):
		return nil
	case errors.Is(err, io.EOF):
		return nil
	default:
		return err
	}
}
