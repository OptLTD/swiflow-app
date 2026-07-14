package browser

import (
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// buildLauncher returns a Chrome launcher with stability and anti-automation flags.
func buildLauncher(headless bool) *launcher.Launcher {
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

	if headless {
		// New headless mode is closer to real Chrome and harder to fingerprint.
		l = l.HeadlessNew(true)
	} else {
		l = l.Headless(false)
	}
	return l
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
