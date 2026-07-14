// Package httputil provides shared HTTP clients that honor env and OS system proxies.
package httputil

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	once            sync.Once
	transport       *http.Transport
	directOnce      sync.Once
	directTransport *http.Transport
)

func baseDialer() *net.Dialer {
	return &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
}

// Transport returns a shared *http.Transport that uses HTTP(S)_PROXY when set,
// otherwise falls back to the OS system proxy (e.g. macOS Network settings).
func Transport() *http.Transport {
	once.Do(func() {
		transport = &http.Transport{
			Proxy:                 Proxy,
			DialContext:           baseDialer().DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          32,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 12 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	})
	return transport
}

// DirectTransport never uses a proxy (for fallback when the proxy path fails).
func DirectTransport() *http.Transport {
	directOnce.Do(func() {
		directTransport = &http.Transport{
			Proxy:                 nil,
			DialContext:           baseDialer().DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          32,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   8 * time.Second,
			ResponseHeaderTimeout: 12 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	})
	return directTransport
}

// Client returns an *http.Client with timeout that uses Transport() (proxy-aware).
func Client(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: Transport(),
	}
}

// DirectClient returns an *http.Client that never uses a proxy.
func DirectClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: DirectTransport(),
	}
}

// Proxy chooses a proxy for req: environment variables first, then OS settings.
func Proxy(req *http.Request) (*url.URL, error) {
	if u, err := http.ProxyFromEnvironment(req); err == nil && u != nil {
		return u, nil
	}
	return systemProxyURL(req)
}

// HasProxy reports whether an env or OS proxy would apply to a dummy HTTPS URL.
func HasProxy() bool {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	u, err := Proxy(req)
	return err == nil && u != nil
}

// Do sends req with the proxy-aware client. On network/timeout failures, retries
// once without a proxy (useful when system proxy breaks some reachable sites).
func Do(req *http.Request, timeout time.Duration) (*http.Response, error) {
	resp, err := Client(timeout).Do(req)
	if err == nil {
		return resp, nil
	}
	if !HasProxy() || !isRetryable(err) {
		return nil, err
	}
	retry, err2 := cloneRequest(req)
	if err2 != nil {
		return nil, err
	}
	resp2, err2 := DirectClient(timeout).Do(retry)
	if err2 != nil {
		return nil, errors.Join(err, err2)
	}
	return resp2, nil
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"timeout", "reset by peer", "connection refused",
		"i/o timeout", "tls handshake", "eof",
		"network is unreachable", "client.timeout",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	out := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return out, nil
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		out.Body = body
		return out, nil
	}
	return nil, errors.New("cannot retry request with non-replayable body")
}
