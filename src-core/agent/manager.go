package agent

import (
	"fmt"
	"log"
	"strings"
	"swiflow/ability"
	"swiflow/action"
	"swiflow/amcp"
	"swiflow/builtin"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/model"
	"swiflow/storage"
	"swiflow/support"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
)

type Store = storage.MyStore
type MyMsg = entity.MsgEntity
type MyTask = entity.TaskEntity
type Worker = entity.BotEntity
type Payload = action.Payload

type Manager struct {
	store Store

	workers []*Worker
	configs map[string]any

	executors map[string]*Executor
	subagents map[string]*SubAgent
	// ensure event listeners registered only once
	initOnce sync.Once
}

func FromAgents(agents []*Worker) (*Manager, error) {
	m := &Manager{}
	// init storage
	m.workers = agents
	store, err := m.InitStorage()
	if err != nil || store == nil {
		log.Println("[AGENT] init store error", err)
		return nil, fmt.Errorf("init store error: %v", err)
	} else {
		m.store = store
	}
	m.configs = map[string]any{}
	m.executors = map[string]*Executor{}
	m.subagents = map[string]*SubAgent{}

	provider := config.Get("SWIFLOW_PROVIDER")
	name, model, _ := strings.Cut(provider, "@")
	m.configs[entity.KEY_USE_MODEL] = map[string]any{
		"provider": name, "useModel": model,
		"apiUrl": config.Get("SWIFLOW_API_URL"),
		"apiKey": config.Get("SWIFLOW_API_KEY"),
	}
	return m, nil
}

func NewManager() (*Manager, error) {
	m := &Manager{}

	// init storage
	m.configs = map[string]any{}
	m.executors = map[string]*Executor{}
	m.subagents = map[string]*SubAgent{}
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
	if m.workers, err = m.store.LoadBot(); err == nil { // Call without parameters to maintain existing behavior
		return nil
	}
	log.Println("[AGENT] init worker error", err)
	return fmt.Errorf("init worker error: %v", err)
}

// onSubtask handles "subtask" events
func (m *Manager) onSubtask(tid string, data any) {
	if tid == "" {
		return
	}
	executor, _ := m.FindExecutor(tid)
	if executor == nil {
		return
	}

	// find from executor
	task := executor.context.mytask
	leader := executor.context.worker
	switch act := data.(type) {
	case *action.StartSubtask:
		log.Println("[AGENT] start subtask", tid, act.SubAgent)
		subagent := m.GetSubAgent(act.SubAgent, leader, task)
		if subagent.target != nil && subagent.target.UUID != tid {
			return
		}
		subagent.OnStart(act)
		log.Println("[AGENT] booted subtask", tid, act.SubAgent)
	case *action.AbortSubtask:
		log.Println("[AGENT] abort subtask", tid, act.SubAgent)
		subagent := m.GetSubAgent(act.SubAgent, leader, task)
		if subagent.target != nil && subagent.target.UUID != tid {
			return
		}
		subagent.OnAbort(act)
		log.Println("[AGENT] leave subtask", tid, act.SubAgent)
	}
}

// onComplete handles "complete" events
func (m *Manager) onComplete(tid string, data any) {
	if tid == "" {
		return
	}
	var subagent *SubAgent
	for _, item := range m.subagents {
		if item.mytask == nil {
			continue
		}
		if item.mytask.UUID == tid {
			subagent = item
			break
		}
	}
	log.Println("[AGENT] task complete", tid)
	act, _ := data.(*action.Complete)
	if subagent != nil && act != nil {
		subagent.OnComplete(act)
	} else {
		log.Println("[AGENT] subagent not found", tid)
	}
}

func (m *Manager) Start(input action.Input, task *MyTask, leader *Worker) {
	// debug mode, it's worker
	if leader.Leader != "" {
		task.Home = config.CurrentHome()
		m.Handle(input, task, leader)
		return
	}
	// register event listeners only once with method handlers
	m.initOnce.Do(func() {
		support.Once("subtask", m.onSubtask)
		support.Once("complete", m.onComplete)
	})

	task.Group = task.UUID
	m.store.SaveTask(task)
	go m.Handle(input, task, leader)
}

func (m *Manager) Handle(input action.Input, task *MyTask, worker *Worker) {
	var executor *Executor
	if e := m.LoadExecutor(task, worker); e != nil {
		executor = e
	}

	if strings.HasPrefix(task.UUID, "#debug#") {
		worker.UUID, worker.Type = task.Name, "debug"
		mcpServ := amcp.GetMcpService(m.store)
		server := &amcp.McpServer{UUID: task.Name}
		if err := mcpServ.QueryServer(server); err != nil {
			log.Println("[AGENT] query server error", err)
			support.Emit("errors", task.UUID, "mcp server not existed")
			return
		}
		if len(server.Status.McpTools) == 0 {
			log.Println("[AGENT] start mcp server", server.UUID)
			// need load package first
			err := mcpServ.ServerStatus(server)
			if len(server.Status.McpTools) == 0 {
				log.Println("[AGENT] query status error", err)
				support.Emit("errors", task.UUID, "mcp server in booting")
				return
			}
		}

		executor.context.store = mcpServ.GetMockStore()
		log.Println("[AGENT] worker", worker.UUID, worker.Type)
		log.Println("[AGENT] prompt", executor.context.usePrompt)
	}

	if executor == nil {
		support.Emit("errors", task.UUID, "load executor error")
		return
	}
	if executor.modelClient == nil {
		if cfg := m.GetLLMConfig(worker.Provider); cfg == nil {
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
		UUID: uuid, BotId: m.CurrentWorker(),
		Name: support.Substring(name, 80),
	}

	if err := m.store.InitTask(task); err != nil {
		log.Println("[AGENT] init task err", err)
		return nil, err
	}
	return task, nil
}
func (m *Manager) FromIntent(worker string, intent *builtin.IntentRequest) (*MyTask, error) {
	uuid, _ := support.UniqueID()
	subtask := &MyTask{
		UUID: "im-" + uuid, BotId: worker,
		Name: support.Substring(intent.Content, 80),
		// log source and session id
		SessID: intent.Session, Source: intent.Source,
	}

	if err := m.store.InitTask(subtask); err != nil {
		log.Println("[AGENT] init im task err", err)
		return nil, err
	}
	return subtask, nil
}

func (m *Manager) InitSubtask(worker string, group string) (*MyTask, error) {
	uuid, _ := support.UniqueID()
	subtask := &MyTask{
		UUID:  "sub-" + uuid,
		BotId: worker, Group: group,
	}

	if err := m.store.InitTask(subtask); err != nil {
		log.Println("[AGENT] init subtask err", err)
		return nil, err
	}
	return subtask, nil
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

func (m *Manager) QueryWorker(uuid string) (*Worker, error) {
	var worker = &Worker{UUID: uuid}
	if err := m.store.FindBot(worker); err != nil {
		log.Println("[AGENT] find bot err:", uuid, err)
		return nil, err
	}
	return worker, nil
}

func (m *Manager) GetWorker(uuid string) (*Worker, error) {
	if uuid == "" {
		uuid = m.CurrentWorker()
	}
	for _, worker := range m.workers {
		if worker.UUID == uuid {
			return worker, nil
		}
	}
	if uuid == "nobody" && len(m.workers) > 0 {
		return m.workers[0], nil
	}
	log.Println("[AGENT] get worker err", uuid)
	return nil, fmt.Errorf("not found: %s", uuid)
}

func (m *Manager) GetLLMConfig(name string) *model.LLMConfig {
	data := map[string]any{}
	cfg := &model.LLMConfig{}
	provider := strings.ToLower(name)
	if provider != "" {
		val, _ := m.configs[provider]
		data, _ = val.(map[string]any)
	}
	if provider == "" || len(data) == 0 {
		val, _ := m.configs[entity.KEY_USE_MODEL]
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

func (m *Manager) GetExecutor(task *MyTask, worker *Worker) *Executor {
	payload := &Payload{
		UUID: task.UUID,
		Time: time.Now(),
		Home: task.Home,
	}
	context := &Context{
		mytask: task,
		worker: worker,
		store:  m.store,
	}
	executor := &Executor{
		UUID:    task.UUID,
		context: context,
		payload: payload,
	}
	if cfg := m.GetLLMConfig(worker.Provider); cfg != nil {
		cfg.TaskId = task.UUID
		c := model.GetClient(cfg)
		executor.modelClient = c
	}

	switch worker.Type {
	case AGENT_DEBUG, AGENT_BASIC:
	case AGENT_LEADER, AGENT_WORKER:
	default:
		if worker.Leader != "" {
			worker.Type = AGENT_WORKER
		} else {
			worker.Type = AGENT_LEADER
		}
	}

	return executor
}

func (m *Manager) LoadExecutor(task *MyTask, worker *Worker) *Executor {
	key := task.UUID + "-" + worker.UUID
	if executor, ok := m.executors[key]; ok {
		return executor
	}
	executor := m.GetExecutor(task, worker)
	if executor != nil {
		m.executors[key] = executor
		return executor
	}
	return nil
}
func (m *Manager) FindExecutor(tid string) (*Executor, error) {
	for key, executor := range m.executors {
		if strings.HasPrefix(key, tid) {
			return executor, nil
		}
	}
	return nil, fmt.Errorf("executor not found")
}

// GetSubAgent creates or retrieves a SubAgent instance for the given task and leader
func (m *Manager) GetSubAgent(key string, leader *Worker, task *MyTask) *SubAgent {
	if subagent, ok := m.subagents[key]; ok {
		return subagent
	}

	subagent := &SubAgent{
		leader: leader,
		target: task,
		parent: m,
	}
	m.subagents[key] = subagent
	return subagent
}

func (m *Manager) GetMemory(worker *Worker) string {
	var memory strings.Builder
	for _, mem := range worker.Memories {
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

// getMcpToolsInfo 构建MCP工具列表
// getMcpToolsInfo moved to amcp.McpService.BuildToolsPrompt

func (m *Manager) ResetWorker(worker *Worker) {
	var found bool
	for idx, item := range m.workers {
		if item.UUID == worker.UUID {
			found = true
			m.workers[idx] = worker
		}
	}
	if !found {
		m.workers = append(m.workers, worker)
	}
}

func (h *Manager) UpdateEnv(cfg *entity.CfgEntity) error {
	var result error
	for key, val := range cfg.Data {
		var err error
		switch key {
		case "useProxyUrl":
			err = config.Set("PROXY_URL", fmt.Sprint(val))
		case "authGateway":
			err = config.Set("AUTH_GATE", fmt.Sprint(val))
		case "useDebugMode":
			if yes, ok := val.(bool); ok && yes {
				err = config.Set("DEBUG_MODE", "yes")
			} else {
				err = config.Set("DEBUG_MODE", "no")
			}
		case "useSubAgent":
			if yes, ok := val.(bool); ok && yes {
				err = config.Set("USE_SUBAGENT", "yes")
			} else {
				err = config.Set("USE_SUBAGENT", "no")
			}
		case "useWorkPath":
			if path, _ := val.(string); path == "" {
				baseHome := config.GetWorkPath("secrets")
				config.Set("CURRENT_HOME", baseHome)
				continue
			}

			origin := config.GetStr("CURRENT_HOME", "")
			if origin != "" && origin != fmt.Sprint(val) {
				config.Set("CURRENT_HOME", fmt.Sprint(val))
				support.Emit("mcp-reboot", "UpdateEnv", nil)
			} else {
				config.Set("CURRENT_HOME", fmt.Sprint(val))
			}
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

func (m *Manager) CurrentWorker() string {
	uuid := config.GetStr("USE_WORKER", "")
	for key, val := range m.configs {
		if key != entity.KEY_USE_WORKER {
			continue
		}
		data, _ := val.(map[string]any)
		if len(data) == 0 {
			continue
		}
		if data, ok := data["uuid"]; ok {
			uuid, _ = data.(string)
		}
	}
	return support.Or(uuid, "nobody")
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
			m.configs[item.Name] = item.Data
		}
	}
	// 清除
	for _, item := range m.executors {
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
		if _, err = store.LoadCfg(); err != nil { // Call without parameters to maintain existing behavior
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
