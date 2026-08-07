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
	proxy    string // optional Chrome --proxy-server value
	launch   *launcherHolder
	browser  *rod.Browser
	page     *rod.Page            // shared page when tid is empty (LocalMode)
	pages    map[string]*rod.Page // per-tenant pages
}

type launcherHolder struct {
	l *launcher.Launcher
}

// NewPool creates a browser pool (direct connection). Call Close on shutdown.
func NewPool(headless bool) *Pool {
	return &Pool{headless: headless, pages: map[string]*rod.Page{}}
}

// NewPoolWithProxy creates a browser pool that launches Chrome with proxy.
func NewPoolWithProxy(headless bool, proxy string) *Pool {
	return &Pool{headless: headless, proxy: proxy, pages: map[string]*rod.Page{}}
}

// Close shuts down the browser and launcher.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cleanupLocked()
}

func (p *Pool) cleanupLocked() {
	if p.pages != nil {
		for tid, pg := range p.pages {
			_ = pg.Close()
			delete(p.pages, tid)
		}
	}
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

	l := buildLauncher(p.headless, p.proxy).Context(launchCtx)
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("launch browser: %w (install Google Chrome or Microsoft Edge, or set CHROME_PATH)", err)
	}

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
	if p.pages == nil {
		p.pages = map[string]*rod.Page{}
	}
	return nil
}

func (p *Pool) pageFor(ctx context.Context, tid string) (*rod.Page, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensureBrowser(); err != nil {
		return nil, err
	}
	if tid == "" {
		return p.page.Context(ctx), nil
	}
	pg := p.pages[tid]
	if pg != nil {
		// Detect dead pages and recreate.
		if _, err := pg.Info(); err == nil {
			return pg.Context(ctx), nil
		}
		_ = pg.Close()
		delete(p.pages, tid)
	}
	pg, err := stealth.Page(p.browser)
	if err != nil {
		return nil, fmt.Errorf("stealth page for tenant: %w", err)
	}
	if err := prepareStealthPage(pg); err != nil {
		_ = pg.Close()
		return nil, fmt.Errorf("prepare tenant page: %w", err)
	}
	p.pages[tid] = pg
	return pg.Context(ctx), nil
}

// WithPage runs fn with the shared page bound to a timeout context.
func (p *Pool) WithPage(ctx context.Context, timeout time.Duration, fn func(*rod.Page) (string, error)) (string, error) {
	return p.WithPageTenant(ctx, "", timeout, fn)
}

// WithPageTenant runs fn with a page for tid. Empty tid uses the shared page
// (LocalMode). Non-empty tid reuses a stealth page per tenant from the same browser.
func (p *Pool) WithPageTenant(ctx context.Context, tid string, timeout time.Duration, fn func(*rod.Page) (string, error)) (string, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	page, err := p.pageFor(cctx, tid)
	if err != nil {
		return "", err
	}
	return fn(page)
}
