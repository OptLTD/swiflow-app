package demo

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

func GetDemoMenu(app *application.App) *application.Menu {

	menu := app.NewMenu()

	menu.Add("Show Window!")
	menu.AddSeparator()
	menu.AddRole(application.Role(application.WindowMenu))
	// myMenu.Add("Wails").SetBitmap(logo).SetEnabled(false)
	menu.Add("Hello World!").OnClick(func(ctx *application.Context) {
		println("Hello World!")
		q := application.QuestionDialog().SetTitle("Ready?").SetMessage("Are you feeling ready?")
		q.AddButton("Yes").OnClick(func() {
			println("Awesome!")
		})
		q.AddButton("No").SetAsDefault().OnClick(func() {
			println("Boo!")
		})
		q.Show()
	})

	subMenu := menu.AddSubmenu("Submenu")
	subMenu.Add("Click me!").OnClick(func(ctx *application.Context) {
		ctx.ClickedMenuItem().SetLabel("Clicked!")
	})
	menu.AddSeparator()
	menu.AddCheckbox("Checked", true).OnClick(func(ctx *application.Context) {
		println("Checked: ", ctx.ClickedMenuItem().Checked())
		application.InfoDialog().SetTitle("Hello World!").SetMessage("Hello World!").Show()
	})
	menu.Add("Enabled").OnClick(func(ctx *application.Context) {
		println("Click me!")
		ctx.ClickedMenuItem().SetLabel("Disabled!").SetEnabled(false)
	})
	menu.AddSeparator()
	// Callbacks can be shared. This is useful for radio groups
	radioCallback := func(ctx *application.Context) {
		menuItem := ctx.ClickedMenuItem()
		menuItem.SetLabel(menuItem.Label() + "!")
	}

	// Radio groups are created implicitly by placing radio items next to each other in a menu
	menu.AddRadio("Radio 1", true).OnClick(radioCallback)
	menu.AddRadio("Radio 2", false).OnClick(radioCallback)
	menu.AddRadio("Radio 3", false).OnClick(radioCallback)
	menu.AddSeparator()
	menu.Add("Hide Tray").SetTooltip("recover after 3 seconds...")
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	return menu
}
