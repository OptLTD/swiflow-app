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
	"swiflow/config"
	"swiflow/httpd"
	"swiflow/service"
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
		service.StartChat(context.Background())
	case "test":
		var s = new(httpd.HttpServie)
		resp := s.InitMcpEnv("uvx-py", "mainland")
		log.Println("[HTTP] resp error", resp)
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
		go service.StartWebServer("127.0.0.1:11235")
		go service.StartSchedule(context.Background())
		// 监听中断信号（Ctrl+C 或 kill）
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
		<-stopChan // 阻塞直到收到信号

		// 优雅关闭（超时时间设为 5 秒）
		ctx, cancel := context.WithTimeout(
			context.Background(), 5*time.Second,
		)
		defer cancel()
		if err := service.StopWebServer(ctx); err != nil {
			log.Println("Server shutdown:", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	case "schedule":
		if err := config.LoadEnv(); err != nil {
			log.Println("load env fail:", err)
		}
		service.StartSchedule(context.Background())
	default:
		log.Println("current env", os.Environ())
		log.Println("nice to meet you~", *desc)
	}
}
