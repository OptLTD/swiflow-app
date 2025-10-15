package demo

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// attach window close tray
// systemTray.AttachWindow(webview).WindowOffset(2)
func GetMiniPanel(app *application.App) *application.WebviewWindow {
	webview := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Width: 500, Height: 500, Name: "Systray Demo Window",
		Frameless: true, AlwaysOnTop: true,
		DisableResize: true, Hidden: true,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})

	webview.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		webview.Hide()
		e.Cancel()
	})

	return webview
}
