package client

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
)

// buildTLSConfig turns the TLS-related Config fields into a *tls.Config. It
// returns (nil, nil) when the defaults are sufficient so the transport keeps
// using Go's built-in configuration.
func buildTLSConfig(cfg Config) (*tls.Config, error) {
	custom := cfg.InsecureSkipVerify ||
		strings.TrimSpace(cfg.CACert) != "" ||
		strings.TrimSpace(cfg.ClientCert) != "" ||
		strings.TrimSpace(cfg.ClientKey) != "" ||
		strings.TrimSpace(cfg.TLSServerName) != ""
	if !custom {
		return nil, nil
	}

	t := &tls.Config{MinVersion: tls.VersionTLS12}

	if cfg.InsecureSkipVerify {
		t.InsecureSkipVerify = true //nolint:gosec // explicitly opted in via provider config
	}
	if sn := strings.TrimSpace(cfg.TLSServerName); sn != "" {
		t.ServerName = sn
	}

	if ca := strings.TrimSpace(cfg.CACert); ca != "" {
		pemData, err := pemBytes(ca)
		if err != nil {
			return nil, fmt.Errorf("ca_cert: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pemData) {
			return nil, errors.New("ca_cert: no valid certificate found in PEM data")
		}
		t.RootCAs = pool
	}

	cert := strings.TrimSpace(cfg.ClientCert)
	key := strings.TrimSpace(cfg.ClientKey)
	if (cert == "") != (key == "") {
		return nil, errors.New("client_cert and client_key must both be set for mTLS")
	}
	if cert != "" {
		certPEM, err := pemBytes(cert)
		if err != nil {
			return nil, fmt.Errorf("client_cert: %w", err)
		}
		keyPEM, err := pemBytes(key)
		if err != nil {
			return nil, fmt.Errorf("client_key: %w", err)
		}
		pair, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("client certificate/key pair: %w", err)
		}
		t.Certificates = []tls.Certificate{pair}
	}

	return t, nil
}

// pemBytes returns v verbatim when it already contains PEM data, otherwise it
// reads v as a path to a PEM file.
func pemBytes(v string) ([]byte, error) {
	if strings.Contains(v, "-----BEGIN") {
		return []byte(v), nil
	}
	return os.ReadFile(v)
}
