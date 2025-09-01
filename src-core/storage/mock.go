package storage

// MockStore 是一个模拟的存储实现，用于测试
type MockStore struct {
	bots  []*BotEntity
	msgs  []*MsgEntity
	cfgs  []*CfgEntity
	mems  []*MemEntity
	tools []*ToolEntity
	tasks []*TaskEntity
	todos []*TodoEntity
}

// NewMockStore 创建一个新的 MockStore 实例
func NewMockStore() *MockStore {
	return &MockStore{
		bots:  make([]*BotEntity, 0),
		msgs:  make([]*MsgEntity, 0),
		cfgs:  make([]*CfgEntity, 0),
		mems:  make([]*MemEntity, 0),
		tools: make([]*ToolEntity, 0),
		tasks: make([]*TaskEntity, 0),
		todos: make([]*TodoEntity, 0),
	}
}

// SetBots 设置机器人列表
func (m *MockStore) SetBots(bots []*BotEntity) {
	m.bots = bots
}

// SetTools 设置工具列表
func (m *MockStore) SetTools(tools []*ToolEntity) {
	m.tools = tools
}

// SetTasks 设置任务列表
func (m *MockStore) SetTasks(tasks []*TaskEntity) {
	m.tasks = tasks
}

// SetMsgs 设置消息列表
func (m *MockStore) SetMsgs(msgs []*MsgEntity) {
	m.msgs = msgs
}

// SetCfgs 设置配置列表
func (m *MockStore) SetCfgs(cfgs []*CfgEntity) {
	m.cfgs = cfgs
}

// SetMems 设置内存列表
func (m *MockStore) SetMems(mems []*MemEntity) {
	m.mems = mems
}

// GetBots 获取机器人列表
func (m *MockStore) GetBots() []*BotEntity {
	return m.bots
}

// GetTools 获取工具列表
func (m *MockStore) GetTools() []*ToolEntity {
	return m.tools
}

// 实现 MyStore 接口的所有方法

func (m *MockStore) AutoMigrate() error {
	return nil
}

func (m *MockStore) InitTask(tool *TaskEntity) error {
	return nil
}

func (m *MockStore) FindTask(tool *TaskEntity) error {
	for _, j := range m.tasks {
		if j.ID == tool.ID || j.Name == tool.Name {
			*tool = *j
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveTask(tool *TaskEntity) error {
	for i, j := range m.tasks {
		if j.ID == tool.ID || j.Name == tool.Name {
			m.tasks[i] = tool
			return nil
		}
	}
	m.tasks = append(m.tasks, tool)
	return nil
}

func (m *MockStore) LoadTask() ([]*TaskEntity, error) {
	return m.tasks, nil
}

func (m *MockStore) FindMsg(msg *MsgEntity) error {
	for _, mmsg := range m.msgs {
		if mmsg.MsgId == msg.MsgId {
			*msg = *mmsg
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveMsg(msg *MsgEntity) error {
	for i, mmsg := range m.msgs {
		if mmsg.MsgId == msg.MsgId {
			m.msgs[i] = msg
			return nil
		}
	}
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *MockStore) LoadMsg(task *TaskEntity) ([]*MsgEntity, error) {
	var result = []*MsgEntity{}
	if len(m.msgs) == 0 {
		return result, nil
	}
	if task == nil || task.UUID == "" {
		return result, nil
	}
	for _, msg := range m.msgs {
		if msg.TaskId != task.UUID {
			continue
		}
		result = append(result, msg)
	}
	return result, nil
}
func (m *MockStore) ClearMsg(task *TaskEntity) error {
	if len(m.msgs) == 0 {
		return nil
	}
	if task == nil || task.UUID == "" {
		return nil
	}
	var result = []*MsgEntity{}
	for _, msg := range m.msgs {
		if msg.TaskId == task.UUID {
			continue
		}
		result = append(result, msg)
	}
	m.msgs = result
	return nil
}

func (m *MockStore) FindBot(bot *BotEntity) error {
	for _, b := range m.bots {
		if b.ID == bot.ID || b.Name == bot.Name {
			*bot = *b
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveBot(bot *BotEntity) error {
	for i, b := range m.bots {
		if b.ID == bot.ID || b.Name == bot.Name {
			m.bots[i] = bot
			return nil
		}
	}
	m.bots = append(m.bots, bot)
	return nil
}

func (m *MockStore) LoadBot() ([]*BotEntity, error) {
	return m.bots, nil
}

func (m *MockStore) FindCfg(cfg *CfgEntity) error {
	for _, c := range m.cfgs {
		if c.Type == cfg.Type && c.Name == cfg.Name {
			cfg.ID = c.ID
			cfg.Data = c.Data
			return nil
		}
	}
	cfg.Data = nil
	return nil
}

func (m *MockStore) SaveCfg(cfg *CfgEntity) error {
	for i, c := range m.cfgs {
		if c.Type == cfg.Type && c.Name == cfg.Name {
			m.cfgs[i] = cfg
			return nil
		}
	}
	m.cfgs = append(m.cfgs, cfg)
	return nil
}

func (m *MockStore) LoadCfg() ([]*CfgEntity, error) {
	return m.cfgs, nil
}

func (m *MockStore) FindMem(mem *MemEntity) error {
	for _, mm := range m.mems {
		if mm.ID == mem.ID || (mm.Bot == mem.Bot && mm.Type == mem.Type) {
			*mem = *mm
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveMem(mem *MemEntity) error {
	for i, mm := range m.mems {
		if mm.ID == mem.ID || (mm.Bot == mem.Bot && mm.Type == mem.Type) {
			m.mems[i] = mem
			return nil
		}
	}
	m.mems = append(m.mems, mem)
	return nil
}

func (m *MockStore) LoadMem() ([]*MemEntity, error) {
	return m.mems, nil
}

func (m *MockStore) FindTool(tool *ToolEntity) error {
	for _, t := range m.tools {
		if t.ID == tool.ID || t.Name == tool.Name {
			*tool = *t
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveTool(tool *ToolEntity) error {
	for i, t := range m.tools {
		if t.ID == tool.ID || t.Name == tool.Name {
			m.tools[i] = tool
			return nil
		}
	}
	m.tools = append(m.tools, tool)
	return nil
}

func (m *MockStore) LoadTool() ([]*ToolEntity, error) {
	return m.tools, nil
}

func (m *MockStore) FindTodo(todo *TodoEntity) error {
	for _, t := range m.todos {
		if t.ID == todo.ID || t.UUID == todo.UUID {
			*todo = *t
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveTodo(todo *TodoEntity) error {
	for i, t := range m.todos {
		if t.ID == todo.ID || t.UUID == todo.UUID {
			m.todos[i] = todo
			return nil
		}
	}
	m.todos = append(m.todos, todo)
	return nil
}

func (m *MockStore) LoadTodo() ([]*TodoEntity, error) {
	return m.todos, nil
}

func (m *MockStore) LoadDone() ([]*TodoEntity, error) {
	var result []*TodoEntity
	for _, t := range m.todos {
		if t.Done != 0 {
			result = append(result, t)
		}
	}
	return result, nil
}
