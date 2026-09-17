//revive:disable:package-comments
package httpserver

import (
	"net/http"
	"slices"
)

type dispatcher struct {
	mux        *http.ServeMux
	middleware []Middleware
}

func (d *dispatcher) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var handler http.Handler = d.mux
	if _, pattern := d.mux.Handler(request); pattern != "" {
		for _, wrap := range slices.Backward(d.middleware) {
			handler = wrap(pattern, handler)
		}
	}

	handler.ServeHTTP(writer, request)
}
