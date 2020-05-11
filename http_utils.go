package flatend

import (
	"github.com/lithdew/bytesutil"
	"net/http"
)

func httpError(w http.ResponseWriter, codec *Codec, status int, err error) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}

	values := acquireValues()
	defer releaseValues(values)

	values["error"] = err.Error()
	values["status"] = status

	if codec != nil {
		buf, err := codec.Encode(values)
		if err == nil {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(status)
			w.Write(buf)

			return
		}
	}

	buf := bytesutil.Slice(err.Error())
	w.Header().Set(HeaderContentType, "text/plain; charset=utf-8")
	w.Header().Set(HeaderContentTypeOptions, "nosniff")
	w.WriteHeader(status)
	w.Write(buf)
}
