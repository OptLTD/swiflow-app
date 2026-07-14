//go:build darwin

package httputil

import (
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var (
	sysOnce   sync.Once
	sysHTTP   *url.URL // http://host:port
	sysHTTPS  *url.URL
	sysSOCKS  *url.URL // socks5://host:port
	sysExcept []string
)

var scutilKV = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9.]+)\s*:\s*(.+)\s*$`)

func systemProxyURL(req *http.Request) (*url.URL, error) {
	sysOnce.Do(loadDarwinProxy)
	host := req.URL.Hostname()
	if host == "" || bypassProxy(host, sysExcept) {
		return nil, nil
	}
	if req.URL.Scheme == "https" && sysHTTPS != nil {
		return sysHTTPS, nil
	}
	if sysHTTP != nil {
		return sysHTTP, nil
	}
	if sysSOCKS != nil {
		return sysSOCKS, nil
	}
	if sysHTTPS != nil {
		return sysHTTPS, nil
	}
	return nil, nil
}

func loadDarwinProxy() {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return
	}
	vals := map[string]string{}
	for _, m := range scutilKV.FindAllStringSubmatch(string(out), -1) {
		vals[m[1]] = strings.TrimSpace(m[2])
	}
	sysExcept = parseExceptions(string(out))

	if vals["HTTPEnable"] == "1" {
		if u := proxyHTTPURL(vals["HTTPProxy"], vals["HTTPPort"]); u != nil {
			sysHTTP = u
		}
	}
	if vals["HTTPSEnable"] == "1" {
		if u := proxyHTTPURL(vals["HTTPSProxy"], vals["HTTPSPort"]); u != nil {
			sysHTTPS = u
		}
	}
	if vals["SOCKSEnable"] == "1" {
		if u := proxySOCKSURL(vals["SOCKSProxy"], vals["SOCKSPort"]); u != nil {
			sysSOCKS = u
		}
	}
}

func proxyHTTPURL(host, port string) *url.URL {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" || port == "" {
		return nil
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}
}

func proxySOCKSURL(host, port string) *url.URL {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" || port == "" {
		return nil
	}
	return &url.URL{Scheme: "socks5", Host: net.JoinHostPort(host, port)}
}

func parseExceptions(scutilOut string) []string {
	var out []string
	in := false
	for _, line := range strings.Split(scutilOut, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ExceptionsList") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if line == "}" {
			break
		}
		if i := strings.Index(line, ":"); i >= 0 {
			v := strings.TrimSpace(line[i+1:])
			if v != "" && v != "<array>" && v != "{" {
				out = append(out, v)
			}
		}
	}
	return out
}

func bypassProxy(host string, exceptions []string) bool {
	host = strings.ToLower(host)
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	for _, ex := range exceptions {
		ex = strings.ToLower(strings.TrimSpace(ex))
		if ex == "" {
			continue
		}
		if strings.Contains(ex, "/") {
			if ip := net.ParseIP(host); ip != nil {
				if _, n, err := net.ParseCIDR(ex); err == nil && n.Contains(ip) {
					return true
				}
			}
			continue
		}
		if strings.HasPrefix(ex, "*.") {
			suf := ex[1:] // ".example.com"
			if strings.HasSuffix(host, suf) || host == ex[2:] {
				return true
			}
			continue
		}
		if host == ex || strings.HasSuffix(host, "."+ex) {
			return true
		}
	}
	return false
}
