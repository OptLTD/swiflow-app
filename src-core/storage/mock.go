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

func (m *MockStore) LoadTask(query ...any) ([]*TaskEntity, error) {
	// Mock implementation simply returns all tasks
	// In a real implementation, you would filter based on query parameters
	return m.tasks, nil
}

func (m *MockStore) FindMsg(msg *MsgEntity) error {
	for _, mmsg := range m.msgs {
		if mmsg.UniqId == msg.UniqId {
			*msg = *mmsg
			return nil
		}
	}
	return nil
}

func (m *MockStore) SaveMsg(msg *MsgEntity) error {
	for i, mmsg := range m.msgs {
		if mmsg.UniqId == msg.UniqId {
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

// LoadBot loads bots with optional query parameters (mock implementation returns all bots)
func (m *MockStore) LoadBot(query ...any) ([]*BotEntity, error) {
	// Note: Mock implementation returns all bots regardless of query parameters
	// In a real implementation, query parameters would be used to filter results
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

// LoadCfg loads configurations with optional query parameters (mock implementation returns all configs)
func (m *MockStore) LoadCfg(query ...any) ([]*CfgEntity, error) {
	// Note: Mock implementation returns all configs regardless of query parameters
	// In a real implementation, query parameters would be used to filter results
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

// LoadMem loads memories with optional query parameters (mock implementation returns all memories)
func (m *MockStore) LoadMem(query ...any) ([]*MemEntity, error) {
	// Note: Mock implementation returns all memories regardless of query parameters
	// In a real implementation, query parameters would be used to filter results
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

// LoadTool loads tools with optional query parameters (mock implementation returns all tools)
func (m *MockStore) LoadTool(query ...any) ([]*ToolEntity, error) {
	// Note: Mock implementation returns all tools regardless of query parameters
	// In a real implementation, query parameters would be used to filter results
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

// LoadTodo loads todos with optional query parameters (mock implementation returns all todos)
func (m *MockStore) LoadTodo(query ...any) ([]*TodoEntity, error) {
	var result []*TodoEntity

	// If no query parameters provided, default to undone todos (backward compatibility)
	if len(query) == 0 {
		for _, t := range m.todos {
			if t.Done == 0 {
				result = append(result, t)
			}
		}
	} else {
		// For mock implementation, we support basic done status filtering
		// First parameter should be "done = ?" and second should be the value (0 or 1)
		if len(query) >= 2 {
			if query[0] == "done = ?" {
				doneValue := query[1]
				for _, t := range m.todos {
					if (doneValue == 0 && t.Done == 0) || (doneValue == 1 && t.Done != 0) {
						result = append(result, t)
					}
				}
			} else {
				// For other query types, return all todos
				result = m.todos
			}
		} else {
			// Fallback to all todos
			result = m.todos
		}
	}

	return result, nil
}
