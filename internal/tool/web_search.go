package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/internal/httputil"
)

// WebOptions configures optional web tools (search backends).
type WebOptions struct {
	SearchProvider string // "", "duckduckgo", "brave", "searxng"
	SearchBaseURL  string // searxng instance base URL, e.g. https://searx.example
	SearchAPIKey   string // brave (and similar) API key
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

type webSearchTool struct {
	opts WebOptions
}

func (t *webSearchTool) Name() string { return "web_search" }
func (t *webSearchTool) Description() string {
	return "Search the web and return titles, URLs, and snippets."
}
func (t *webSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string", "description": "Search query."},
			"limit": map[string]any{"type": "integer", "default": 5, "description": "Max results (1-10)."},
		},
		"required": []string{"query"},
	}
}

func (t *webSearchTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	query, _ := args["query"].(string)
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("query required")
	}
	limit := 5
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	if limit > 10 {
		limit = 10
	}

	provider := strings.ToLower(strings.TrimSpace(t.opts.SearchProvider))
	if provider == "" {
		return "", fmt.Errorf("web search is not configured (set tools.web_search_provider, e.g. duckduckgo)")
	}

	var (
		results []searchResult
		err     error
	)
	switch provider {
	case "duckduckgo", "ddg":
		results, err = searchDuckDuckGo(ctx, query, limit)
	case "brave":
		results, err = searchBrave(ctx, t.opts.SearchAPIKey, query, limit)
	case "searxng", "searx":
		results, err = searchSearXNG(ctx, t.opts.SearchBaseURL, query, limit)
	default:
		return "", fmt.Errorf("unknown search_provider %q (supported: duckduckgo, brave, searxng)", provider)
	}
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "No results found.", nil
	}
	return formatSearchResults(results), nil
}

func formatSearchResults(results []searchResult) string {
	var b strings.Builder
	for i, r := range results {
		fmt.Fprintf(&b, "%d. %s\n   %s\n", i+1, r.Title, r.URL)
		if r.Snippet != "" {
			fmt.Fprintf(&b, "   %s\n", r.Snippet)
		}
		if i < len(results)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func httpGetJSON(ctx context.Context, rawURL string, headers map[string]string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Swiflow/1.0")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httputil.Do(req, 15*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncateErrBody(body))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func truncateErrBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// --- DuckDuckGo (HTML lite / html endpoint, no API key) ---

var (
	ddgResultBlockRe = regexp.MustCompile(`(?s)<div[^>]*class="[^"]*result[^"]*"[^>]*>.*?</div>\s*</div>`)
	ddgLinkRe        = regexp.MustCompile(`(?i)<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	ddgSnippetRe     = regexp.MustCompile(`(?i)<(?:a|td)[^>]*class="[^"]*result__snippet[^"]*"[^>]*>(.*?)</(?:a|td)>`)
	ddgTagRe         = regexp.MustCompile(`<[^>]*>`)
)

func searchDuckDuckGo(ctx context.Context, query string, limit int) ([]searchResult, error) {
	var errs []string

	// Try HTML endpoints first (full SERP). Network failures fall through.
	for _, endpoint := range []string{ddgHTMLEndpoint, ddgLiteEndpoint} {
		results, err := fetchDuckDuckGoHTML(ctx, endpoint, query, limit)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if len(results) > 0 {
			return results, nil
		}
	}

	// Fallback: Instant Answer API (less blocked; not full SERP).
	results, err := searchDuckDuckGoInstant(ctx, query, limit)
	if err != nil {
		errs = append(errs, err.Error())
		return nil, fmt.Errorf("duckduckgo unavailable (%s)", strings.Join(errs, "; "))
	}
	if len(results) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("duckduckgo returned no results after: %s", strings.Join(errs, "; "))
	}
	return results, nil
}

func fetchDuckDuckGoHTML(ctx context.Context, endpoint, query string, limit int) ([]searchResult, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := httputil.Do(req, 15*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("duckduckgo http %d", resp.StatusCode)
	}
	return parseDuckDuckGoHTML(string(body), limit), nil
}

func parseDuckDuckGoHTML(html string, limit int) []searchResult {
	blocks := ddgResultBlockRe.FindAllString(html, -1)
	if len(blocks) == 0 {
		// Fallback: match links globally when block regex misses.
		blocks = []string{html}
	}
	out := make([]searchResult, 0, limit)
	seen := map[string]bool{}
	for _, block := range blocks {
		m := ddgLinkRe.FindStringSubmatch(block)
		if m == nil {
			continue
		}
		href := unwrapDuckDuckGoURL(htmlUnescape(m[1]))
		title := strings.TrimSpace(ddgTagRe.ReplaceAllString(htmlUnescape(m[2]), ""))
		if href == "" || title == "" || seen[href] {
			continue
		}
		if strings.Contains(href, "duckduckgo.com") && !strings.HasPrefix(href, "http") {
			continue
		}
		snippet := ""
		if sm := ddgSnippetRe.FindStringSubmatch(block); sm != nil {
			snippet = strings.TrimSpace(ddgTagRe.ReplaceAllString(htmlUnescape(sm[1]), ""))
			snippet = wsRe.ReplaceAllString(snippet, " ")
		}
		seen[href] = true
		out = append(out, searchResult{Title: title, URL: href, Snippet: snippet})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func unwrapDuckDuckGoURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if uddg := u.Query().Get("uddg"); uddg != "" {
		if decoded, err := url.QueryUnescape(uddg); err == nil {
			return decoded
		}
		return uddg
	}
	return href
}

func htmlUnescape(s string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
		"&nbsp;", " ",
	)
	return replacer.Replace(s)
}

type ddgInstantResp struct {
	AbstractText   string       `json:"AbstractText"`
	AbstractURL    string       `json:"AbstractURL"`
	AbstractSource string       `json:"AbstractSource"`
	Heading        string       `json:"Heading"`
	Answer         string       `json:"Answer"`
	Definition     string       `json:"Definition"`
	DefinitionURL  string       `json:"DefinitionURL"`
	RelatedTopics  []ddgRelated `json:"RelatedTopics"`
	Results        []ddgRelated `json:"Results"`
}

type ddgRelated struct {
	Text     string       `json:"Text"`
	FirstURL string       `json:"FirstURL"`
	Topics   []ddgRelated `json:"Topics"`
}

func searchDuckDuckGoInstant(ctx context.Context, query string, limit int) ([]searchResult, error) {
	u := ddgInstantEndpoint + "?q=" + url.QueryEscape(query) +
		"&format=json&no_html=1&skip_disambig=1&t=swiflow"
	var resp ddgInstantResp
	if err := httpGetJSON(ctx, u, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]searchResult, 0, limit)
	add := func(title, link, snippet string) {
		link = strings.TrimSpace(link)
		title = strings.TrimSpace(title)
		if link == "" || title == "" {
			return
		}
		for _, existing := range out {
			if existing.URL == link {
				return
			}
		}
		out = append(out, searchResult{Title: title, URL: link, Snippet: strings.TrimSpace(snippet)})
	}
	if resp.AbstractText != "" && resp.AbstractURL != "" {
		title := resp.Heading
		if title == "" {
			title = resp.AbstractSource
		}
		if title == "" {
			title = query
		}
		add(title, resp.AbstractURL, resp.AbstractText)
	}
	if resp.Answer != "" {
		add(query+" (answer)", resp.AbstractURL, resp.Answer)
	}
	if resp.Definition != "" && resp.DefinitionURL != "" {
		add(query+" (definition)", resp.DefinitionURL, resp.Definition)
	}
	var walk func([]ddgRelated)
	walk = func(items []ddgRelated) {
		for _, it := range items {
			if len(out) >= limit {
				return
			}
			if len(it.Topics) > 0 {
				walk(it.Topics)
				continue
			}
			if it.FirstURL == "" {
				continue
			}
			title := it.Text
			snippet := ""
			if i := strings.Index(title, " - "); i > 0 {
				snippet = strings.TrimSpace(title[i+3:])
				title = strings.TrimSpace(title[:i])
			}
			if title == "" {
				title = it.FirstURL
			}
			add(title, it.FirstURL, snippet)
		}
	}
	walk(resp.Results)
	walk(resp.RelatedTopics)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// --- Brave Search API ---

type braveSearchResp struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

// overridable in tests
var (
	braveSearchEndpoint = "https://api.search.brave.com/res/v1/web/search"
	ddgHTMLEndpoint     = "https://html.duckduckgo.com/html/"
	ddgLiteEndpoint     = "https://lite.duckduckgo.com/lite/"
	ddgInstantEndpoint  = "https://api.duckduckgo.com/"
)

func searchBrave(ctx context.Context, apiKey, query string, limit int) ([]searchResult, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("brave web search requires tools.web_search_api_key (or SWIFLOW_WEB_SEARCH_API_KEY)")
	}
	u := braveSearchEndpoint + "?q=" + url.QueryEscape(query) +
		"&count=" + strconv.Itoa(limit)
	var resp braveSearchResp
	if err := httpGetJSON(ctx, u, map[string]string{
		"X-Subscription-Token": apiKey,
		"Accept":               "application/json",
	}, &resp); err != nil {
		return nil, err
	}
	out := make([]searchResult, 0, limit)
	for _, r := range resp.Web.Results {
		if r.URL == "" {
			continue
		}
		title := r.Title
		if title == "" {
			title = r.URL
		}
		out = append(out, searchResult{Title: title, URL: r.URL, Snippet: r.Description})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// --- SearXNG ---

type searxResp struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

func searchSearXNG(ctx context.Context, baseURL, query string, limit int) ([]searchResult, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("searxng web search requires tools.web_search_url")
	}
	u := baseURL + "/search?q=" + url.QueryEscape(query) + "&format=json"
	var resp searxResp
	if err := httpGetJSON(ctx, u, nil, &resp); err != nil {
		return nil, err
	}
	out := make([]searchResult, 0, limit)
	for _, r := range resp.Results {
		if r.URL == "" {
			continue
		}
		title := r.Title
		if title == "" {
			title = r.URL
		}
		out = append(out, searchResult{Title: title, URL: r.URL, Snippet: r.Content})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
