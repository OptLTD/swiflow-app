package httputil_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/OptLTD/swiflow/internal/httputil"
)

func TestProxyFallsBackToEnv(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:18080")
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:18080")
	t.Setenv("NO_PROXY", "")
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	u, err := httputil.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if u == nil {
		t.Fatal("expected proxy from env")
	}
	if u.Host != "127.0.0.1:18080" {
		t.Fatalf("host = %q", u.Host)
	}
}

func TestClientUsesSharedTransport(t *testing.T) {
	c := httputil.Client(0)
	if c.Transport != httputil.Transport() {
		t.Fatal("client should use shared transport")
	}
	_ = url.URL{}
}
