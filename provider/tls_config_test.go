package provider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// selfSignedCAPEM returns a PEM-encoded self-signed CA certificate, so the
// happy path can be asserted without checking in a fixture that will expire.
func selfSignedCAPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// A cacert_file that cannot be turned into a usable CA pool must fail loudly.
// Previously the read error and the AppendCertsFromPEM result were both
// discarded, leaving RootCAs set to an empty pool -- which trusts nothing and
// surfaces much later as "certificate signed by unknown authority".
func TestTLSHttpClientCacertFileErrors(t *testing.T) {
	dir := t.TempDir()

	unreadable := filepath.Join(dir, "notpem.crt")
	if err := os.WriteFile(unreadable, []byte("this is not a certificate"), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	tests := []struct {
		name       string
		cacertFile string
		wantSubstr []string
	}{
		{
			name:       "directory instead of file",
			cacertFile: dir,
			wantSubstr: []string{"cacert_file", dir},
		},
		{
			name:       "path that does not exist",
			cacertFile: filepath.Join(dir, "missing", "ca.pem"),
			wantSubstr: []string{"cacert_file", "neither a readable file nor a PEM-encoded CA bundle"},
		},
		{
			name:       "file that is not PEM",
			cacertFile: unreadable,
			wantSubstr: []string{"cacert_file", "no certificates found", unreadable},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ProviderConf{cacertFile: tt.cacertFile}

			client, err := tlsHttpClient(conf, map[string]string{})
			if err == nil {
				t.Fatalf("expected an error for cacert_file %q, got client %v", tt.cacertFile, client)
			}
			for _, want := range tt.wantSubstr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err.Error(), want)
				}
			}
		})
	}
}

func TestTLSHttpClientCacertFileValid(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, selfSignedCAPEM(t), 0o600); err != nil {
		t.Fatalf("writing CA: %v", err)
	}

	conf := &ProviderConf{cacertFile: caPath}

	client, err := tlsHttpClient(conf, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error for a valid CA bundle: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
}

// Inline PEM content (rather than a path) is a supported way to pass
// cacert_file, and must keep working.
func TestTLSHttpClientCacertInlineContent(t *testing.T) {
	conf := &ProviderConf{cacertFile: string(selfSignedCAPEM(t))}

	if _, err := tlsHttpClient(conf, map[string]string{}); err != nil {
		t.Fatalf("unexpected error for inline PEM content: %v", err)
	}
}

// The failure has to reach the caller, not just tlsHttpClient: getClient takes
// the TLS branch whenever cacert_file is set, and used to discard the result.
func TestGetClientSurfacesCacertError(t *testing.T) {
	conf := &ProviderConf{
		rawUrl:     "https://localhost:9200",
		cacertFile: t.TempDir(), // a directory
	}
	parsed, err := url.Parse(conf.rawUrl)
	if err != nil {
		t.Fatalf("parsing url: %v", err)
	}
	conf.parsedUrl = parsed

	if _, err := getClient(conf); err == nil {
		t.Fatal("expected getClient to fail on an unusable cacert_file")
	} else if !strings.Contains(err.Error(), "cacert_file") {
		t.Errorf("error %q does not mention cacert_file", err.Error())
	}
}
