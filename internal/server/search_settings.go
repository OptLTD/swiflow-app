package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
)

const (
	settingSearchProvider = "search.provider"
	settingSearchAPIKey   = "search.api_key"
	settingSearchBaseURL  = "search.base_url"
)

// LoadSearchSettings overlays persisted search settings onto opts when keys exist in DB.
func LoadSearchSettings(ctx context.Context, st store.Store, opts *tool.WebOptions) {
	if opts == nil || st == nil {
		return
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchProvider); err == nil && ok {
		opts.SearchProvider = v
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchAPIKey); err == nil && ok {
		opts.SearchAPIKey = v
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchBaseURL); err == nil && ok {
		opts.SearchBaseURL = v
	}
}

func (s *Server) getSearchSettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":    s.webOpts.SearchProvider,
		"base_url":    s.webOpts.SearchBaseURL,
		"api_key_set": strings.TrimSpace(s.webOpts.SearchAPIKey) != "",
	})
}

func (s *Server) putSearchSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Provider *string `json:"provider"`
		APIKey   *string `json:"api_key"`
		BaseURL  *string `json:"base_url"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Provider != nil {
		p := strings.ToLower(strings.TrimSpace(*in.Provider))
		switch p {
		case "", "duckduckgo", "ddg", "brave", "searxng", "searx", "bing", "google":
		default:
			writeErr(w, http.StatusBadRequest, ErrUnsupportedSearchProvider)
			return
		}
		if p == "ddg" {
			p = "duckduckgo"
		}
		if p == "searx" {
			p = "searxng"
		}
		if err := s.st.SetSysSetting(r.Context(), settingSearchProvider, p); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrSaveFailed)
			return
		}
		s.webOpts.SearchProvider = p
	}
	if in.APIKey != nil {
		if err := s.st.SetSysSetting(r.Context(), settingSearchAPIKey, *in.APIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrSaveFailed)
			return
		}
		s.webOpts.SearchAPIKey = *in.APIKey
	}
	if in.BaseURL != nil {
		u := strings.TrimSpace(*in.BaseURL)
		if err := s.st.SetSysSetting(r.Context(), settingSearchBaseURL, u); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrSaveFailed)
			return
		}
		s.webOpts.SearchBaseURL = u
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "updated",
		"provider":    s.webOpts.SearchProvider,
		"base_url":    s.webOpts.SearchBaseURL,
		"api_key_set": strings.TrimSpace(s.webOpts.SearchAPIKey) != "",
	})
}
