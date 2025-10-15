package demo

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func GetMainView(app *application.App) *application.WebviewWindow {
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Width: 1200, Height: 800, Name: "Systray Demo Window",
		StartState: application.WindowStateMaximised,

		Frameless: false, AlwaysOnTop: false,
		DisableResize: false, Hidden: false,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: false,
		},
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
		Linux: application.LinuxWindow{},
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})
	window.Show()
	return window
}
