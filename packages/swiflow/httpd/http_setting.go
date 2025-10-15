package httpd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/amcp"
	"swiflow/builtin"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/model"
	"swiflow/storage"
	"swiflow/support"
	"time"

	"github.com/duke-git/lancet/v2/slice"
)

type SettingHandler struct {
	service *HttpServie
	manager *agent.Manager
}

func NewSettingHandle(m *agent.Manager) *SettingHandler {
	var s = new(HttpServie)
	store, e := storage.GetStorage()
	if store != nil && e == nil {
		s.store = store
	}
	return &SettingHandler{s, m}
}

func (h *SettingHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks := []map[string]any{}
	store, _ := storage.GetStorage()
	query := "(`group`='' or `group`=`uuid`)"
	if config.Get("USE_SUBAGENT") != "yes" {
		// filter task not by leader
		query = "(`group`='' or `group`!=`uuid`)"
	}
	if list, err := store.LoadTask(query); err == nil {
		if len(list) == 0 {
			list, _ = store.LoadTask()
		}
		for _, item := range list {
			tasks = append(tasks, item.ToMap())
		}
	}
	JsonResp(w, tasks)
}

func (h *SettingHandler) GetMsgs(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("task")
	store, _ := storage.GetStorage()
	tasks, _ := store.LoadTask("group", uuid)
	useSubagent := config.Get("USE_SUBAGENT")
	if len(tasks) == 0 || useSubagent != "yes" {
		tasks, _ = store.LoadTask("uuid", uuid)
	}
	context := agent.Context{}
	result := []*action.SuperAction{}
	for _, task := range tasks {
		msgs, err := store.LoadMsg(task)
		if err != nil || len(msgs) == 0 {
			continue
		}

		acts := context.ParseMsgs(msgs)
		for _, act := range acts {
			if act.WorkerID == "" {
				act.WorkerID = task.BotId
			}
			result = append(result, act)
		}
	}
	// Sort by datetime using sort function
	sort.Slice(result, func(i, j int) bool {
		return result[i].Datetime < result[j].Datetime
	})

	// SuperAction now has custom MarshalJSON method that uses ToMap format
	if err := JsonResp(w, result); err != nil {
		log.Println("[HTTP] resp error", err)
	}
}

func (h *SettingHandler) TaskSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	uuid := r.URL.Query().Get("uuid")
	task, _ := h.manager.QueryTask(uuid)
	store, _ := storage.GetStorage()
	switch act {
	case "get-task", "":
		subtasks := []string{}
		result := task.ToMap()
		tasks, _ := store.LoadTask("group", uuid)
		for _, item := range tasks {
			subtasks = append(subtasks, item.UUID)
		}
		result["subtasks"] = subtasks
		if err := JsonResp(w, result); err != nil {
			log.Println("resp error", err)
		}
		return
	case "del-task":
		task.DeletedAt.Time = time.Now()
	case "set-bot":
		task.BotId = r.URL.Query().Get("bot")
	case "set-home":
		task.Home = r.URL.Query().Get("home")
	}
	if err := store.SaveTask(task); err != nil {
		JsonResp(w, fmt.Errorf("error: %w", err))
		return
	}
	if act == "del-task" && task.Home != "" {
		os.RemoveAll(config.GetWorkPath(task.UUID))
	}
	if err := JsonResp(w, task.ToMap()); err != nil {
		log.Println("resp error", err)
	}
}

func (h *SettingHandler) TodoSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	store, _ := storage.GetStorage()
	switch act {
	case "get-todo":
		resp := []any{}
		list, _ := store.LoadTodo("done = ?", 0)
		for _, item := range list {
			resp = append(resp, item)
		}
		if err := JsonResp(w, resp); err != nil {
			log.Println("resp error", err)
		}
		return
	case "get-done":
		resp := []any{}
		list, _ := store.LoadTodo("done = ?", 1)
		for _, item := range list {
			resp = append(resp, item.ToMap())
		}
		if err := JsonResp(w, resp); err != nil {
			log.Println("resp error", err)
		}
	case "set-done":
		uuid := r.URL.Query().Get("uuid")
		todo := &entity.TodoEntity{UUID: uuid}
		if err := store.FindTodo(todo); err != nil {
			JsonResp(w, err)
			return
		}
		if todo.Done += 1; todo.Done == 1 {
			if err := store.SaveTodo(todo); err == nil {
				support.Emit("wait-todo", "remove", todo)
			}
			JsonResp(w, todo.ToMap())
		} else {
			JsonResp(w, todo.ToMap())
		}
	}
}

func (h *SettingHandler) BotSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	uuid := r.URL.Query().Get("uuid")
	bot := h.service.FindBot(uuid)
	if uuid != "" && bot == nil {
		http.NotFound(w, r)
		return
	}
	switch act {
	case "get-bot":
		// here need return usePrompt and sysPrompt
		if err := JsonResp(w, bot); err != nil {
			log.Println("resp error", err)
		}
		return
	case "get-bots":
		bots := h.service.LoadBot()
		data := []map[string]any{}
		for _, bot := range bots {
			data = append(data, bot.ToMap())
		}
		JsonResp(w, data)
		return
	case "init-bot":
		bots := h.service.InitBot()
		if len(bots) == 0 {
			JsonResp(w, fmt.Errorf("no bot avaliable"))
			return
		}
		if err := h.manager.Initial(h.service.store); err != nil {
			JsonResp(w, fmt.Errorf("initial err: %w", err))
			return
		}
		if err := JsonResp(w, bots); err != nil {
			log.Println("resp error", err)
		}
		return
	case "use-bot":
		if err := h.service.UseBot(bot); err == nil {
			// @todo because bot.Home Future Disabled
			// support.Emit("mcp-reboot", "use-bot", bot)
		}
	case "del-bot":
		bot.UUID, bot.DeletedAt.Time = uuid, time.Now()
		if err := h.service.SaveBot(bot); err != nil {
			JsonResp(w, bot.ToMap())
			return
		}
	case "prompt":
		if bot.Leader != "" {
			bot.Type = agent.AGENT_WORKER
		} else {
			bot.Type = agent.AGENT_LEADER
		}
		store, _ := storage.GetStorage()
		context := agent.NewContext(store, bot)
		w.Write([]byte(*context.UsePrompt()))
		return
	case "set-home":
		bot.Home = r.URL.Query().Get("home")
		if err := h.service.SaveBot(bot); err != nil {
			JsonResp(w, bot.ToMap())
			return
		}
	case "set-tools":
		tools := r.URL.Query().Get("tools")
		bot.Tools = strings.Split(tools, ",")
		if err := h.service.SaveBot(bot); err != nil {
			JsonResp(w, bot.ToMap())
			return
		}
	case "set-provider":
		bot.Provider = r.URL.Query().Get("provider")
		if bot.Provider != "" && !h.service.HasProvider(bot.Provider) {
			JsonResp(w, fmt.Errorf("no provider config of %s", bot.Provider))
			return
		}
		if err := h.service.SaveBot(bot); err != nil {
			JsonResp(w, bot.ToMap())
			return
		}
	case "set-bot":
		if uuid == "" {
			bot = new(entity.BotEntity)
		}
		data, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(data, bot)
		if err != nil || bot.Name == "" {
			JsonResp(w, fmt.Errorf("error input"))
			return
		}
		if bot.UUID == "" {
			uuid, _ := support.UniqueID(8)
			bot.UUID = "bot-" + uuid
		}
		if h.service.SaveBot(bot) != nil {
			JsonResp(w, fmt.Errorf("error save"))
			return
		}
	}
	h.manager.ResetWorker(bot)
	JsonResp(w, bot.ToMap())
}
func (h *SettingHandler) NewMcp(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	store, _ := storage.GetStorage()
	service := amcp.GetMcpService(store)
	server := &amcp.McpServer{}
	if err := h.service.ReadTo(r.Body, server); err != nil {
		JsonResp(w, err)
		return
	}
	switch act {
	case "set-new":
		server.Name = support.Or(server.Name, server.UUID)
		if err := service.UpsertServer(server); err != nil {
			JsonResp(w, fmt.Errorf("upsert: %v", err))
			return
		}
		if err := server.Preload(); err != nil {
			err = JsonResp(w, err)
			return
		}
		if err := service.ServerStatus(server); err == nil {
			_ = service.EnableServer(server)
			err = JsonResp(w, server)
		} else {
			err = JsonResp(w, err)
		}
	case "test-mcp":
		server.Type = "debug"
		if server.Name == "" && server.UUID != "" {
			server.Name = server.UUID
		}
		uuid := r.URL.Query().Get("uuid")
		err := service.QueryServer(server, 1)
		if err == nil && uuid == "" { // new mcp test
			JsonResp(w, fmt.Errorf("server existed: %v", server.Name))
			return
		}
		if err := server.Preload(); err != nil {
			err = JsonResp(w, err)
			return
		}
		if err := service.ServerStatus(server); err != nil {
			JsonResp(w, fmt.Errorf("Status: %v", err))
			return
		}
		JsonResp(w, server.Status)
	}
}

func (h *SettingHandler) GetMcp(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	switch act {
	case "get-mcp":
		store, _ := storage.GetStorage()
		service := amcp.GetMcpService(store)
		mcpList := service.ListServers()
		var tools, _ = store.LoadTool()
		var manager = builtin.GetManager().Init(tools)
		var builtinServer = &amcp.McpServer{
			Name: "Builtin Tools", UUID: "builtin",
		}
		var mcpTools = builtinServer.Status.McpTools
		var prependList = []*amcp.McpServer{builtinServer}
		for _, item := range manager.GetList() {
			mcpTools = append(mcpTools, &amcp.McpTool{
				Name: item.UUID, Title: item.Name,
			})
		}
		builtinServer.Status.Enable = true
		builtinServer.Status.Active = true
		builtinServer.Status.McpTools = mcpTools
		mcpList = append(prependList, mcpList...)
		if err := JsonResp(w, mcpList); err != nil {
			log.Println("resp error", err)
		}
		return
	case "servers":
		w.Header().Set("Content-Type", "application/json")
		path := config.GetWorkPath("servers.json")
		data, _ := os.ReadFile(path)
		_, _ = w.Write(data)
	}
}

func (h *SettingHandler) McpSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	list := []string{"set-new", "test-mcp"}
	if slice.Contain(list, act) {
		h.NewMcp(w, r)
		return
	}
	list = []string{"get-mcp", "servers"}
	if slice.Contain(list, act) {
		h.GetMcp(w, r)
		return
	}

	var found *amcp.McpServer
	store, _ := storage.GetStorage()
	service := amcp.GetMcpService(store)
	mcpList := service.ListServers()
	uuid := r.URL.Query().Get("uuid")
	for _, server := range mcpList {
		if server.UUID == uuid {
			found = server
			break
		}
	}
	if found == nil {
		JsonResp(w, fmt.Errorf("未找到MCP服务器: %s", uuid))
		return
	}
	switch act {
	case "clear":
		service.ClearDebugMsgs(found)
		if err := JsonResp(w, "success"); err != nil {
			log.Println("query status error", err)
		}
	case "msgs":
		list := []map[string]any{}
		msgs := service.LoadDebugMsgs(found)
		acts := new(agent.Context).ParseMsgs(msgs)
		for _, act := range acts {
			list = append(list, act.ToMap())
		}
		if err := JsonResp(w, list); err != nil {
			log.Println("query status error", err)
		}
	case "stop":
		if err := model.Cancel(uuid); err == nil {
			uuid := "#debug#" + uuid
			task := &entity.TaskEntity{
				UUID: uuid, Name: uuid,
			}
			bot, _ := h.manager.GetWorker(uuid)
			executor := h.manager.LoadExecutor(task, bot)
			if err = executor.Terminate(); err != nil {
				log.Println("query status error", err)
			}
			JsonResp(w, "success")
		}
	case "execute":
		tool := r.URL.Query().Get("tool")
		data := h.service.ReadMap(r.Body)
		client := service.GetMcpClient(found)
		args, _ := data.(map[string]any)
		res, err := client.Execute(tool, args)
		if err == nil && res != "" {
			err = JsonResp(w, res)
			return
		}
		if e := JsonResp(w, err); e != nil {
			log.Println("query status error", e)
			return
		}
	case "active":
		status := found.Status
		service.QueryServer(found)
		if status.Active && len(status.McpTools) > 0 {
			JsonResp(w, found.Status)
			return
		}
		if err := found.Preload(); err != nil {
			err = JsonResp(w, err)
			return
		}
		if err := service.ServerStatus(found); err == nil {
			_ = service.EnableServer(found)
			err = JsonResp(w, found.Status)
		} else if e := JsonResp(w, err); e != nil {
			log.Println("query status error", e)
		}
	case "disable":
		if err := service.ServerClose(found); err == nil {
			_ = service.DisableServer(found)
			err = JsonResp(w, found.Status)
		} else if e := JsonResp(w, err); e != nil {
			log.Println("server disable error", e)
		}
	case "set-mcp":
		if err := h.service.ReadTo(r.Body, found); err != nil {
			JsonResp(w, err)
			return
		} else {
			service.ServerClose(found)
			service.ClearDebugMsgs(found)
		}
		if err := service.UpsertServer(found); err != nil {
			JsonResp(w, err)
			return
		}
		if err := JsonResp(w, found); err != nil {
			_ = service.EnableServer(found)
			log.Println("upsert server error", err)
		}
	case "del-mcp":
		_ = service.ServerClose(found)
		resp := service.RemoveServer(found)
		if err := JsonResp(w, resp); err != nil {
			log.Println("remove server error", err)
		}
	default:
		JsonResp(w, fmt.Errorf("undefined act"))
	}
}

// ToolSet handles tool-related operations under setting.
// Purpose: list builtin tools and manage user-defined tool aliases.
func (h *SettingHandler) ToolSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	store, _ := storage.GetStorage()
	var tools, _ = store.LoadTool()
	var manager = builtin.GetManager().Init(tools)
	switch act {
	case "get-tools", "":
		JsonResp(w, manager.AllTools())
		return
	case "set-tool":
		// Save or update a user-defined tool alias
		tool := new(entity.ToolEntity)
		if err := h.service.ReadTo(r.Body, tool); err != nil {
			_ = JsonResp(w, err)
			return
		}
		// Default type to alias; generate UUID if missing
		if strings.TrimSpace(tool.Type) == "" {
			tool.Type = "cmd_alias"
		}
		if strings.TrimSpace(tool.UUID) == "" {
			tool.UUID, _ = support.UniqueID(12)
		}
		if err := store.SaveTool(tool); err != nil {
			_ = JsonResp(w, err)
			return
		}
		if items, err := store.LoadTool(); err == nil {
			builtin.GetManager().Init(items)
		}
		_ = JsonResp(w, tool.ToMap())
		return
	default:
		_ = JsonResp(w, "undefined command")
		return
	}
}

func (h *SettingHandler) MemSet(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	if act == "get-mem" {
		list := h.service.LoadMem()
		mems := []map[string]any{}
		for _, r := range list {
			mems = append(mems, r.ToMap())
		}
		if err := JsonResp(w, mems); err != nil {
			log.Println("resp error", err)
		}
		return
	}

	uuid := r.URL.Query().Get("uuid")

	switch act {
	case "set-mem":
		data, _ := io.ReadAll(r.Body)
		var mem entity.MemEntity
		if err := json.Unmarshal(data, &mem); err != nil {
			JsonResp(w, fmt.Errorf("error input"))
			return
		}

		// 验证bot字段不能为空
		if mem.Bot == "" {
			JsonResp(w, fmt.Errorf("bot field is required"))
			return
		}

		// 如果是更新现有记录
		if uuid != "" {
			uid, err := strconv.ParseUint(uuid, 10, 32)
			if err != nil || uid == 0 {
				JsonResp(w, fmt.Errorf("invalid id format: %v", uuid))
				return
			}
			find := h.service.FindMem(uint(uid))
			if find == nil {
				http.NotFound(w, r)
				return
			}
			// 更新现有记录
			find.Bot = mem.Bot
			find.Type = mem.Type
			find.Subject = mem.Subject
			find.Content = mem.Content
			h.service.SaveMem(find)
			if err := JsonResp(w, find.ToMap()); err != nil {
				log.Println("resp error", err)
			}
		} else {
			// 新增记录
			h.service.SaveMem(&mem)
			if err := JsonResp(w, mem.ToMap()); err != nil {
				log.Println("resp error", err)
			}
		}
		return

	case "del-mem":
		if uuid == "" {
			JsonResp(w, fmt.Errorf("uuid is required for delete operation"))
			return
		}
		uid, err := strconv.ParseUint(uuid, 10, 32)
		if err != nil || uid == 0 {
			JsonResp(w, fmt.Errorf("invalid id format: %v", uuid))
			return
		}
		mem := h.service.FindMem(uint(uid))
		if mem == nil {
			http.NotFound(w, r)
			return
		}
		// 设置删除时间，然后保存
		mem.DeletedAt.Time = time.Now()
		h.service.SaveMem(mem)
		if err := JsonResp(w, mem.ToMap()); err != nil {
			log.Println("resp error", err)
		}
	}
}
