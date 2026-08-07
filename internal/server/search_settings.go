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
	provider, baseURL, apiKey := resolveSearchSettings(ctx, st, opts)
	opts.SearchProvider = provider
	opts.SearchBaseURL = baseURL
	opts.SearchAPIKey = apiKey
}

// BindSearchResolver attaches a per-request resolver so web_search reads tenant settings.
func BindSearchResolver(opts *tool.WebOptions, st store.Store) {
	if opts == nil || st == nil {
		return
	}
	fallback := *opts
	opts.ResolveSearch = func(ctx context.Context) (provider, baseURL, apiKey string) {
		return resolveSearchSettings(ctx, st, &fallback)
	}
}

func resolveSearchSettings(ctx context.Context, st store.Store, fallback *tool.WebOptions) (provider, baseURL, apiKey string) {
	if fallback != nil {
		provider = fallback.SearchProvider
		baseURL = fallback.SearchBaseURL
		apiKey = fallback.SearchAPIKey
	}
	if st == nil {
		return provider, baseURL, apiKey
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchProvider); err == nil && ok {
		provider = v
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchAPIKey); err == nil && ok {
		apiKey = v
	}
	if v, ok, err := st.GetSysSetting(ctx, settingSearchBaseURL); err == nil && ok {
		baseURL = v
	}
	return provider, baseURL, apiKey
}

func (s *Server) getSearchSettings(w http.ResponseWriter, r *http.Request) {
	provider, baseURL, apiKey := resolveSearchSettings(r.Context(), s.st, s.webOpts)
	writeJSON(w, http.StatusOK, map[string]any{
		"provider":    provider,
		"base_url":    baseURL,
		"api_key_set": strings.TrimSpace(apiKey) != "",
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
		// LocalMode: keep in-memory defaults in sync for tools without ResolveSearch.
		if s.cfg.LocalMode {
			s.webOpts.SearchProvider = p
		}
	}
	if in.APIKey != nil {
		if err := s.st.SetSysSetting(r.Context(), settingSearchAPIKey, *in.APIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrSaveFailed)
			return
		}
		if s.cfg.LocalMode {
			s.webOpts.SearchAPIKey = *in.APIKey
		}
	}
	if in.BaseURL != nil {
		u := strings.TrimSpace(*in.BaseURL)
		if err := s.st.SetSysSetting(r.Context(), settingSearchBaseURL, u); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrSaveFailed)
			return
		}
		if s.cfg.LocalMode {
			s.webOpts.SearchBaseURL = u
		}
	}
	provider, baseURL, apiKey := resolveSearchSettings(r.Context(), s.st, s.webOpts)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "updated",
		"provider":    provider,
		"base_url":    baseURL,
		"api_key_set": strings.TrimSpace(apiKey) != "",
	})
}
