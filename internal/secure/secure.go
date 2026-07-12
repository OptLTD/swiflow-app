// Package secure provides pure helpers: AES-256-GCM encryption for secrets,
// an SSRF guard for outbound web requests, and a path sandbox for file tools.
// Spec §6.3, §12.
package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"
)

// --- AES-256-GCM ---

// DeriveKey turns an arbitrary passphrase into a 32-byte AES-256 key via SHA-256.
func DeriveKey(passphrase string) []byte {
	h := sha256.Sum256([]byte(passphrase))
	return h[:]
}

// Encrypt returns nonce||ciphertext.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ct...), nil
}

// Decrypt reverses Encrypt (input is nonce||ciphertext).
func Decrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(data) < ns+1 {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	return gcm.Open(nil, nonce, ct, nil)
}

// --- Path sandbox ---

// SandboxPath resolves requested against workspace and rejects any path that
// escapes the workspace. Returns the cleaned absolute path.
func SandboxPath(workspace, requested string) (string, error) {
	ws, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	full := requested
	if !filepath.IsAbs(full) {
		full = filepath.Join(ws, full)
	}
	full, err = filepath.Abs(filepath.Clean(full))
	if err != nil {
		return "", err
	}
	// Resolve symlinks where possible; fall back to lexical check.
	if resolved, err := filepath.EvalSymlinks(full); err == nil {
		full = resolved
	}
	rel, err := filepath.Rel(ws, full)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("path escapes workspace: %s", requested)
	}
	return full, nil
}

// ValidateHTTPURL checks that rawURL is an absolute http or https URL with a host.
// Provider base URLs are not SSRF-filtered (private inference endpoints are allowed).
func ValidateHTTPURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are supported")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("missing hostname")
	}
	return nil
}

// --- SSRF guard ---

var blockedHostnames = map[string]bool{
	"localhost":                true,
	"metadata.google.internal": true,
}

// CheckURL validates that rawURL is http(s), has a host, and does not resolve
// to a private/loopback/link-local/CGNAT address or a blocked metadata host.
func CheckURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("only http and https URLs are supported")
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("missing hostname")
	}
	lhost := strings.ToLower(host)
	if blockedHostnames[lhost] {
		return fmt.Errorf("blocked hostname: %s", host)
	}
	if strings.HasSuffix(lhost, ".localhost") ||
		strings.HasSuffix(lhost, ".local") ||
		strings.HasSuffix(lhost, ".internal") {
		return fmt.Errorf("blocked hostname: %s", host)
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("private IP not allowed: %s", host)
		}
		return nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed: %w", err)
	}
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip != nil && isPrivateIP(ip) {
			return fmt.Errorf("hostname %s resolves to private IP %s", host, a)
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	for _, c := range privateCIDRs {
		if c.Contains(ip) {
			return true
		}
	}
	return false
}

var privateCIDRs []*net.IPNet

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8", "10.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10",
		"::1/128", "fc00::/7", "fe80::/10",
	} {
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			privateCIDRs = append(privateCIDRs, n)
		}
	}
}

// MaskKey returns a masked form of an API key for safe logging.
func MaskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// HexEncode is a convenience for storing ciphertext as hex if needed.
func HexEncode(b []byte) string { return hex.EncodeToString(b) }
