package httputil

import (
	"net/http"
	"strings"
)

// NegotiateContentType determines the best type of content to offer to a HTTP request given a slice of offered
// types. If no offers suit the request, an optional defaultOffer is provided.
func NegotiateContentType(r *http.Request, offers []string, defaultOffer string) string {
	bestOffer := defaultOffer
	bestQ := -1.0
	bestWild := 3

	specs := ParseAccept(r.Header, "Accept")

	for _, offer := range offers {
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
					bestOffer = offer
				}
			case strings.HasSuffix(spec.Value, "/*"):
				if strings.HasPrefix(offer, spec.Value[:len(spec.Value)-1]) &&
					(spec.Q > bestQ || bestWild > 1) {
					bestQ = spec.Q
					bestWild = 1
					bestOffer = offer
				}
			default:
				if spec.Value == offer &&
					(spec.Q > bestQ || bestWild > 0) {
					bestQ = spec.Q
					bestWild = 0
					bestOffer = offer
				}
			}
		}
	}
	return bestOffer
}
