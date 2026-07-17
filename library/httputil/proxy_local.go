package httputil

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Common local HTTP proxy ports (Clash / Surge / etc.) used when env and OS
// system proxy are unset. Only adopted after a real HTTPS probe succeeds.
var localProxyCandidates = []string{
	"127.0.0.1:7890",
	"127.0.0.1:7891",
	"127.0.0.1:1087",
	"127.0.0.1:6152",
	// "127.0.0.1:8080",
}

var (
	localOnce  sync.Once
	localProxy *url.URL
)

// LocalProxyServer returns a reachable local HTTP proxy (e.g. Clash :7890) after
// an HTTPS probe, or "" when none works. Intended for Google browser search retry
// when env/OS proxy is unset — not for all HTTP traffic.
func LocalProxyServer() string {
	localOnce.Do(discoverLocalProxy)
	if localProxy == nil {
		return ""
	}
	return localProxy.String()
}

func discoverLocalProxy() {
	for _, host := range localProxyCandidates {
		if !tcpOpen(host, 300*time.Millisecond) {
			continue
		}
		u := &url.URL{Scheme: "http", Host: host}
		if probeHTTPProxy(u) {
			localProxy = u
			return
		}
	}
}

func tcpOpen(host string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", host)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeHTTPProxy(u *url.URL) bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(u),
			DialContext:           (&net.Dialer{Timeout: 2 * time.Second}).DialContext,
			TLSHandshakeTimeout:   2 * time.Second,
			ResponseHeaderTimeout: 2 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
	resp, err := client.Get("https://www.gstatic.com/generate_204")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK
}
