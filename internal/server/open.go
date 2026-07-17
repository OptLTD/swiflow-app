package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/browser"
)

// openURL opens an http(s) URL in the user's default system browser.
// Used by the WebUI "访问" action on web-related tool calls.
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
