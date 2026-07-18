package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LightAppService exposes light-app window management to the frontend via Wails IPC.
type LightAppService struct {
	app *application.App
}

// OpenWindow opens a light app in a dedicated child window.
// url is the http://127.0.0.1:<port> URL returned by the launch API.
// title is the app name shown in the window title bar.
func (s *LightAppService) OpenWindow(url, title string) {
	if url == "" {
		return
	}
	opts := application.WebviewWindowOptions{
		URL: url, Title: title,
		Width: 1024, Height: 768, MinWidth: 640, MinHeight: 480,
		BackgroundColour: application.NewRGB(255, 255, 255),
	}
	switch runtime.GOOS {
	case "darwin":
		opts.Mac = application.MacWindow{
			TitleBar: application.MacTitleBarDefault,
		}
	case "windows":
		opts.Frameless = false
	}
	win := s.app.Window.NewWithOptions(opts)
	win.Show()
}
