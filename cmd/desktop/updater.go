package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"

	"github.com/OptLTD/swiflow/internal/version"
)

// Default Wails Update Manifest URL (served from R2 via dl.swiflow.cc / r2.swiflow.cc).
// Override with SWIFLOW_UPDATE_MANIFEST.
const defaultUpdateManifestURL = "https://dl.swiflow.cc/update.json"

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

// CheckLatest silently queries the update manifest (no update window).
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

func updateManifestURL() string {
	if v := strings.TrimSpace(os.Getenv("SWIFLOW_UPDATE_MANIFEST")); v != "" {
		return v
	}
	return defaultUpdateManifestURL
}

// setupUpdater wires endpoint-based self-update, menu, and silent polling.
func setupUpdater(app *application.App) {
	manifestURL := updateManifestURL()
	ep, err := endpoint.New(endpoint.Config{
		URL: manifestURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	})
	if err != nil {
		slog.Error("updater endpoint provider", "error", err)
		return
	}
	// No CheckInterval: framework interval uses CheckAndInstall (pops a window).
	// We poll with silent Check() instead so the UI can show a badge first.
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version.Version,
		Providers:      []updater.Provider{ep},
	}); err != nil {
		slog.Error("updater init", "error", err)
		return
	}
	slog.Info("updater ready", "version", version.Version, "manifest", manifestURL)
	setupUpdateMenu(app)
	go silentUpdatePoll(app)
}

func silentUpdatePoll(app *application.App) {
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
