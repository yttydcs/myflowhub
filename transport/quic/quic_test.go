package quic

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/internal/transporttest"
	"github.com/yttydcs/myflowhub/runtime/link"
)

func TestTransportContract(t *testing.T) {
	certificate := selfSignedCertificate(t)
	driver, err := New(Config{
		ServerTLS: &tls.Config{Certificates: []tls.Certificate{certificate}},
		ClientTLS: &tls.Config{InsecureSkipVerify: true}, // Test-only self-signed loopback.
	})
	if err != nil {
		t.Fatal(err)
	}
	transporttest.Run(t, driver, link.Endpoint("quic://127.0.0.1:0"))
}

func TestEndpointAndTLSValidation(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("missing TLS configuration accepted")
	}
	driver, err := New(Config{ClientTLS: &tls.Config{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.Dial(context.Background(), "quic://127.0.0.1:1234"); err == nil {
		t.Fatal("verified IP dial without server_name accepted")
	}
	if _, err := parseEndpoint("tcp://127.0.0.1:1", false); err == nil {
		t.Fatal("wrong endpoint scheme accepted")
	}
}

func selfSignedCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: privateKey}
}
