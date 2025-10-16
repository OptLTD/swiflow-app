package start

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"swiflow/config"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
)

var docker = dock.New()

func GetMainView(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Width: 1200, Height: 800, Name: app.Config().Name,
		StartState: application.WindowStateMaximised,

		Frameless: false, AlwaysOnTop: false,
		DisableResize: false, Hidden: false,
		DevToolsEnabled: true, EnableDragAndDrop: true,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: false,
		},
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
		Linux: application.LinuxWindow{},
	})
	dropped := events.Common.WindowDropZoneFilesDropped
	window.OnWindowEvent(dropped, func(event *application.WindowEvent) {
		detail := UploadFiles(event.Context().DroppedFiles())
		app.Event.Emit("app:Uploaded", detail)
	})

	app.Event.On("app:FileSelected", func(event *application.CustomEvent) {
		files := toStringSlice(event.Data)
		detail := UploadFiles(files)
		app.Event.Emit("app:Uploaded", detail)
	})

	closing := events.Common.WindowClosing
	window.RegisterHook(closing, func(e *application.WindowEvent) {
		docker.HideAppIcon()
		window.Hide()
		e.Cancel()
	})
	window.Show()
	return window
}

func GetMainMenu(app *application.App) *application.Menu {
	view, menu := GetMainView(app), app.NewMenu()
	menu.Add("Show").OnClick(func(ctx *application.Context) {
		if view.IsVisible() && view.IsFocused() {
			return
		}
		if !view.IsVisible() {
			docker.ShowAppIcon()
			view.Show()
		}
		view.Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	return menu
}

type UploadDetail struct {
	Files  []string `json:"files"`
	Result []string `json:"result"`
	Errors []error  `json:"errors"`
}

func UploadFiles(files []string) *UploadDetail {
	// Determine destination home directory
	home := strings.TrimSpace(config.CurrentHome())
	if home == "" {
		home = config.GetWorkHome()
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(home, 0755); err != nil {
		return &UploadDetail{Files: files, Result: nil, Errors: []error{err}}
	}

	results := make([]string, 0, len(files))
	errs := make([]error, 0)

	for _, src := range files {
		if strings.TrimSpace(src) == "" {
			errs = append(errs, fmt.Errorf("empty source file path"))
			continue
		}

		// Verify source exists and is a regular file
		info, err := os.Stat(src)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !info.Mode().IsRegular() {
			errs = append(errs, fmt.Errorf("not a regular file: %s", src))
			continue
		}

		base := filepath.Base(src)
		dest := filepath.Join(home, base)
		if err := copyFile(src, dest); err != nil {
			errs = append(errs, err)
			continue
		}
		// Only keep the filename in result
		results = append(results, base)
	}

	return &UploadDetail{Files: files, Result: results, Errors: errs}
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// Ensure the writer is closed and propagate close error if needed
	defer func() {
		cerr := out.Close()
		if err == nil && cerr != nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	if fi, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, fi.Mode())
	}
	return nil
}

// toStringSlice converts arbitrary event data into a []string safely.
// Supports []string and []interface{} (each element string or fmt-printable).
func toStringSlice(data any) []string {
	switch v := data.(type) {
	case nil:
		return []string{}
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			} else {
				out = append(out, fmt.Sprint(item))
			}
		}
		return out
	case string:
		return []string{v}
	default:
		return []string{}
	}
}
