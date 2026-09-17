//revive:disable:package-comments
package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
)

func tlsConfig(configured configuration) (*tls.Config, error) {
	if configured.TLSCert == "" && configured.TLSKey == "" {
		if configured.TLSClientCA != "" {
			return nil, errors.New("client CA requires server TLS material")
		}

		return nil, nil
	}

	cert, err := tls.X509KeyPair([]byte(configured.TLSCert), []byte(configured.TLSKey))
	if err != nil {
		return nil, fmt.Errorf("improperly configured cert and/or key: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if configured.TLSClientCA != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(configured.TLSClientCA)) {
			return nil, errors.New("no certificate found")
		}

		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return cfg, nil
}
