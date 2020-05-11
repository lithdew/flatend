package flatend

import (
	"github.com/lithdew/bytesutil"
	"io"
	"net/http"
)

func httpError(w http.ResponseWriter, codec *Codec, status int, err error) error {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return io.EOF
	}

	values := acquireValues()
	defer releaseValues(values)

	values["error"] = err.Error()
	values["status"] = status

	var (
		encoded    []byte
		encodedErr error
	)

	if codec != nil {
		encoded, encodedErr = codec.Encode(values)
	}

	if codec == nil || encodedErr != nil {
		w.Header().Set(HeaderContentType, "text/plain; charset=utf-8")
		w.Header().Set(HeaderContentTypeOptions, "nosniff")
		w.WriteHeader(status)

		encoded = bytesutil.Slice(err.Error())
		w.Write(encoded)

		return err
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	w.Write(encoded)

	return err
}
