package smtptest

import (
	"testing"
	"time"

	"github.com/flashmob/go-guerrilla/tests/testcert"
)

// GenerateTLSFiles writes a TLS key and certificate to a temporary test
// directory that is removed after the test suite runs. It returns the file
// paths of the key and certificate. The certificate is a root cert.
func GenerateTLSFiles(t *testing.T) (keyPath string, certPath string, err error) {
	host := "127.0.0.1"
	d := t.TempDir()
	err = testcert.GenerateCert(
		host,
		"",                         // defaults to now
		time.Duration(1)*time.Hour, // the test suite won't run for this long
		true,                       // is a CA cert
		2048,                       // usually seen in online tutorials
		"",                         // using the default ecdsa curve,
		d,
	)

	if err != nil {
		return
	}

	// These path names are hardcoded into testcert.GenerateCert
	keyPath = d + host + ".key.pem"
	certPath = d + host + ".cert.pem"

	return
}
