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
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// 关闭主窗口时隐藏窗口并取消关闭事件
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
