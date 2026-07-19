package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type actCtxKey struct{}

type actValues struct {
	ID   string
	Key  string
	Name string
	Slug string
}

func withActValues(r *http.Request, v actValues) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), actCtxKey{}, v))
}

func actFrom(r *http.Request) actValues {
	if v, ok := r.Context().Value(actCtxKey{}).(actValues); ok {
		return v
	}
	return actValues{}
}

func requestID(r *http.Request) string {
	if v := actFrom(r).ID; v != "" {
		return v
	}
	if v := r.PathValue("id"); v != "" {
		return v
	}
	return r.PathValue("key")
}

func requestKey(r *http.Request) string {
	if v := actFrom(r).Key; v != "" {
		return v
	}
	return r.PathValue("key")
}

func requestName(r *http.Request) string {
	if v := actFrom(r).Name; v != "" {
		return v
	}
	return r.PathValue("name")
}

func requestSlug(r *http.Request) string {
	if v := actFrom(r).Slug; v != "" {
		return v
	}
	return r.PathValue("slug")
}

// readActBody reads the JSON body once and returns raw fields + act string.
func readActBody(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, string, bool) {
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, ErrInvalidJSON)
		return nil, "", false
	}
	if len(bytes.TrimSpace(data)) == 0 {
		writeErr(w, http.StatusBadRequest, ErrActRequired)
		return nil, "", false
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		writeErr(w, http.StatusBadRequest, ErrInvalidJSON)
		return nil, "", false
	}
	act := strings.TrimSpace(rawString(raw, "act"))
	if act == "" {
		writeErr(w, http.StatusBadRequest, ErrActRequired)
		return nil, "", false
	}
	return raw, act, true
}

func rawString(raw map[string]json.RawMessage, key string) string {
	v, ok := raw[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return ""
	}
	return s
}

// bodyWithoutAct returns JSON bytes with act (and optional keys) removed, for
// re-feeding handlers that call bindJSON.
func bodyWithoutAct(raw map[string]json.RawMessage, drop ...string) []byte {
	out := make(map[string]json.RawMessage, len(raw))
	skip := map[string]bool{"act": true}
	for _, k := range drop {
		skip[k] = true
	}
	for k, v := range raw {
		if skip[k] {
			continue
		}
		out[k] = v
	}
	b, _ := json.Marshal(out)
	return b
}

func withJSONBody(r *http.Request, body []byte) *http.Request {
	r2 := r.Clone(r.Context())
	r2.Body = io.NopCloser(bytes.NewReader(body))
	r2.ContentLength = int64(len(body))
	r2.Header = r.Header.Clone()
	if r2.Header == nil {
		r2.Header = make(http.Header)
	}
	r2.Header.Set("Content-Type", "application/json")
	return r2
}

func requireID(w http.ResponseWriter, id string) bool {
	if id == "" {
		writeErr(w, http.StatusBadRequest, ErrIDRequired)
		return false
	}
	return true
}

func unknownAct(w http.ResponseWriter, act string) {
	writeErr(w, http.StatusBadRequest, ErrUnknownAct, act)
}
