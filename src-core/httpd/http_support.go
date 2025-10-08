package httpd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"swiflow/config"
	"sync"
	"time"
)

func IsPrivateIP(host string) bool {
	host = strings.Split(host, ":")[0]
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	// 检查本地地址
	if ip.IsLoopback() {
		return true
	}

	// 检查私有地址
	if ip.IsPrivate() {
		return true
	}

	return false
}

func IsInternal(host string) bool {
	// 去掉端口号（如 "localhost:8080" → "localhost"）
	host = strings.Split(host, ":")[0]

	// 检查本地地址
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	// 检查内网域名（如 .local, .internal）
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return true
	}

	if IsPrivateIP(host) || config.Get("SWIFLOW_SERVER") != "" {
		return true
	}

	return false
}

// getClientIP 获取客户端 IP（考虑 X-Forwarded-For）
func GetClientIP(r *http.Request) string {
	// 如果使用了反向代理（如 Nginx），检查 X-Forwarded-For
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// 可能有多个 IP（如 "1.2.3.4, 5.6.7.8"），取第一个
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 否则直接取 RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // 如果无法解析，返回原始值
	}
	return ip
}

func BatchCheck(domains []string) map[string]string {
	var workers = 10
	var timeout = time.Second * 3
	ctx, cancel := context.WithTimeout(
		context.Background(), timeout,
	)
	defer cancel()

	var wg sync.WaitGroup
	var mu = sync.Mutex{}
	results := make(map[string]string)
	sem := make(chan struct{}, workers) // 控制并发数

	for _, domain := range domains {
		wg.Add(1)
		sem <- struct{}{} // 占用信号量

		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			err := CheckDomain(ctx, d, timeout)
			mu.Lock()
			if err != nil {
				results[d] = err.Error()
			}
			mu.Unlock()
		}(domain)
	}

	wg.Wait()
	return results
}

func CheckDomain(ctx context.Context, domain string, timeout time.Duration) error {
	// 1. DNS 解析
	_, err := net.LookupIP(domain)
	if err != nil {
		return fmt.Errorf("DNS error: %v", err)
	}

	// 2. 创建 HTTP 客户端
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	url := fmt.Sprintf("https://%s", domain)
	req, _ := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if resp, err := client.Do(req); err == nil {
		resp.Body.Close()
		return nil
	}
	return fmt.Errorf("%s - all failed", domain)
}
