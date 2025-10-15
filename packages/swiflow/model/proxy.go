package model

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"swiflow/config"

	"golang.org/x/net/proxy"
)

// headerRoundTripper
type headerRoundTripper struct {
	rt      http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for key, value := range h.headers {
		req.Header.Set(key, value)
		log.Printf("[PROXY] header: %s = %s\n", key, value)
	}
	return h.rt.RoundTrip(req)
}

func NewProxyHttpClient(headers map[string]string) *http.Client {
	var transport http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if len(headers) > 0 {
		transport = &headerRoundTripper{
			rt: transport, headers: headers,
		}
	}
	client := &http.Client{
		Transport: transport,
	}
	proxyUrl := config.GetStr("PROXY_URL", "")
	u, err := url.Parse(proxyUrl)
	if err != nil || proxyUrl == "" {
		return client
	}
	switch u.Scheme {
	case "http", "https":
		transport = &http.Transport{Proxy: http.ProxyURL(u)}
	case "socks5":
		var auth *proxy.Auth
		if u.User != nil {
			password, _ := u.User.Password()
			auth = &proxy.Auth{
				User:     u.User.Username(),
				Password: password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
		if err == nil {
			proxyTransport := &http.Transport{}
			proxyTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
			transport = proxyTransport
		}
	}

	if len(headers) > 0 {
		transport = &headerRoundTripper{
			rt: transport, headers: headers,
		}
	}

	client.Transport = transport
	return client
}
