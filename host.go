//revive:disable:package-comments
package httpserver

import (
	"net"
	"net/http"

	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Host contains the HTTP service address, route mux, and configured server.
type Host struct {
	// Address is the TCP address the application supplies to its listener.
	Address string

	// Mux carries every HTTP route served by HTTP.
	Mux *http.ServeMux

	// Server serves Mux with the configured protocols, timeouts, and TLS.
	Server *http.Server
}

// Serve serves HTTP on listener and terminates TLS when the server carries TLS
// configuration. It blocks until the server stops or the listener fails.
func (s *Host) Serve(listener net.Listener) error {
	if s.Server.TLSConfig != nil {
		return s.Server.ServeTLS(listener, "", "")
	}

	return s.Server.Serve(listener)
}

// FromEnv creates a server from HOST_ADDRESS, HOST_IDLE_TIMEOUT,
// HTTP2_SEND_PING_TIMEOUT, HTTP2_PING_TIMEOUT, and the TLS environment
// variables.
func FromEnv(middleware ...Middleware) (*Host, error) {
	configured, err := env.ParseAs[configuration]()
	if err != nil {
		return nil, err
	}

	tlsConfig, err := tlsConfig(configured)
	if err != nil {
		return nil, err
	}

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	if tlsConfig == nil {
		protocols.SetUnencryptedHTTP2(true)
	} else {
		protocols.SetHTTP2(true)
	}

	mux := http.NewServeMux()
	handler := otelhttp.NewHandler(&dispatcher{mux: mux, middleware: middleware}, "")

	return &Host{
		Address: configured.Address,
		Mux:     mux,
		Server: &http.Server{
			Handler:     handler,
			IdleTimeout: configured.IdleTimeout,
			Protocols:   protocols,
			TLSConfig:   tlsConfig,
			HTTP2: &http.HTTP2Config{
				SendPingTimeout: configured.SendPingTimeout,
				PingTimeout:     configured.PingTimeout,
			},
		},
	}, nil
}
