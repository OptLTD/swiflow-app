package browser

import (
	"time"

	"github.com/go-rod/rod"
)

// WaitLoaded waits for navigation and a short stability window.
func WaitLoaded(page *rod.Page) error {
	if err := page.WaitLoad(); err != nil {
		return err
	}
	_ = page.WaitStable(500 * time.Millisecond)
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
