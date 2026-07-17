package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/browser"
)

// openURL opens an http(s) URL in the server machine's default browser.
// Intended for the desktop (Wails) app, where the UI and backend share a host.
// Plain web / Docker clients must open links in the browser via window.open
// instead — calling this from a remote browser would open a URL on the server.
func (s *Server) openURL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if !bindJSON(w, r, &body) {
		return
	}
	raw := strings.TrimSpace(body.URL)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url must be http or https"})
		return
	}
	if err := browser.OpenURL(parsed.String()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
