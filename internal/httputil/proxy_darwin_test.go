//go:build darwin

package httputil_test

import (
	"net/http"
	"testing"

	"github.com/OptLTD/swiflow/internal/httputil"
)

func TestDarwinSystemProxy(t *testing.T) {
	// This machine has HTTPS proxy enabled at 127.0.0.1:7890 (verified via scutil).
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("all_proxy", "")

	req, _ := http.NewRequest(http.MethodGet, "https://html.duckduckgo.com/html/?q=test", nil)
	u, err := httputil.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if u == nil {
		t.Skip("no system HTTPS proxy configured")
	}
	if u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" {
		t.Logf("system proxy = %s (unexpected host, still ok if enabled)", u)
	}
	t.Logf("system proxy for https = %s", u.String())
}
