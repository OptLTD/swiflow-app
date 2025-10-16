package main

import (
	"app/start"
	"context"
	"embed"
	_ "embed"
	"log"
	"runtime"
	"time"

	"swiflow/config"
	"swiflow/entry"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed html
var html embed.FS

func main() {
	app := application.New(application.Options{
		Name: "Swiflow", Description: "定制属于你的 AI 助理",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(html),
			// disable logging
			DisableLogging: true,
		},
		// Services: []application.Service{
		// 	application.NewService(&hello.Hello{}),
		// },
	})
	tray := app.SystemTray.New()
	if runtime.GOOS == "darwin" {
		data, _ := html.ReadFile(
			"html/images/icon-light.png",
		)
		tray.SetTemplateIcon(data)
	} else {
		data, _ := html.ReadFile(
			"html/images/icon-dark.png",
		)
		tray.SetTemplateIcon(data)
	}

	menu := start.GetMainMenu(app)
	tray.SetMenu(menu).OnClick(func() {
		tray.OpenMenu()
	})

	if file, e := config.ServerLog(); e == nil {
		defer file.Close()
		log.SetOutput(file)
	}

	// 启动 Web 服务与定时任务
	go entry.StartWebServer("127.0.0.1:11235")
	go entry.StartSchedule(context.Background())
	defer func() {
		ctx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()
		if err := entry.StopWebServer(ctx); err != nil {
			log.Println("Server shutdown err:", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	}()

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
