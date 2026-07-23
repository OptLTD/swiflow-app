package main

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"

	"github.com/OptLTD/swiflow/internal/version"
)

const updateRepository = "OptLTD/swiflow-app"

// UpdateCheckResult is returned by silent update detection (no window).
type UpdateCheckResult struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest,omitempty"`
	Name      string `json:"name,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Error     string `json:"error,omitempty"`
}

// UpdateService exposes update checks to the desktop frontend.
type UpdateService struct {
	app *application.App
}

// CurrentVersion returns the running desktop build version.
func (s *UpdateService) CurrentVersion() string {
	return version.Version
}

// CheckLatest silently queries GitHub Releases (no update window).
func (s *UpdateService) CheckLatest() UpdateCheckResult {
	res := UpdateCheckResult{Current: version.Version}
	if s == nil || s.app == nil || s.app.Updater == nil {
		res.Error = "updater unavailable"
		return res
	}
	rel, err := s.app.Updater.Check(context.Background())
	if err != nil {
		res.Error = err.Error()
		return res
	}
	if rel == nil {
		return res
	}
	res.Available = true
	res.Latest = rel.Version
	res.Name = rel.Name
	res.Notes = rel.Notes
	return res
}

// CheckForUpdates opens the Wails update window and runs the install flow.
func (s *UpdateService) CheckForUpdates() {
	if s == nil || s.app == nil || s.app.Updater == nil {
		return
	}
	go func() {
		if err := s.app.Updater.CheckAndInstall(context.Background()); err != nil {
			slog.Error("update flow", "error", err)
		}
	}()
}

// setupUpdater wires GitHub Releases self-update, menu, and silent polling.
func setupUpdater(app *application.App) {
	gh, err := github.New(github.Config{
		Repository:    updateRepository,
		ChecksumAsset: "SHA256SUMS",
		// Release assets omit OS names (e.g. Swiflow-1.0.0-arm64.zip / -amd64.exe);
		// match by extension + arch instead of DefaultAssetMatcher's "darwin"/"windows" tokens.
		AssetMatcher: swiflowAssetMatcher,
	})
	if err != nil {
		slog.Error("updater github provider", "error", err)
		return
	}
	// No CheckInterval: framework interval uses CheckAndInstall (pops a window).
	// We poll with silent Check() instead so the UI can show a badge first.
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version.Version,
		Providers:      []updater.Provider{gh},
	}); err != nil {
		slog.Error("updater init", "error", err)
		return
	}
	slog.Info("updater ready", "version", version.Version, "repo", updateRepository)
	setupUpdateMenu(app)
	go silentUpdatePoll(app)
}

// swiflowAssetMatcher picks update payloads by file extension + arch.
// macOS → .zip/.dmg, Windows → .exe (skip *installer*), Linux → .tar.gz/.tgz/.AppImage.
func swiflowAssetMatcher(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	plat := strings.ToLower(req.Platform)
	arch := strings.ToLower(req.Arch)
	for i, a := range assets {
		name := strings.ToLower(a.Name)
		if strings.HasSuffix(name, ".sig") || strings.HasSuffix(name, ".asc") {
			continue
		}
		if name == "sha256sums" || strings.HasSuffix(name, ".sums") || strings.HasSuffix(name, ".sha256") {
			continue
		}
		if strings.Contains(name, "installer") {
			continue
		}
		if arch != "" && !assetMatchesArch(name, arch) {
			continue
		}
		switch plat {
		case "darwin":
			if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".dmg") {
				return i
			}
		case "windows":
			if strings.HasSuffix(name, ".exe") {
				return i
			}
		case "linux":
			if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz") ||
				strings.HasSuffix(name, ".appimage") {
				return i
			}
		default:
			return i
		}
	}
	return -1
}

func assetMatchesArch(name, arch string) bool {
	if strings.Contains(name, arch) {
		return true
	}
	if arch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "x64")) {
		return true
	}
	if arch == "arm64" && strings.Contains(name, "aarch64") {
		return true
	}
	return false
}

func silentUpdatePoll(app *application.App) {
	// Initial check shortly after launch, then once a day.
	timer := time.NewTimer(45 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			if rel, err := app.Updater.Check(context.Background()); err != nil {
				slog.Debug("silent update check", "error", err)
			} else if rel != nil {
				slog.Info("update available", "version", rel.Version)
			}
			timer.Reset(24 * time.Hour)
		}
	}
}

func setupUpdateMenu(app *application.App) {
	check := func(*application.Context) {
		go func() {
			if err := app.Updater.CheckAndInstall(context.Background()); err != nil {
				slog.Error("update flow", "error", err)
			}
		}()
	}

	menu := app.Menu.New()
	switch runtime.GOOS {
	case "darwin":
		appMenu := menu.AddSubmenu("Swiflow")
		appMenu.AddRole(application.About)
		appMenu.AddSeparator()
		appMenu.Add("Check for Updates…").OnClick(check)
		appMenu.AddSeparator()
		appMenu.AddRole(application.ServicesMenu)
		appMenu.AddSeparator()
		appMenu.AddRole(application.Hide)
		appMenu.AddRole(application.HideOthers)
		appMenu.AddRole(application.UnHide)
		appMenu.AddSeparator()
		appMenu.AddRole(application.Quit)
		menu.AddRole(application.EditMenu)
		menu.AddRole(application.WindowMenu)
	default:
		help := menu.AddSubmenu("Help")
		help.Add("Check for Updates…").OnClick(check)
		menu.AddRole(application.EditMenu)
	}
	app.Menu.SetApplicationMenu(menu)
}
