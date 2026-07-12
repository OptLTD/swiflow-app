package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
)

// Pool keeps a single Chromium instance for browser automation.
type Pool struct {
	mu       sync.Mutex
	headless bool
	launch   *launcherHolder
	browser  *rod.Browser
	page     *rod.Page
}

type launcherHolder struct {
	l *launcher.Launcher
}

// NewPool creates a browser pool. Call Close on shutdown.
func NewPool(headless bool) *Pool {
	return &Pool{headless: headless}
}

// Close shuts down the browser and launcher.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupLocked()
}

func (p *Pool) cleanupLocked() {
	if p.browser != nil {
		_ = p.browser.Close()
		p.browser = nil
		p.page = nil
	}
	if p.launch != nil {
		p.launch.l.Kill()
		p.launch.l.Cleanup()
		p.launch = nil
	}
}

func (p *Pool) ensureBrowser() error {
	if p.browser != nil {
		if _, err := p.browser.Pages(); err == nil {
			return nil
		}
		p.cleanupLocked()
	}

	launchCtx, launchCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer launchCancel()

	l := buildLauncher(p.headless).Context(launchCtx)
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}

	// Long-lived connection — do not bind browser to a request context (goclaw pattern).
	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		l.Kill()
		l.Cleanup()
		return fmt.Errorf("connect browser: %w", err)
	}

	page, err := stealth.Page(browser)
	if err != nil {
		_ = browser.Close()
		l.Kill()
		l.Cleanup()
		return fmt.Errorf("stealth page: %w", err)
	}
	if err := prepareStealthPage(page); err != nil {
		_ = browser.Close()
		l.Kill()
		l.Cleanup()
		return fmt.Errorf("prepare page: %w", err)
	}

	p.launch = &launcherHolder{l: l}
	p.browser = browser
	p.page = page
	return nil
}

func (p *Pool) pageFor(ctx context.Context) (*rod.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureBrowser(); err != nil {
		return nil, err
	}
	return p.page.Context(ctx), nil
}

// WithPage runs fn with a page bound to a timeout context.
func (p *Pool) WithPage(ctx context.Context, timeout time.Duration, fn func(*rod.Page) (string, error)) (string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	page, err := p.pageFor(cctx)
	if err != nil {
		return "", err
	}
	return fn(page)
}
