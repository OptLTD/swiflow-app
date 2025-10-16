package start

import (
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
		app.Event.Emit("app:FileDropped", event.Context().DroppedFiles())
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
