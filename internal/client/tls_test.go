package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildTLSConfigDefaults(t *testing.T) {
	cfg, err := buildTLSConfig(Config{})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil tls.Config for default config, got %+v", cfg)
	}
}

func TestBuildTLSConfigInsecureAndServerName(t *testing.T) {
	cfg, err := buildTLSConfig(Config{InsecureSkipVerify: true, TLSServerName: "idoit.internal"})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg == nil || !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify not applied")
	}
	if cfg.ServerName != "idoit.internal" {
		t.Fatalf("ServerName = %q", cfg.ServerName)
	}
}

func TestBuildTLSConfigCACertPEM(t *testing.T) {
	certPEM, _ := selfSignedPEM(t)
	cfg, err := buildTLSConfig(Config{CACert: string(certPEM)})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatal("RootCAs not populated from PEM CA cert")
	}
}

func TestBuildTLSConfigCACertFile(t *testing.T) {
	certPEM, _ := selfSignedPEM(t)
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := buildTLSConfig(Config{CACert: path})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatal("RootCAs not populated from PEM CA file")
	}
}

func TestBuildTLSConfigCACertInvalid(t *testing.T) {
	if _, err := buildTLSConfig(Config{CACert: "this-is-not-pem-and-not-a-file"}); err == nil {
		t.Fatal("expected error for bogus ca_cert")
	}
}

func TestBuildTLSConfigClientCertNeedsKey(t *testing.T) {
	certPEM, _ := selfSignedPEM(t)
	if _, err := buildTLSConfig(Config{ClientCert: string(certPEM)}); err == nil {
		t.Fatal("expected error when client_key is missing")
	}
}

func TestBuildTLSConfigClientCertPair(t *testing.T) {
	certPEM, keyPEM := selfSignedPEM(t)
	cfg, err := buildTLSConfig(Config{ClientCert: string(certPEM), ClientKey: string(keyPEM)})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg == nil || len(cfg.Certificates) != 1 {
		t.Fatal("client certificate not loaded")
	}
}

// selfSignedPEM returns a throwaway self-signed certificate and its private key,
// both PEM-encoded.
func selfSignedPEM(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}
