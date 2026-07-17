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

	"github.com/go-rod/rod"

	"github.com/OptLTD/swiflow/library/browser"
	"github.com/OptLTD/swiflow/library/httputil"
)

// WebOptions configures optional web tools (search backends).
type WebOptions struct {
	SearchProvider string // "", "duckduckgo", "brave", "searxng", "bing", "google"
	SearchBaseURL  string // searxng instance base URL, e.g. https://searx.example
	SearchAPIKey   string // brave (and similar) API key
	// BrowserPool powers bing/google search via a real browser SERP visit.
	BrowserPool    *browser.Pool
	BrowserEnabled bool
}

type searchResult struct {
	Title   string
	URL     string
	Snippet string
}

type webSearchTool struct {
	opts *WebOptions
}

func (t *webSearchTool) Name() string { return "web_search" }
func (t *webSearchTool) Description() string {
	return "Search the web and return titles, URLs, and snippets. " +
		"Providers bing/google open the SERP in the headless browser (requires browser enabled)."
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

	opts := t.opts
	if opts == nil {
		opts = &WebOptions{}
	}
	provider := strings.ToLower(strings.TrimSpace(opts.SearchProvider))
	if provider == "" {
		return "", fmt.Errorf("web search is not configured (set search provider in Settings → System)")
	}

	var (
		results []searchResult
		err     error
	)
	switch provider {
	case "duckduckgo", "ddg":
		results, err = searchDuckDuckGo(ctx, query, limit)
	case "brave":
		results, err = searchBrave(ctx, opts.SearchAPIKey, query, limit)
	case "searxng", "searx":
		results, err = searchSearXNG(ctx, opts.SearchBaseURL, query, limit)
	case "bing":
		results, err = searchBingBrowser(ctx, opts, query, limit)
	case "google":
		results, err = searchGoogleBrowser(ctx, opts, query, limit)
	default:
		return "", fmt.Errorf("unknown search_provider %q (supported: duckduckgo, brave, searxng, bing, google)", provider)
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

// --- Bing / Google via headless browser SERP ---

const browserSearchTimeout = 60 * time.Second

type serpItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

func searchBingBrowser(ctx context.Context, opts *WebOptions, query string, limit int) ([]searchResult, error) {
	if err := requireBrowserSearch(opts, "bing"); err != nil {
		return nil, err
	}
	// form=QBRE mimics an organic search-box submit. Bare /search?q= often gets
	// Bing's bot-poisoned SERP (correct title, unrelated result links).
	// Direct pool (no-proxy-server): system Clash proxy commonly closes Bing.
	rawURL := "https://cn.bing.com/search?q=" + url.QueryEscape(query) + "&form=QBRE&sp=-1"
	results, err := searchViaBrowser(ctx, opts, rawURL, "bing", limit)
	if err == nil && len(results) > 0 {
		return results, nil
	}
	ddg, ddgErr := searchDuckDuckGo(ctx, query, limit)
	if ddgErr == nil && len(ddg) > 0 {
		return ddg, nil
	}
	if err != nil {
		return nil, err
	}
	if ddgErr != nil {
		return nil, fmt.Errorf("bing search failed; duckduckgo fallback: %w", ddgErr)
	}
	return results, nil
}

func searchGoogleBrowser(ctx context.Context, opts *WebOptions, query string, limit int) ([]searchResult, error) {
	if err := requireBrowserSearch(opts, "google"); err != nil {
		return nil, err
	}
	rawURL := "https://www.google.com/search?q=" + url.QueryEscape(query)

	var (
		results []searchResult
		err     error
	)

	// Prefer env/OS or probed local proxy (Clash etc.) when available — direct
	// Google is often unreachable in CN. Bing stays on the main direct pool.
	if proxy := browser.ResolveBrowserProxy(); proxy != "" {
		pool := browser.NewPoolWithProxy(true, proxy)
		defer pool.Close()
		proxyOpts := *opts
		proxyOpts.BrowserPool = pool
		results, err = searchViaBrowser(ctx, &proxyOpts, rawURL, "google", limit)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	} else {
		results, err = searchViaBrowserTimeout(ctx, opts, rawURL, "google", limit, 20*time.Second)
		if err == nil && len(results) > 0 {
			return results, nil
		}
	}

	// Google bot-walls / empty SERP → HTTP DuckDuckGo (httputil may use local proxy).
	ddg, ddgErr := searchDuckDuckGo(ctx, query, limit)
	if ddgErr == nil && len(ddg) > 0 {
		return ddg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("google search failed: %w", err)
	}
	if ddgErr != nil {
		return nil, fmt.Errorf("google search failed; duckduckgo fallback: %w", ddgErr)
	}
	return nil, fmt.Errorf("google search returned no results")
}

func requireBrowserSearch(opts *WebOptions, name string) error {
	if opts == nil || !opts.BrowserEnabled || opts.BrowserPool == nil {
		return fmt.Errorf("%s search uses the headless browser; enable tools.browser_enabled (Settings → Tools)", name)
	}
	return nil
}

func searchViaBrowser(ctx context.Context, opts *WebOptions, rawURL, engine string, limit int) ([]searchResult, error) {
	return searchViaBrowserTimeout(ctx, opts, rawURL, engine, limit, browserSearchTimeout)
}

func searchViaBrowserTimeout(ctx context.Context, opts *WebOptions, rawURL, engine string, limit int, timeout time.Duration) ([]searchResult, error) {
	out, err := opts.BrowserPool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
		if _, err := browser.Open(page, rawURL); err != nil {
			return "", err
		}
		if err := waitSERPReady(page, engine); err != nil {
			return "", err
		}
		info, err := page.Info()
		if err != nil {
			return "", err
		}
		items, err := extractSERP(page, engine, limit)
		if err != nil {
			return "", err
		}
		if len(items) == 0 {
			body, _ := browser.PageText(page, 6000)
			if block, ok := browser.DetectBotBlock(info.Title, info.URL, body); ok {
				return "", block
			}
			return "", fmt.Errorf("%s returned no parseable results (title=%q url=%q)", engine, info.Title, info.URL)
		}
		payload, err := json.Marshal(items)
		if err != nil {
			return "", err
		}
		return string(payload), nil
	})
	if err != nil {
		return nil, err
	}
	var items []serpItem
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("decode %s results: %w", engine, err)
	}
	results := make([]searchResult, 0, len(items))
	for _, it := range items {
		if strings.TrimSpace(it.URL) == "" || strings.TrimSpace(it.Title) == "" {
			continue
		}
		results = append(results, searchResult{
			Title:   strings.TrimSpace(it.Title),
			URL:     strings.TrimSpace(it.URL),
			Snippet: strings.TrimSpace(it.Snippet),
		})
	}
	return results, nil
}

func waitSERPReady(page *rod.Page, engine string) error {
	var sel string
	switch engine {
	case "bing":
		sel = "#b_results li.b_algo, ol#b_results li.b_algo"
	case "google":
		sel = "#search a h3, #rso a h3, div.g a h3"
	default:
		return nil
	}
	_, err := page.Timeout(15 * time.Second).Element(sel)
	if err != nil {
		// Not fatal: extractSERP / bot-block detection still run.
		return nil
	}
	return nil
}

func extractSERP(page *rod.Page, engine string, limit int) ([]serpItem, error) {
	if limit < 1 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}
	var expr string
	switch engine {
	case "bing":
		expr = `(limit) => {
  const out = [];
  const nodes = document.querySelectorAll('#b_results > li.b_algo, ol#b_results li.b_algo');
  for (const el of nodes) {
    if (out.length >= limit) break;
    const a = el.querySelector('h2 a');
    if (!a || !a.href) continue;
    const title = (a.innerText || a.textContent || '').trim();
    if (!title) continue;
    const sn = el.querySelector('.b_caption p, .b_lineclamp2, .b_algoSlug, .b_caption, .b_snippet');
    out.push({
      title,
      url: a.href,
      snippet: sn ? (sn.innerText || sn.textContent || '').trim() : '',
    });
  }
  return out;
}`
	case "google":
		expr = `(limit) => {
  const out = [];
  const seen = new Set();
  const nodes = document.querySelectorAll('#search a h3, #rso a h3, div.g a h3');
  for (const h3 of nodes) {
    if (out.length >= limit) break;
    const a = h3.closest('a');
    if (!a || !a.href) continue;
    const href = a.href;
    if (seen.has(href)) continue;
    if (href.includes('google.') && (href.includes('/search') || href.includes('accounts.google'))) continue;
    const title = (h3.innerText || h3.textContent || '').trim();
    if (!title) continue;
    seen.add(href);
    let snippet = '';
    const root = a.closest('div.g, div[data-sokoban-container], div[data-hveid]') || a.parentElement;
    if (root) {
      const sn = root.querySelector('div[data-sncf], div[style*="-webkit-line-clamp"], span[class*="st"], div.VwiC3b');
      if (sn) snippet = (sn.innerText || sn.textContent || '').trim();
    }
    out.push({ title, url: href, snippet });
  }
  return out;
}`
	default:
		return nil, fmt.Errorf("unknown browser search engine %q", engine)
	}
	res, err := page.Eval(expr, limit)
	if err != nil {
		return nil, fmt.Errorf("extract %s results: %w", engine, err)
	}
	var items []serpItem
	if err := res.Value.Unmarshal(&items); err != nil {
		return nil, fmt.Errorf("decode %s extract: %w", engine, err)
	}
	return items, nil
}
