// Browser automation via go-rod (headless Chromium). Gated by config.
package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"

	"github.com/OptLTD/swiflow/internal/browser"
	"github.com/OptLTD/swiflow/internal/secure"
	"github.com/OptLTD/swiflow/internal/util"
)

const ToolBrowser = "browser"

var browserToolNames = []string{ToolBrowser}

// BrowserToolNames returns browser tool names.
func BrowserToolNames() []string {
	return append([]string(nil), browserToolNames...)
}

// IsBrowserTool reports whether name is the browser tool.
func IsBrowserTool(name string) bool {
	return name == ToolBrowser
}

type browserTool struct {
	pool    *browser.Pool
	ws      WorkspaceRoots
	allowed bool
}

func (t *browserTool) Name() string { return ToolBrowser }
func (t *browserTool) Description() string {
	return "Control a headless browser: navigate, read page text, screenshot, click, type, or evaluate JavaScript."
}
func (t *browserTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": []string{"navigate", "content", "screenshot", "click", "type", "eval"},
				"description": "navigate: open URL; content: page text; screenshot: PNG to workspace; click/type: interact with selector; eval: run JS.",
			},
			"url": map[string]any{
				"type":        "string",
				"description": "URL for navigate (http/https only).",
			},
			"selector": map[string]any{
				"type":        "string",
				"description": "CSS selector for click or type. For click, optional when text is set (defaults to a,button,[role=button],input[type=button],input[type=submit]).",
			},
			"text": map[string]any{
				"type":        "string",
				"description": "For type: text to input. For click: match element by visible text (substring).",
			},
			"expression": map[string]any{
				"type":        "string",
				"description": "JavaScript expression for eval (return value is serialized).",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Screenshot filename under workspace/browser/ (default: shot-<id>.png).",
			},
			"max_chars": map[string]any{
				"type":        "integer",
				"description": "Max characters for navigate/content text (default 20000).",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Timeout seconds (1–120, default 30).",
			},
		},
		"required": []string{"action"},
	}
}

func (t *browserTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	if !t.allowed {
		return "", fmt.Errorf("browser is disabled (set tools.browser_enabled or SWIFLOW_BROWSER=true)")
	}
	if t.pool == nil {
		return "", fmt.Errorf("browser pool not initialized")
	}
	action := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", args["action"])))
	timeout := browserTimeout(args)
	maxChars := browserMaxChars(args)

	switch action {
	case "navigate":
		url, _ := args["url"].(string)
		if url == "" {
			return "", fmt.Errorf("url is required for navigate")
		}
		if err := secure.CheckURL(url); err != nil {
			return "", err
		}
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			note, err := browser.Open(page, url)
			if err != nil {
				return "", err
			}
			info, err := page.Info()
			if err != nil {
				return "", err
			}
			body, err := browser.PageText(page, maxChars)
			if err != nil {
				return "", err
			}
			out := fmt.Sprintf("title: %s\nurl: %s\n\n%s", info.Title, info.URL, body)
			if note != "" {
				out = note + "\n\n" + out
			} else if block, ok := browser.DetectBotBlock(info.Title, info.URL, body); ok {
				return "", fmt.Errorf("%s", block.Error())
			}
			return out, nil
		})
	case "content":
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			if err := browser.WaitLoaded(page); err != nil {
				return "", err
			}
			info, err := page.Info()
			if err != nil {
				return "", err
			}
			body, err := browser.PageText(page, maxChars)
			if err != nil {
				return "", err
			}
			out := fmt.Sprintf("title: %s\nurl: %s\n\n%s", info.Title, info.URL, body)
			if block, ok := browser.DetectBotBlock(info.Title, info.URL, body); ok {
				return "", fmt.Errorf("%s", block.Error())
			}
			return out, nil
		})
	case "screenshot":
		name, _ := args["filename"].(string)
		if name == "" {
			name = "shot-" + util.NewID() + ".png"
		}
		if !strings.HasSuffix(strings.ToLower(name), ".png") {
			name += ".png"
		}
		rel := filepath.Join("browser", filepath.Base(name))
		full, err := secure.SandboxPath(t.ws.Base, rel)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return "", err
		}
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			if err := browser.WaitLoaded(page); err != nil {
				return "", err
			}
			data, err := page.Screenshot(true, &proto.PageCaptureScreenshot{
				Format: proto.PageCaptureScreenshotFormatPng,
			})
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(full, data, 0o644); err != nil {
				return "", err
			}
			return fmt.Sprintf("screenshot saved to %s (%d bytes)", rel, len(data)), nil
		})
	case "click":
		sel, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		if sel == "" && text == "" {
			return "", fmt.Errorf("selector or text is required for click")
		}
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			el, label, err := findClickTarget(page, sel, text)
			if err != nil {
				return "", err
			}
			// DOM click bypasses rod's WaitInteractable (hidden/covered nav links
			// otherwise burn the full timeout as "context deadline exceeded").
			if _, err := el.Eval(`() => { this.click(); return true }`); err != nil {
				if err2 := el.Click(proto.InputMouseButtonLeft, 1); err2 != nil {
					return "", fmt.Errorf("click %s: %w", label, wrapBrowserTimeout(err2))
				}
			}
			_ = page.Timeout(3 * time.Second).WaitStable(200 * time.Millisecond)
			info, _ := page.Info()
			out := "clicked " + label
			if info != nil && info.URL != "" {
				out += "\nurl: " + info.URL
			}
			return out, nil
		})
	case "type":
		sel, _ := args["selector"].(string)
		text, _ := args["text"].(string)
		if sel == "" {
			return "", fmt.Errorf("selector is required for type")
		}
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			el, err := page.Element(sel)
			if err != nil {
				return "", wrapBrowserTimeout(err)
			}
			if err := el.Input(text); err != nil {
				return "", wrapBrowserTimeout(err)
			}
			return fmt.Sprintf("typed into %s", sel), nil
		})
	case "eval":
		expr, _ := args["expression"].(string)
		if expr == "" {
			return "", fmt.Errorf("expression is required for eval")
		}
		return t.pool.WithPage(ctx, timeout, func(page *rod.Page) (string, error) {
			res, err := page.Eval(expr)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%v", res.Value), nil
		})
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
}

func browserTimeout(args map[string]any) time.Duration {
	sec := 30
	if v, ok := args["timeout"].(float64); ok && v > 0 {
		sec = int(v)
	}
	if sec < 1 {
		sec = 1
	}
	if sec > 120 {
		sec = 120
	}
	return time.Duration(sec) * time.Second
}

func browserMaxChars(args map[string]any) int {
	max := 20000
	if v, ok := args["max_chars"].(float64); ok && v > 0 {
		max = int(v)
	}
	return max
}

const defaultClickSelector = "a,button,[role=button],input[type=button],input[type=submit]"

func findClickTarget(page *rod.Page, sel, text string) (*rod.Element, string, error) {
	if sel == "" {
		sel = defaultClickSelector
	}
	label := sel
	if text != "" {
		label = fmt.Sprintf("%s text=%q", sel, text)
	}

	var (
		el  *rod.Element
		err error
	)
	if text != "" {
		el, err = page.ElementR(sel, jsLiteralRegex(text))
	} else {
		el, err = page.Element(sel)
	}
	if err != nil {
		return nil, "", fmt.Errorf("element not found (%s): %w", label, wrapBrowserTimeout(err))
	}
	return el, label, nil
}

// jsLiteralRegex builds a JS /.../i pattern for rod ElementR.
func jsLiteralRegex(s string) string {
	escaped := regexp.QuoteMeta(s)
	escaped = strings.ReplaceAll(escaped, "/", `\/`)
	return "/" + escaped + "/i"
}

func wrapBrowserTimeout(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded") {
		return fmt.Errorf("%w (element missing, hidden, or covered — try a tighter selector, text match, or navigate to the href directly)", err)
	}
	return err
}

// BrowserOptions configures the browser tool.
type BrowserOptions struct {
	Enabled  bool
	Headless bool
}

// RegisterBrowser registers the browser tool.
func RegisterBrowser(r *Registry, ws WorkspaceRoots, pool *browser.Pool, opt BrowserOptions) {
	r.Register(&browserTool{pool: pool, ws: ws, allowed: opt.Enabled})
	if !opt.Enabled {
		r.SetEnabled(ToolBrowser, false)
	}
}
