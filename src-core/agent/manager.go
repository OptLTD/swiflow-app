package agent

import (
	"fmt"
	"log"
	"strings"
	"swiflow/ability"
	"swiflow/action"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/initial"
	"swiflow/model"
	"swiflow/storage"
	"swiflow/support"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
)

type Store = storage.MyStore
type MyBot = entity.BotEntity
type MyMsg = entity.MsgEntity
type MyTask = entity.TaskEntity
type MyInput = action.UserInput
type Payload = action.Payload

type Manager struct {
	store Store

	mybots []*MyBot
	config map[string]any
	active map[string]*Executor
}

func NewManager() (*Manager, error) {
	m := &Manager{}

	// init storage
	m.config = map[string]any{}
	m.active = map[string]*Executor{}
	store, err := m.InitStorage()
	if err != nil || store == nil {
		log.Println("[AGENT] init store error", err)
		return nil, fmt.Errorf("init store error: %v", err)
	} else {
		m.store = store
	}
	if err := m.Initial(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) Initial() (err error) {
	if info := config.EpigraphInfo(); len(info) > 0 {
		epigraph := &entity.CfgEntity{
			Type: entity.KEY_EPIGRAPH,
			Name: entity.KEY_EPIGRAPH,
			Data: info,
		}
		m.store.SaveCfg(epigraph)
	}
	if err = m.InitConfig(); err != nil {
		log.Println("[AGENT] init cfg error", err)
	}
	if m.mybots, err = m.store.LoadBot(); err == nil {
		return nil
	}
	log.Println("[AGENT] init bot error", err)
	return fmt.Errorf("init bot error: %v", err)
}

func (m *Manager) Handle(input action.Input, task *MyTask, bot *MyBot) {
	var executor *Executor
	if e := m.LoadExecutor(task, bot); e != nil {
		executor = e
	}

	if strings.HasPrefix(task.UUID, "#debug#") {
		bot.UUID, bot.Type = task.Name, "debug"
		mcpServ := amcp.GetMcpService(m.store)
		server := &amcp.McpServer{UUID: task.Name}
		if err := mcpServ.QueryServer(server); err != nil {
			log.Println("[AGENT] query server error", err)
			support.Emit("errors", task.UUID, "mcp server not existed")
			return
		}
		if len(server.Status.McpTools) == 0 {
			log.Println("[AGENT] start mcp server", server.UUID)
			err := mcpServ.ServerStatus(server)
			if len(server.Status.McpTools) == 0 {
				log.Println("[AGENT] query status error", err)
				support.Emit("errors", task.UUID, "mcp server in booting")
				return
			}
		}

		executor.context.usePrompt = m.GetPrompt(bot)
		executor.context.store = mcpServ.GetMockStore()
		log.Println("[AGENT] bot name", bot.UUID, bot.Type)
		log.Println("[AGENT] prompt", executor.context.usePrompt)
	}

	if executor == nil {
		support.Emit("errors", task.UUID, "load executor error")
		return
	}
	if executor.modelClient == nil {
		if cfg := m.GetLLMConfig(bot.Provider); cfg == nil {
			support.Emit("errors", task.UUID, "no model avalible")
			return
		} else {
			executor.modelClient = model.GetClient(cfg)
		}
	}

	// 直接Enqueue，无论是否正在运行
	executor.Enqueue(input)
}

func (m *Manager) InitTask(name string, uuid string) (*MyTask, error) {
	if uuid == "" {
		uuid, _ = support.UniqueID()
	}
	task := &MyTask{
		UUID: uuid, Bots: m.CurrentBot(),
		Name: support.Substring(name, 20),
	}

	if err := m.store.InitTask(task); err != nil {
		log.Println("[AGENT] init task err", err)
		return nil, err
	}
	return task, nil
}

func (m *Manager) NewMcpTask(uuid string) (*MyTask, error) {
	name, found := strings.CutPrefix(uuid, "#debug#")
	if !found || name == "" {
		return nil, fmt.Errorf("wrong format uuid")
	}

	task := &MyTask{UUID: uuid, Name: name}
	return task, nil
}

func (m *Manager) QueryTask(uuid string) (*MyTask, error) {
	if uuid == "" {
		return nil, fmt.Errorf("empty task uuid")
	}
	task := &MyTask{UUID: uuid}
	if err := m.store.FindTask(task); err != nil {
		log.Println("[AGENT] find task err:", uuid, err)
		return nil, err
	}
	return task, nil
}

func (m *Manager) QueryBot(uuid string) (*MyBot, error) {
	var bot = &MyBot{UUID: uuid}
	if err := m.store.FindBot(bot); err != nil {
		log.Println("[AGENT] find bot err:", uuid, err)
		return nil, err
	}
	baseHome := config.GetWorkPath(bot.UUID)
	bot.Home = support.Or(bot.Home, baseHome)
	return bot, nil
}

func (m *Manager) SelectBot(uuid string) (*MyBot, error) {
	if uuid == "" {
		uuid = m.CurrentBot()
	}
	for _, bot := range m.mybots {
		if bot.UUID == uuid {
			return bot, nil
		}
	}
	if uuid == "nobody" && len(m.mybots) > 0 {
		return m.mybots[0], nil
	}
	log.Println("[AGENT] select bot err", uuid)
	return nil, fmt.Errorf("not found: %s", uuid)
}

func (m *Manager) GetLLMConfig(name string) *model.LLMConfig {
	data := map[string]any{}
	cfg := &model.LLMConfig{}
	provider := strings.ToLower(name)
	if provider != "" {
		val, _ := m.config[provider]
		data, _ = val.(map[string]any)
	}
	if provider == "" || len(data) == 0 {
		val, _ := m.config[entity.KEY_USE_MODEL]
		data, _ = val.(map[string]any)
	}
	if len(data) == 0 {
		return nil
	}
	if maputil.MapTo(data, cfg) == nil {
		// cfg.Provider = provider
	}
	if provider != "" && cfg.Provider == "" {
		cfg.Provider = provider
	}
	if cfg.ApiKey == "" && provider != "" {
		return m.GetLLMConfig("")
	}
	return cfg
}

func (m *Manager) GetExecutor(task *MyTask, bot *MyBot) *Executor {
	baseHome := config.GetWorkPath(bot.UUID)
	bot.Home = support.Or(bot.Home, baseHome)
	payload := &Payload{
		UUID: task.UUID, Time: time.Now(),
		Home: bot.Home, Path: task.Home,
	}
	context := &Context{
		bot: bot, task: task,
		store: m.store,
	}
	executor := &Executor{
		UUID:    task.UUID,
		context: context,
		payload: payload,
	}
	if cfg := m.GetLLMConfig(bot.Provider); cfg != nil {
		cfg.TaskId = task.UUID
		c := model.GetClient(cfg)
		executor.modelClient = c
	}

	switch bot.UUID {
	case "master":
		bot.Type = "master"
	default:
		bot.Type = "slave"
	}
	context.useMemory = m.GetMemory(bot)
	context.usePrompt = m.GetPrompt(bot)
	log.Println("[AGENT] bot's tools", bot.Name, bot.Tools)

	return executor
}

func (m *Manager) LoadExecutor(task *MyTask, bot *MyBot) *Executor {
	key := task.UUID + "-" + bot.UUID
	if executor, ok := m.active[key]; ok {
		return executor
	}
	executor := m.GetExecutor(task, bot)
	if executor != nil {
		m.active[key] = executor
		return executor
	}
	return nil
}

func (m *Manager) GetMemory(bot *MyBot) string {
	var memory strings.Builder
	for _, mem := range bot.Memories {
		memory.WriteString("\n")
		memorize := &action.Memorize{
			Content:  strings.TrimSpace(mem.Content),
			Subject:  strings.TrimSpace(mem.Subject),
			Datetime: mem.CreatedAt.String(),
		}
		memory.WriteString(support.ToXML(memorize, nil))
	}
	return memory.String()
}
func (m *Manager) GetPrompt(bot *MyBot) string {
	prompt := m.UsePrompt(bot)
	prompt = strings.ReplaceAll(
		prompt, "${{WORK_PATH}}", bot.Home,
	)
	prompt = strings.ReplaceAll(
		prompt, "${{SELF_BOTS}}",
		m.buildSelfBots(),
	)

	prompt = strings.ReplaceAll(
		prompt, "${{SELF_TOOLS}}",
		m.buildSelfTools(bot),
	)
	return m.buildSysInfo(prompt)
}

func (m *Manager) UsePrompt(bot *MyBot) string {
	bot.Type = support.Or(bot.Type, "slave")
	var prompt = initial.UsePrompt(bot.Type)
	prompt = strings.ReplaceAll(
		prompt, "${{USER_PROMPT}}", bot.UsePrompt,
	)

	mcpTools := m.buildMcpTools(bot)
	prompt = strings.ReplaceAll(
		prompt, "${{MCP_TOOLS}}", mcpTools,
	)
	if mcpTools == "error:server-no-tools" {
		return ""
	}

	return prompt
}
func (m *Manager) buildSysInfo(prompt string) string {
	tag := action.TOOL_RESULT_TAG
	osName, shell := config.GetShellName()
	prompt = strings.ReplaceAll(prompt, "${{OS_NAME}}", osName)
	prompt = strings.ReplaceAll(prompt, "${{SHELL_NAME}}", shell)
	return strings.ReplaceAll(prompt, "${{TOOL_RESULT_TAG}}", tag)
}

func (m *Manager) buildSelfBots() string {
	var result strings.Builder
	for _, bot := range m.mybots {
		if bot.Type == "master" {
			continue
		}
		result.WriteString(fmt.Sprintf(
			"- **%s** (%s): %s\n",
			bot.Name, bot.UUID, bot.Desc,
		))
	}

	if result.Len() == 0 {
		return "none"
	}

	return result.String()
}

func (m *Manager) buildSelfTools(bot *MyBot) string {
	if len(bot.Tools) == 0 {
		return "none"
	}

	botTools := make(map[string]bool)
	for _, toolName := range bot.Tools {
		botTools[toolName] = true
	}

	var result strings.Builder
	tools, _ := m.store.LoadTool()
	for _, tool := range tools {
		if !botTools[tool.Name] {
			continue
		}
		result.WriteString(fmt.Sprintf(
			"- **%s** (%s): %s\n",
			tool.Name, tool.Type, tool.Desc,
		))
	}

	if result.Len() == 0 {
		return "none"
	}
	return result.String()
}

// buildMcpTools 构建MCP工具列表
func (m *Manager) buildMcpTools(bot *MyBot) string {
	var prompt strings.Builder
	mcpServ := amcp.GetMcpService(m.store)
	servers := mcpServ.ListServers()
	for _, server := range servers {
		enable := server.Status.Enable
		tools := server.Status.McpTools
		if enable && len(tools) == 0 {
			prompt.Reset()
			prompt.WriteString("error:server-no-tools")
			break
		}
		checked := server.Checked(bot)
		if len(checked) == 0 {
			continue
		}
		prompt.WriteString("## " + server.UUID + "\n")
		for _, tool := range checked {
			prompt.WriteString(fmt.Sprintf("### **%s**\n", tool.Name))
			prompt.WriteString(fmt.Sprintf("- 描述： %s\n", tool.Description))
			if data, err := tool.InputSchema.MarshalJSON(); err == nil {
				prompt.WriteString("- 入参：\n")
				prompt.WriteString("```json\n")
				prompt.WriteString(string(data))
				prompt.WriteString("\n```\n\n\n")
			}
		}
	}
	if prompt.Len() == 0 {
		return "none"
	}
	return prompt.String()
}

func (m *Manager) RefreshBot(bot *MyBot) {
	var found bool
	for idx, item := range m.mybots {
		if item.UUID == bot.UUID {
			found = true
			m.mybots[idx] = bot
		}
	}
	if !found {
		m.mybots = append(m.mybots, bot)
	}
}

func (h *Manager) UpdateEnv(cfg *entity.CfgEntity) error {
	var result error
	for key, val := range cfg.Data {
		var err error
		switch key {
		case "proxyUrl":
			err = config.Set("PROXY_URL", fmt.Sprint(val))
		case "authGate":
			err = config.Set("AUTH_GATE", fmt.Sprint(val))
		case "useDebug":
			if yes, ok := val.(bool); ok && yes {
				err = config.Set("DEBUG_MODE", "yes")
			} else {
				err = config.Set("DEBUG_MODE", "no")
			}
		case "dataPath":
			if path, _ := val.(string); path == "" {
				continue
			}
			err = config.Set("SWIFLOW_HOME", fmt.Sprint(val))
		case "ctxMsgSize":
			err = config.Set("CTX_MSG_SIZE", fmt.Sprint(val))
		case "maxCallTurns":
			err = config.Set("MAX_CALL_TURNS", fmt.Sprint(val))
		case "streamOutput":
			err = config.Set("STREAM_OUTPUT", fmt.Sprint(val))
		}
		if err != nil {
			result = err
		}
	}
	return result
}

func (m *Manager) CurrentBot() string {
	bot := config.GetStr("ACTIVE_BOT", "")
	for key, val := range m.config {
		if key != entity.KEY_ACTIVE_BOT {
			continue
		}
		data, _ := val.(map[string]any)
		if len(data) == 0 {
			continue
		}
		if data, ok := data["uuid"]; ok {
			bot, _ = data.(string)
		}
	}
	return support.Or(bot, "nobody")
}

func (m *Manager) GetStorage() (Store, error) {
	if m.store != nil {
		return m.store, nil
	}
	return m.InitStorage()
}

func (m *Manager) InitConfig() error {
	list, err := m.store.LoadCfg()
	if err != nil {
		return err
	}
	for _, item := range list {
		if item.Type == entity.KEY_CFG_DATA {
		} else {
			m.config[item.Name] = item.Data
		}
	}
	// 清除
	for _, item := range m.active {
		if !item.IsRunning() {
			item.modelClient = nil
		}
	}
	return nil
}

func (m *Manager) InitStorage() (Store, error) {
	needUpgrade := config.NeedUpgrade()
	kind := config.GetStr("STORAGE_TYPE", "sqlite")
	cfg := map[string]any{"path": config.GetWorkHome()}
	switch strings.ToLower(kind) {
	case "sqlite":
		cfg["path"] = config.SQLiteFile()
	case "mysql":
		dsn := config.MySQLDSN()
		switch dsn := dsn.(type) {
		case string:
			cfg["dsn"] = dsn
		case error:
			return nil, dsn
		}
	}
	store, err := storage.NewStorage(kind, cfg)
	if store == nil || err != nil {
		return store, err
	}
	// handle migrate
	switch strings.ToLower(kind) {
	case "sqlite", "mysql":
		if _, err = store.LoadCfg(); err != nil {
			if strings.Contains(err.Error(), "no such table") {
				needUpgrade = true
			}
		}
	}
	if needUpgrade {
		err = store.AutoMigrate()
	}
	return store, err
}

func (m *Manager) ClearProcess() {
	log.Println("[AGENT] start clear process")
	new(ability.DevAsyncCmdAbility).Clear()
}
