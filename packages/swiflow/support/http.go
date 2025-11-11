package support

import (
	"log"
	"net/http"
)

type headerRoundTripper struct {
	rt      http.RoundTripper
	headers map[string]string
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for key, value := range h.headers {
		req.Header.Set(key, value)
		val := MaskMiddle(value)
		log.Printf("[HTTP] header: %s = %s\n", key, val)
	}
	return h.rt.RoundTrip(req)
}

func NewHttpTransport(headers map[string]string) http.RoundTripper {
	return &headerRoundTripper{rt: http.DefaultTransport, headers: headers}
}
func NewHttpClient(headers map[string]string) *http.Client {
	return &http.Client{Transport: NewHttpTransport(headers)}
}
