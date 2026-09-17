//revive:disable:package-comments
package httpserver

import "net/http"

// Middleware wraps the handler mounted for one route. pattern is the route
// selected by the server's mux. The first middleware passed to FromEnv is the
// outermost.
type Middleware func(pattern string, next http.Handler) http.Handler
