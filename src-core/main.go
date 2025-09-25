package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"swiflow/ability"
	"swiflow/action"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/entry"
	"swiflow/httpd"
)

var (
	Version  string
	Epigraph string
)

func main() {
	mode := flag.String("m", "", "run mode of swiflow core")
	desc := flag.String("d", "", "description of swiflow~")
	flag.Parse()
	switch *mode {
	case "chat":
		if err := config.LoadEnv(); err != nil {
			log.Println("load env fail:", err)
		}
		entry.StartChat(context.Background())
	case "work":
		if err := config.LoadEnv(); err != nil {
			log.Println("load env fail:", err)
		}
		entry.StartWork("../workdata")
	case "test":
		var s = new(httpd.HttpServie)
		// resp := s.InitMcpEnvAsync("uvx-py", "mainland")
		// log.Println("[HTTP] async installation started:", resp)

		// Initial environment check
		env := s.GetMcpEnv().(map[string]any)
		if uv, _ := env["uvx"]; uv != "" {
			log.Println("[HTTP] uvx already available:", uv)
		}
		if py, _ := env["python"]; py != "" {
			log.Println("[HTTP] python already available:", py)
		}

		// check mcp tool is available
		mcpLark := &amcp.McpServer{
			Name: "mcp-pdf-reader",
			Cmd:  "npx", Args: []string{
				"-y", "@larksuiteoapi/lark-mcp",
			},
		}
		if err := mcpLark.Preload(); err != nil {
			log.Println("[HTTP] preload mcp pdf reader fail:", err)
		}
		log.Println("[HTTP] progress... (Press Ctrl+C to exit)")
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("[HTTP] Shutting down...")

	case "debug":
		if err := config.LoadEnv(); err != nil {
			log.Println("load env fail:", err)
		}
		file := ".test/mixed-respond.xml"
		if data, err := os.ReadFile(file); err == nil {
			super := action.Parse(string(data))
			super.Payload = &action.Payload{
				UUID: ".test", Home: ".",
			}
			for _, tool := range super.UseTools {
				switch act := tool.(type) {
				case *action.ExecuteCommand:
					result := act.Handle(super)
					log.Println("exec", result)
				case *action.StartAsyncCmd:
					result := act.Handle(super)
					log.Println("start", result)
				case *action.QueryAsyncCmd:
					result := act.Handle(super)
					log.Println("query", result)
				case *action.AbortAsyncCmd:
					result := act.Handle(super)
					log.Println("abort", result)
				}
			}

			log.Println("start clear process")
			new(ability.DevAsyncCmdAbility).Clear()
		} else {
			log.Println("data:", err, string(data))
		}
	case "serve":
		if err := config.LoadEnv(); err != nil {
			log.Println("load env fail: ", err)
		}
		config.Set("EPIGRAPH", Epigraph)
		if config.GetVersion() == "" {
			config.SetVersion(Version)
		}
		// 设置日志输出路径
		if file, e := config.ServerLog(); e == nil {
			defer file.Close()
			if Version != "" {
				log.SetOutput(file)
			}
		}
		go entry.StartWebServer("127.0.0.1:11235")
		go entry.StartSchedule(context.Background())
		// 监听中断信号（Ctrl+C 或 kill）
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
		<-stopChan // 阻塞直到收到信号

		// 优雅关闭（超时时间设为 5 秒）
		ctx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()
		if err := entry.StopWebServer(ctx); err != nil {
			log.Println("Server shutdown:", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	default:
		log.Println("nice to meet you~", *desc)
	}
}
