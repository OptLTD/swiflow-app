package browser

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// Open navigates to rawURL. Blocked Google searches fall back to DuckDuckGo; the
// returned note is non-empty when a fallback was used.
func Open(page *rod.Page, rawURL string) (note string, err error) {
	if query, ok := googleSearchQuery(rawURL); ok {
		if err := googleSearch(page, query, rawURL); err != nil {
			return "", err
		}
		if blocked, _ := pageBotBlock(page); blocked {
			ddg := duckDuckGoSearchURL(query)
			if err := page.Navigate(ddg); err != nil {
				return "", fmt.Errorf("google blocked and duckduckgo fallback failed: %w", err)
			}
			if err := WaitLoaded(page); err != nil {
				return "", err
			}
			return "Google blocked automated search; showing DuckDuckGo results instead.", nil
		}
		return "", nil
	}
	if err := Navigate(page, rawURL); err != nil {
		return "", err
	}
	return "", nil
}

// Navigate opens a URL with optional warm-up for Google search pages.
func Navigate(page *rod.Page, rawURL string) error {
	if query, ok := googleSearchQuery(rawURL); ok {
		return googleSearch(page, query, rawURL)
	}
	if warmup, ok := googleWarmupURL(rawURL); ok {
		if err := page.Navigate(warmup); err != nil {
			return err
		}
		if err := WaitLoaded(page); err != nil {
			return err
		}
		_ = page.WaitStable(time.Second)
	}
	if err := page.Navigate(rawURL); err != nil {
		return err
	}
	return WaitLoaded(page)
}

func googleSearchQuery(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if !strings.Contains(host, "google.") {
		return "", false
	}
	path := strings.ToLower(u.Path)
	if path != "/search" && !strings.HasPrefix(path, "/search/") {
		return "", false
	}
	q := u.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		return "", false
	}
	return q, true
}

func googleSearch(page *rod.Page, query, rawURL string) error {
	home := "https://www.google.com/"
	if u, err := url.Parse(rawURL); err == nil {
		host := strings.ToLower(u.Hostname())
		if strings.HasSuffix(host, ".com.hk") || strings.HasSuffix(host, ".hk") {
			home = "https://www.google.com.hk/"
		}
	}
	if err := page.Navigate(home); err != nil {
		return err
	}
	if err := WaitLoaded(page); err != nil {
		return err
	}
	_ = page.WaitStable(800 * time.Millisecond)

	box, err := page.Timeout(10 * time.Second).Element("textarea[name='q'], input[name='q']")
	if err != nil {
		// Fall back to direct URL if the search box is unavailable.
		if err := page.Navigate(rawURL); err != nil {
			return err
		}
		return WaitLoaded(page)
	}
	if err := box.SelectAllText(); err != nil {
		_ = box.Click(proto.InputMouseButtonLeft, 1)
	}
	if err := box.Input(query); err != nil {
		return fmt.Errorf("type google query: %w", err)
	}
	if err := page.Keyboard.Press(input.Enter); err != nil {
		return fmt.Errorf("submit google query: %w", err)
	}
	return WaitLoaded(page)
}

func duckDuckGoSearchURL(query string) string {
	return "https://duckduckgo.com/?q=" + url.QueryEscape(query)
}

func pageBotBlock(page *rod.Page) (bool, BotBlockInfo) {
	info, err := page.Info()
	if err != nil {
		return false, BotBlockInfo{}
	}
	body, err := PageText(page, 8000)
	if err != nil {
		return false, BotBlockInfo{}
	}
	block, ok := DetectBotBlock(info.Title, info.URL, body)
	return ok, block
}

func googleWarmupURL(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if !strings.Contains(host, "google.") {
		return "", false
	}
	path := strings.ToLower(u.Path)
	if path != "/search" && !strings.HasPrefix(path, "/search/") {
		return "", false
	}
	// Visit regional homepage first so cookies/referrer look less automated.
	if strings.HasSuffix(host, ".com.hk") || strings.HasSuffix(host, ".hk") {
		return "https://www.google.com.hk/", true
	}
	return "https://www.google.com/", true
}

// BotBlockInfo summarizes an anti-bot interstitial when detected.
type BotBlockInfo struct {
	Title   string
	PageURL string
	Reason  string
}

// DetectBotBlock inspects a navigated page for common bot walls.
func DetectBotBlock(title, pageURL, body string) (BotBlockInfo, bool) {
	combined := strings.ToLower(title + "\n" + pageURL + "\n" + body)
	if strings.Contains(pageURL, "/sorry/") || strings.Contains(combined, "/sorry/index") {
		return BotBlockInfo{
			Title:   title,
			PageURL: pageURL,
			Reason:  "redirected to Google /sorry/ challenge",
		}, true
	}
	if !LooksLikeBotBlock(combined) {
		return BotBlockInfo{}, false
	}
	reason := "anti-bot interstitial detected"
	if strings.Contains(combined, "unusual traffic") {
		reason = "unusual traffic challenge"
	}
	return BotBlockInfo{Title: title, PageURL: pageURL, Reason: reason}, true
}

func (b BotBlockInfo) Error() string {
	if b.PageURL != "" {
		return fmtBotBlock(b.Reason, b.Title, b.PageURL)
	}
	return fmtBotBlock(b.Reason, b.Title, "")
}

func fmtBotBlock(reason, title, pageURL string) string {
	msg := "page looks like a bot challenge (" + reason
	if title != "" {
		msg += "; title: " + title
	}
	if pageURL != "" {
		msg += "; url: " + pageURL
	}
	return msg + "). Try browser_headless=false or reduce automated search frequency"
}
