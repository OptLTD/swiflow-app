package browser

import (
	"os"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/OptLTD/swiflow/library/httputil"
)

// buildLauncher returns a Chrome launcher with stability and anti-automation flags.
// proxy is optional (env/OS or probed local); empty means direct connection.
func buildLauncher(headless bool, proxy string) *launcher.Launcher {
	l := launcher.New().
		Leakless(true).
		Set("disable-gpu").
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("disable-dev-shm-usage").
		Set("disable-software-rasterizer").
		Set("disable-extensions").
		Set("disable-background-networking").
		Set("disable-renderer-backgrounding").
		Set("disable-background-timer-throttling").
		Set("disable-backgrounding-occluded-windows").
		Set("disable-blink-features", "AutomationControlled").
		Set("excludeSwitches", "enable-automation").
		Set("lang", "zh-CN,zh,en-US,en")

	// Prefer a local Chrome/Edge/Chromium. Auto-download often fails on Windows
	// (CDN blocked, or unsupported GOARCH like windows/arm64 in rod's host map).
	if bin := resolveBrowserBin(); bin != "" {
		l = l.Bin(bin)
	}
	if wantBrowserNoSandbox() {
		l = l.NoSandbox(true)
	}

	if proxy != "" {
		l = l.Proxy(proxy)
	} else {
		// Chromium on macOS/Windows picks up the OS system proxy by default.
		// That breaks domestic SERPs (cn.bing.com) when Clash is the system
		// proxy — force a direct connection for the default pool.
		l = l.Set("no-proxy-server")
	}

	if headless {
		l = l.HeadlessNew(true)
	} else {
		l = l.Headless(false)
	}
	return l
}

func resolveBrowserBin() string {
	for _, key := range []string{"CHROME_PATH", "CHROMIUM_PATH", "PUPPETEER_EXECUTABLE_PATH"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	if path, ok := launcher.LookPath(); ok {
		return path
	}
	return ""
}

func wantBrowserNoSandbox() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SWIFLOW_BROWSER_NO_SANDBOX"))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	// Default on inside Docker / rootless containers without a sandbox profile.
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

// ResolveBrowserProxy returns env/OS proxy when reachable, otherwise a probed
// local HTTP proxy. Empty means launch Chrome with a direct connection.
func ResolveBrowserProxy() string {
	if proxy := httputil.ProxyServer(); proxy != "" {
		return proxy
	}
	return httputil.LocalProxyServer()
}

func prepareStealthPage(page *rod.Page) error {
	device := devices.LaptopWithMDPIScreen
	if err := page.SetUserAgent(device.UserAgentEmulation()); err != nil {
		return err
	}
	if err := page.SetViewport(device.MetricsEmulation()); err != nil {
		return err
	}
	tz := proto.EmulationSetTimezoneOverride{TimezoneID: "Asia/Shanghai"}
	if err := tz.Call(page); err != nil {
		return err
	}
	loc := proto.EmulationSetLocaleOverride{Locale: "zh-CN"}
	if err := loc.Call(page); err != nil {
		return err
	}
	locale := acceptLanguages("zh-CN,zh,en-US,en")
	_, err := page.SetExtraHeaders([]string{"Accept-Language", locale})
	return err
}

func acceptLanguages(lang string) string {
	if lang == "" {
		return "en-US,en;q=0.9"
	}
	if strings.Contains(lang, ",") {
		return lang
	}
	return lang + ",en;q=0.9"
}
