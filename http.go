package flatend

import (
	"net/http"
)

type Server struct {
	srv    *http.Server
	routes map[string]http.HandlerFunc
}
