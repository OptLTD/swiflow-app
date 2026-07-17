package browser

import (
	"time"

	"github.com/go-rod/rod"
)

// WaitLoaded waits for navigation and a short stability window.
// WaitStable is hard-capped: ad-heavy SERPs (e.g. Bing) never go fully idle and
// would otherwise burn the whole page context as "context deadline exceeded".
func WaitLoaded(page *rod.Page) error {
	if err := page.Timeout(20 * time.Second).WaitLoad(); err != nil {
		// DOM may already be usable after a slow/partial load; keep going.
		_ = err
	}
	_ = page.Timeout(3 * time.Second).WaitStable(400 * time.Millisecond)
	return nil
}

// PageText returns body text, truncated to max runes when max > 0.
func PageText(page *rod.Page, max int) (string, error) {
	el, err := page.Element("body")
	if err != nil {
		return "", err
	}
	text, err := el.Text()
	if err != nil {
		return "", err
	}
	if max > 0 && len(text) > max {
		text = text[:max] + "\n...[truncated]"
	}
	return text, nil
}
