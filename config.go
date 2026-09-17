//revive:disable:package-comments
package httpserver

import "time"

type configuration struct {
	IdleTimeout     time.Duration `env:"HOST_IDLE_TIMEOUT" envDefault:"5m"`
	SendPingTimeout time.Duration `env:"HTTP2_SEND_PING_TIMEOUT" envDefault:"2m"`
	PingTimeout     time.Duration `env:"HTTP2_PING_TIMEOUT" envDefault:"20s"`
	TLSCert         string        `env:"TLS_CERT"`
	TLSKey          string        `env:"TLS_KEY"`
	TLSClientCA     string        `env:"TLS_CLIENT_CA"`
}
