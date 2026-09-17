//revive:disable:package-comments
package httpserver

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/pbrpc/testing/mocks/certificate"
	"github.com/pbrpc/testing/mocks/listener"
)

func setConfiguration(t *testing.T) {
	t.Helper()
	t.Setenv("HOST_ADDRESS", "127.0.0.1:8080")
	t.Setenv("HOST_IDLE_TIMEOUT", "1m")
	t.Setenv("HTTP2_SEND_PING_TIMEOUT", "4m")
	t.Setenv("HTTP2_PING_TIMEOUT", "5s")
	t.Setenv("TLS_CERT", "")
	t.Setenv("TLS_KEY", "")
	t.Setenv("TLS_CLIENT_CA", "")
}

func TestFromEnv(t *testing.T) {
	t.Run("builds the cleartext HTTP server", func(t *testing.T) {
		setConfiguration(t)

		srv, err := FromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.Address != "127.0.0.1:8080" {
			t.Errorf("address = %q, want 127.0.0.1:8080", srv.Address)
		}
		if srv.Mux == nil || srv.Server == nil || srv.Server.Handler == nil {
			t.Fatalf("server = %+v, want the mux and HTTP server", srv)
		}
		if srv.Server.IdleTimeout != time.Minute {
			t.Errorf("idle timeout = %v, want 1m", srv.Server.IdleTimeout)
		}
		if srv.Server.HTTP2.SendPingTimeout != 4*time.Minute {
			t.Errorf("send ping timeout = %v, want 4m", srv.Server.HTTP2.SendPingTimeout)
		}
		if srv.Server.HTTP2.PingTimeout != 5*time.Second {
			t.Errorf("ping timeout = %v, want 5s", srv.Server.HTTP2.PingTimeout)
		}
		if !srv.Server.Protocols.HTTP1() || !srv.Server.Protocols.UnencryptedHTTP2() {
			t.Errorf("protocols = %v, want HTTP/1.1 and cleartext HTTP/2", srv.Server.Protocols)
		}
		if srv.Server.Protocols.HTTP2() {
			t.Errorf("protocols = %v, want TLS HTTP/2 disabled", srv.Server.Protocols)
		}
	})

	t.Run("returns invalid duration configuration", func(t *testing.T) {
		setConfiguration(t)
		t.Setenv("HOST_IDLE_TIMEOUT", "not-a-duration")

		if _, err := FromEnv(); err == nil {
			t.Fatal("error = nil, want the configuration error")
		}
	})

	t.Run("builds the TLS server", func(t *testing.T) {
		setConfiguration(t)
		certPEM, keyPEM := certificate.PEM(t)
		t.Setenv("TLS_CERT", certPEM)
		t.Setenv("TLS_KEY", keyPEM)

		host, err := FromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(host.Server.TLSConfig.Certificates) != 1 {
			t.Fatalf("certificates = %d, want the one loaded", len(host.Server.TLSConfig.Certificates))
		}
		if host.Server.TLSConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("minimum TLS version = %d, want TLS 1.2", host.Server.TLSConfig.MinVersion)
		}
		if !host.Server.Protocols.HTTP1() || !host.Server.Protocols.HTTP2() {
			t.Errorf("protocols = %v, want HTTP/1.1 and HTTP/2 over TLS", host.Server.Protocols)
		}
		if host.Server.Protocols.UnencryptedHTTP2() {
			t.Errorf("protocols = %v, want cleartext HTTP/2 disabled", host.Server.Protocols)
		}
	})

	t.Run("requires a client certificate when a CA is configured", func(t *testing.T) {
		setConfiguration(t)
		certPEM, keyPEM := certificate.PEM(t)
		t.Setenv("TLS_CERT", certPEM)
		t.Setenv("TLS_KEY", keyPEM)
		t.Setenv("TLS_CLIENT_CA", certPEM)

		srv, err := FromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv.Server.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Errorf("client authentication = %v, want required and verified", srv.Server.TLSConfig.ClientAuth)
		}
		if srv.Server.TLSConfig.ClientCAs == nil {
			t.Error("client CA pool is nil")
		}
	})

	t.Run("returns invalid certificate material", func(t *testing.T) {
		setConfiguration(t)
		_, keyPEM := certificate.PEM(t)
		t.Setenv("TLS_KEY", keyPEM)

		if _, err := FromEnv(); err == nil {
			t.Fatal("error = nil, want the certificate error")
		}
	})

	t.Run("returns an invalid client CA", func(t *testing.T) {
		setConfiguration(t)
		certPEM, keyPEM := certificate.PEM(t)
		t.Setenv("TLS_CERT", certPEM)
		t.Setenv("TLS_KEY", keyPEM)
		t.Setenv("TLS_CLIENT_CA", "not pem")

		if _, err := FromEnv(); err == nil {
			t.Fatal("error = nil, want the client CA error")
		}
	})

	t.Run("returns a client CA without server TLS material", func(t *testing.T) {
		setConfiguration(t)
		certPEM, _ := certificate.PEM(t)
		t.Setenv("TLS_CLIENT_CA", certPEM)

		if _, err := FromEnv(); err == nil {
			t.Fatal("error = nil, want the missing server TLS material error")
		}
	})
}

func TestMiddleware(t *testing.T) {
	setConfiguration(t)

	var calls []string
	wrap := func(name string) Middleware {
		return func(pattern string, next http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				calls = append(calls, name+":"+pattern)
				next.ServeHTTP(writer, request)
			})
		}
	}

	srv, err := FromEnv(wrap("outer"), wrap("inner"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	srv.Mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	srv.Server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if want := []string{"outer:/healthz", "inner:/healthz"}; !slices.Equal(calls, want) {
		t.Errorf("middleware calls = %v, want %v", calls, want)
	}

	calls = nil
	response = httptest.NewRecorder()
	request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/missing", nil)
	srv.Server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", response.Code)
	}
	if len(calls) != 0 {
		t.Errorf("middleware calls = %v, want none for an unmatched route", calls)
	}
}

func TestServe(t *testing.T) {
	serveErr := errors.New("listener failed")

	t.Run("serves cleartext", func(t *testing.T) {
		setConfiguration(t)
		srv, err := FromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err = srv.Serve(listener.NewFailing(serveErr)); !errors.Is(err, serveErr) {
			t.Fatalf("error = %v, want %v", err, serveErr)
		}
	})

	t.Run("serves TLS", func(t *testing.T) {
		setConfiguration(t)
		certPEM, keyPEM := certificate.PEM(t)
		t.Setenv("TLS_CERT", certPEM)
		t.Setenv("TLS_KEY", keyPEM)

		srv, err := FromEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err = srv.Serve(listener.NewFailing(serveErr)); !errors.Is(err, serveErr) {
			t.Fatalf("error = %v, want %v", err, serveErr)
		}
	})
}
