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
			ResponseHeaderTimeout: 90 * time.Second,
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
			ResponseHeaderTimeout: 90 * time.Second,
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
// It does not auto-pick a local Clash port — that can break direct-reachable
// sites (e.g. cn.bing.com). Use LocalProxyServer for optional browser retries.
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

// ProxyServer returns a Chrome-compatible proxy server string (e.g. "http://127.0.0.1:7890")
// when a configured proxy answers a short TCP dial. Returns "" when none is set or unreachable.
func ProxyServer() string {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/", nil)
	u, err := Proxy(req)
	if err != nil || u == nil || u.Host == "" {
		return ""
	}
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", u.Host)
	if err != nil {
		return ""
	}
	_ = conn.Close()
	return u.String()
}

// Do sends req with the proxy-aware client. On network/timeout failures it
// retries direct, then a probed local HTTP proxy (Clash etc.) when available.
func Do(req *http.Request, timeout time.Duration) (*http.Response, error) {
	resp, err := Client(timeout).Do(req)
	if err == nil {
		return resp, nil
	}
	var errs []error
	errs = append(errs, err)
	if !isRetryable(err) {
		return nil, err
	}

	if HasProxy() {
		retry, err2 := cloneRequest(req)
		if err2 != nil {
			return nil, err
		}
		resp2, err2 := DirectClient(timeout).Do(retry)
		if err2 == nil {
			return resp2, nil
		}
		errs = append(errs, err2)
		if !isRetryable(err2) {
			return nil, errors.Join(errs...)
		}
	}

	if local := LocalProxyServer(); local != "" {
		if u, perr := url.Parse(local); perr == nil && u.Host != "" {
			retry, err2 := cloneRequest(req)
			if err2 != nil {
				return nil, errors.Join(errs...)
			}
			client := &http.Client{
				Timeout: timeout,
				Transport: &http.Transport{
					Proxy:                 http.ProxyURL(u),
					DialContext:           baseDialer().DialContext,
					ForceAttemptHTTP2:     true,
					TLSHandshakeTimeout:   8 * time.Second,
					ResponseHeaderTimeout: 90 * time.Second,
				},
			}
			resp3, err3 := client.Do(retry)
			if err3 == nil {
				return resp3, nil
			}
			errs = append(errs, err3)
		}
	}
	return nil, errors.Join(errs...)
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
