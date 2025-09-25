package entry

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/support"
	"time"

	"github.com/go-co-op/gocron/v2"
)

var scheduler gocron.Scheduler

func StartSchedule(ctx context.Context) error {
	var err error
	if scheduler, err = gocron.NewScheduler(); err != nil {
		return fmt.Errorf("[CRON] failed to create scheduler: %w", err)
	}
	defer func() {
		if err := scheduler.Shutdown(); err != nil {
			log.Printf("[CRON] error shutting down scheduler: %v", err)
		}
	}()

	// 1.5s后立即执行
	boot := gocron.NewTask(handleBooted, "BOOT")
	desc := gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(
		time.Now().Add(1500 * time.Millisecond),
	))
	if job, err := scheduler.NewJob(desc, boot); err != nil {
		log.Printf("[BOOT] fail immediately: %v", err)
	} else {
		log.Printf("[BOOT] start immediately: %s", job.ID())
	}

	intval := time.Duration(config.GetInt("REBOOT", 10))
	desc = gocron.DurationJob(time.Hour * intval)
	task := gocron.NewTask(fetchTodoList, "TODO")
	if job, err := scheduler.NewJob(desc, task); err != nil {
		return fmt.Errorf("[TODO] fail create job: %w", err)
	} else {
		log.Printf("[TODO] start create job: %s", job.ID())
	}

	// servers.json 每天0点定时任务
	desc = gocron.CronJob("0 0 * * *", false)
	task = gocron.NewTask(func() { fetchMcpServers("crontab") })
	if job, err := scheduler.NewJob(desc, task); err != nil {
		log.Printf("[CRON] fail create job: %v", err)
	} else {
		log.Printf("[CRON] start create job: %s", job.ID())
	}

	support.Listen("wait-todo", handleNewTodo)
	support.Listen("mcp-reboot", rebootMcpServers)

	// start
	scheduler.Start()

	<-ctx.Done()
	log.Println("Context cancelled, stopping scheduler")
	return ctx.Err()
}

func handleBooted(name string) {
	fetchTodoList(name)
	fetchSystemEnv(name)
	fetchMcpServers(name)
	startMcpServers(name)
}

func handleNewTodo(name string, data any) {
	todo, _ := data.(*entity.TodoEntity)
	if todo == nil {
		return
	}
	log.Printf(
		"[TODO] handle: %s,%s",
		name, todo.UUID,
	)
	if name != "remove" {
		setTodoTask(todo)
		return
	}
	scheduler.RemoveByTags(todo.UUID)
}

func fetchTodoList(name string) {
	log.Printf("[%s] start pull", name)
	scheduler.RemoveByTags()

	store, _ := manager.GetStorage()
	list, err := store.LoadTodo()
	if len(list) == 0 || err != nil {
		log.Printf("[%s] no todo list", name)
		return
	}
	// todo pull task
	log.Printf("[%s] finish pull", name)
	for _, todo := range list {
		setTodoTask(todo)
	}
}

func setTodoTask(todo *entity.TodoEntity) bool {
	desc := gocron.CronJob(todo.Time, false)
	task := gocron.NewTask(execThisTodo, todo)
	tags := gocron.WithTags(todo.UUID, todo.Task)
	if _, err := scheduler.NewJob(desc, task, tags); err != nil {
		log.Printf("[CRON] failed to create job: %v", err)
		return false
	}
	log.Printf("[%s] %s is ready", todo.Task, todo.UUID)
	return true
}

func execThisTodo(todo *entity.TodoEntity) {
	log.Printf("[%s] start exec todo", todo.UUID)
	act := &action.WaitTodo{
		UUID: todo.UUID,
		Time: todo.Time,
		Todo: todo.Todo,
	}
	task, err := manager.QueryTask(todo.Task)
	if task == nil || err != nil {
		log.Printf("[%s] error %v", todo.UUID, err)
		return
	}

	bot, _ := manager.GetWorker(task.Bots)
	go manager.Handle(act, task, bot)
	log.Printf("[%s] finish exec todo", todo.UUID)
}

func fetchSystemEnv(_ string) {
	store, _ := manager.GetStorage()
	cfg := &entity.CfgEntity{
		Name: entity.KEY_APP_SETUP,
		Type: entity.KEY_APP_SETUP,
	}
	if err := store.FindCfg(cfg); err == nil {
		err = manager.UpdateEnv(cfg)
		log.Println("[BOOT] PRESET ENV", err)
	}

	// @todo bot.Home Future Disabled
	// cfg = &entity.CfgEntity{
	// 	Name: entity.KEY_ACTIVE_BOT,
	// 	Type: entity.KEY_ACTIVE_BOT,
	// }
	// if err := store.FindCfg(cfg); err == nil {
	// 	bot := &entity.BotEntity{}
	// 	maputil.MapTo(cfg.Data, bot)
	// 	config.Set("CURRENT_HOME", bot.Home)
	// 	log.Println("[BOOT] CURRENT_HOME", bot.Home)
	// }
}

// 拉取并保存 servers.json
func fetchMcpServers(name string) {
	host := "https://swiflow.cc"
	url := host + "/servers.json"
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[%s] get servers.json fail: %v", name, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[%s] get servers error: %v", name, resp.Status)
		return
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[%s] read servers fail: %v", name, err)
		return
	}
	path := config.GetWorkPath("servers.json")
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Printf("[%s] save servers fail: %v", name, err)
		return
	}
	log.Printf("[%s] save servers succ", name)
}

// 启动所有MCP服务器
func startMcpServers(_ string) {
	var err error
	if manager == nil {
		manager, err = agent.NewManager()
	}

	if err != nil || manager == nil {
		log.Println("[MCP] 获取存储失败", err)
		return
	}
	store, err := manager.GetStorage()
	if err != nil || store == nil {
		log.Println("[MCP] 获取store失败", err)
		return
	}

	mcpSrv := amcp.GetMcpService(store)
	servers := mcpSrv.ListServers()
	for _, server := range servers {
		if !server.Status.Enable {
			continue
		}
		if err := server.Preload(); err != nil {
			log.Printf("[MCP] Preload Mcp Tools %s Failure: %v", server.Name, err)
		} else {
			log.Printf("[MCP] Preload Mcp Tools %s Success", server.Name)
		}
	}
}

func rebootMcpServers(name string, data any) {
	bot, _ := data.(*entity.BotEntity)
	if bot == nil || bot.Home == "" {
		return
	}

	store, err := manager.GetStorage()
	if err != nil || store == nil {
		log.Println("[MCP] 获取store失败", err)
		return
	}

	mcpSrv := amcp.GetMcpService(store)
	servers := mcpSrv.ListServers()
	for _, server := range servers {
		if !server.Status.Enable {
			continue
		}
		if !server.Status.Active {
			continue
		}

		var needReboot = false
		for _, val := range server.Env {
			if val == "$CURRENT_HOME" {
				needReboot = true
				break
			}
		}
		if !needReboot {
			continue
		}
		go func() {
			if mcpSrv.ServerClose(server) == nil {
				log.Printf("[MCP] Stop Server %s Success", server.Name)
			}
			if err := mcpSrv.ServerStatus(server); err == nil {
				log.Printf("[MCP] Start Server %s Success", server.Name)
				log.Printf("[MCP] 启动服务器 %s 失败: %v", server.Name, err)
			} else {
				log.Printf("[MCP] Start Server %s Fail: %v", server.Name, err)
			}
		}()
	}
}
