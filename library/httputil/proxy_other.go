//go:build !darwin

package httputil

import (
	"net/http"
	"net/url"
)

func systemProxyURL(_ *http.Request) (*url.URL, error) {
	return nil, nil
}
