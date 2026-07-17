package httputil_test

import (
	"testing"

	"github.com/OptLTD/swiflow/library/httputil"
)

func TestLocalProxyServer(t *testing.T) {
	s := httputil.LocalProxyServer()
	if s == "" {
		t.Skip("no reachable local HTTP proxy (e.g. Clash :7890)")
	}
	t.Logf("LocalProxyServer = %s", s)
}
