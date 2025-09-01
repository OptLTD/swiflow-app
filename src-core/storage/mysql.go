package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// MySQLStorage MySQL存储实现
type MySQLStorage struct {
	baseDb *sql.DB
	gormDB *gorm.DB
}

// NewMySQLStorage 创建MySQL存储实例
func NewMySQLStorage(config map[string]any) (*MySQLStorage, error) {
	var dsn string
	if val, ok := config["dsn"].(string); !ok {
		log.Printf("[MYSQL INIT]mysql dsn not exists")
		return nil, fmt.Errorf("mysql dsn not exists")
	} else {
		dsn = val + "?charset=utf8mb4&parseTime=True&loc=Local"
	}
	// 使用GORM连接数据库
	gormLogger := logger.Default.LogMode(logger.Warn)
	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		log.Printf("[MYSQL INIT]mysql connect err %v", err)
		return nil, fmt.Errorf("mysql connect err %w", err)
	}

	// 获取底层的sql.DB对象以保持兼容性
	if baseDB, err := gormDB.DB(); err != nil {
		log.Printf("[MYSQL INIT]failed to get sql.DB: %v", err)
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	} else {
		// 设置连接池参数
		baseDB.SetMaxIdleConns(10)
		baseDB.SetMaxOpenConns(100)
		baseDB.SetConnMaxLifetime(time.Hour)
		return &MySQLStorage{baseDB, gormDB}, nil
	}
}

func (s *MySQLStorage) AutoMigrate() error {
	// 自动迁移表结构
	task, bot, msg := new(TaskEntity), new(BotEntity), new(MsgEntity)
	if err := s.gormDB.AutoMigrate(task, bot, msg); err != nil {
		log.Printf("[MYSQL]failed to migrate tables: %v", err)
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	cfg, tool := new(CfgEntity), new(ToolEntity)
	if err := s.gormDB.AutoMigrate(cfg, tool); err != nil {
		log.Printf("[MYSQL]failed to migrate tables: %v", err)
		return fmt.Errorf("failed to migrate tables: %w", err)
	}

	mem, todo := new(MemEntity), new(TodoEntity)
	if err := s.gormDB.AutoMigrate(mem, todo); err != nil {
		log.Printf("[MYSQL]failed to migrate tables: %v", err)
		return fmt.Errorf("failed to migrate tables: %w", err)
	}
	return nil
}

// InitTask 初始化存储
func (s *MySQLStorage) InitTask(task *TaskEntity) error {
	if task.UUID == "" {
		return fmt.Errorf("task uuid empty")
	}

	// 使用FirstOrCreate创建记录
	if result := s.gormDB.Create(task); result.Error != nil {
		log.Printf("[mysql init]failed to init task: %v", result.Error)
		return fmt.Errorf("failed to init task: %w", result.Error)
	}
	return nil
}

// LoadTask 列出任务
func (s *MySQLStorage) LoadTask() ([]*TaskEntity, error) {
	var models []*TaskEntity
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)

	query := s.gormDB.Where("updated_at >= ?", threeMonthsAgo)
	if result := query.Order("id DESC").Find(&models); result.Error != nil {
		log.Printf("[MYSQL]failed to list tasks: %v", result.Error)
		return nil, fmt.Errorf("failed to list tasks: %w", result.Error)
	}

	return models, nil
}

func (s *MySQLStorage) FindTask(task *TaskEntity) error {
	query := s.gormDB.Where("uuid = ?", task.UUID)
	if result := query.First(&task); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load task: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load task: %v", result.Error)
			return fmt.Errorf("failed to load task: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveTask(task *TaskEntity) error {
	query := s.gormDB.Where("uuid = ?", task.UUID)
	if !task.DeletedAt.Time.IsZero() {
		if r := query.Delete(task); r.Error != nil {
			log.Printf("[MYSQL]failed to delete task: %v", r.Error)
			return fmt.Errorf("failed to delete task: %w", r.Error)
		}
	}

	update := map[string]any{
		"name": task.Name, "home": task.Home, "bots": task.Bots,
		"state": task.State, "context": task.Context,
		"command": task.Command, "process": task.Process,
	}

	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(update)),
	}
	query = s.gormDB.Model(task).Clauses(clauses).Assign(update)
	if result := query.Create(&task); result.Error != nil {
		log.Printf("[MYSQL]failed to save task: %v", result.Error)
		return fmt.Errorf("failed to save task: %w", result.Error)
	}
	return nil
}

func (s *MySQLStorage) FindMsg(msg *MsgEntity) error {
	query := s.gormDB.Where("msg_id = ?", msg.MsgId)
	if result := query.First(&msg); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load msg: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load msg: %v", result.Error)
			return fmt.Errorf("failed to load msg: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveMsg(msg *MsgEntity) error {
	// 构建更新数据
	updates := map[string]any{
		"op_type": msg.OpType,
		"task_id": msg.TaskId,
		"pre_msg": msg.PreMsg,
	}
	if msg.IsSend {
		updates["request"] = msg.Request
		updates["send_at"] = msg.SendAt
	} else if msg.Context == "" {
		updates["recv_at"] = msg.RecvAt
		updates["respond"] = msg.Respond
	} else {
		updates["respond"] = msg.Respond
		updates["context"] = msg.Context
	}

	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "msg_id"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(updates)),
	}
	query := s.gormDB.Model(msg).Clauses(clauses).Assign(updates)
	if result := query.Create(msg); result.Error != nil {
		log.Printf("[MYSQL]failed to save msg: %v", result.Error)
		return fmt.Errorf("failed to save msg: %w", result.Error)
	}
	return nil
}

// LoadMsg 加载消息数据
func (s *MySQLStorage) LoadMsg(task *TaskEntity) ([]*MsgEntity, error) {
	// var msgs []*Msg
	var result []*MsgEntity

	// 使用GORM查询消息
	query := s.gormDB.Where("task_id = ?", task.UUID).Order("id ASC")
	if err := query.Find(&result).Error; err != nil {
		log.Printf("[MYSQL]failed to query msgs: %v", err)
		return result, fmt.Errorf("failed to query msgs: %w", err)
	}
	return result, nil
}

func (s *MySQLStorage) FindBot(bot *BotEntity) error {
	query := s.gormDB.Where("uuid = ?", bot.UUID)
	if result := query.First(&bot); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load bot: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load bot: %v", result.Error)
			return fmt.Errorf("failed to load bot: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveBot(bot *BotEntity) error {
	query := s.gormDB.Model(bot).Where("uuid = ?", bot.UUID)
	if !bot.DeletedAt.Time.IsZero() {
		if r := query.Delete(bot); r.Error != nil {
			log.Printf("[MYSQL]failed to delete bot: %v", r.Error)
			return fmt.Errorf("failed to delete bot: %w", r.Error)
		}
		return nil
	}

	updates := map[string]any{
		"name": bot.Name, "type": bot.Type, "desc": bot.Desc,
		"emoji": bot.Emoji, "tools": bot.Tools, "deleted_at": nil,
		"sys_prompt": bot.SysPrompt, "use_prompt": bot.UsePrompt,
		"home": bot.Home, "provider": bot.Provider,
	}

	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(updates)),
	}
	query = s.gormDB.Model(bot).Clauses(clauses).Assign(updates)
	if r := query.Create(bot); r.Error != nil {
		log.Printf("[MYSQL]failed to save bot: %v", r.Error)
		return fmt.Errorf("failed to save bot: %w", r.Error)
	}
	return nil
}

// LoadMsgs 加载消息数据
func (s *MySQLStorage) LoadBot() ([]*BotEntity, error) {
	var result []*BotEntity
	query := s.gormDB.Model(&BotEntity{}).Order("id desc")
	if r := query.Preload("Memories").Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query bots: %v", r.Error)
		return nil, fmt.Errorf("failed to query bots: %w", r.Error)
	}
	return result, nil
}

func (s *MySQLStorage) FindCfg(cfg *CfgEntity) error {
	query := s.gormDB.Where("type = ? AND name = ?", cfg.Type, cfg.Name)
	if result := query.First(&cfg); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load cfg: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load cfg: %v", result.Error)
			return fmt.Errorf("failed to load cfg: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveCfg(cfg *CfgEntity) error {
	if !cfg.DeletedAt.Time.IsZero() {
		where := map[string]any{"type": cfg.Type, "name": cfg.Name}
		if r := s.gormDB.Where(where).Delete(cfg); r.Error != nil {
			log.Printf("[MYSQL]failed to delete cfg: %v", r.Error)
			return fmt.Errorf("failed to delete cfg: %w", r.Error)
		}
		return nil
	}

	update := map[string]any{
		"type": cfg.Type, "name": cfg.Name,
		"data": cfg.Data, "deleted_at": nil,
	}
	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "type"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(update)),
	}
	query := s.gormDB.Model(cfg).Clauses(clauses).Assign(update)
	if r := query.Create(cfg); r.Error != nil {
		log.Printf("[MYSQL]failed to save config: %v", r.Error)
		return fmt.Errorf("failed to save config: %w", r.Error)
	}
	return nil
}

// LoadMsgs 加载消息数据
func (s *MySQLStorage) LoadCfg() ([]*CfgEntity, error) {
	var result []*CfgEntity
	query := s.gormDB.Model(&CfgEntity{}).Order("id ASC")
	if r := query.Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query cfg: %v", r.Error)
		return nil, fmt.Errorf("failed to query cfg: %w", r.Error)
	}
	return result, nil
}

func (s *MySQLStorage) FindMem(mem *MemEntity) error {
	query := s.gormDB.Where("id = ?", mem.ID)
	if result := query.First(&mem); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load mem: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load mem: %v", result.Error)
			return fmt.Errorf("failed to load mem: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveMem(mem *MemEntity) error {
	// 如果是删除操作
	if !mem.DeletedAt.Time.IsZero() {
		if r := s.gormDB.Model(mem).Where("id = ?", mem.ID).Delete(mem); r.Error != nil {
			log.Printf("[MYSQL]failed to delete mem: %v", r.Error)
			return fmt.Errorf("failed to delete mem: %w", r.Error)
		}
		return nil
	}

	// 如果是更新操作（ID不为0）
	if mem.ID != 0 {
		updates := map[string]any{
			"type": mem.Type, "subject": mem.Subject,
			"bot": mem.Bot, "content": mem.Content,
		}
		if r := s.gormDB.Model(mem).Where("id = ?", mem.ID).Updates(updates); r.Error != nil {
			log.Printf("[MYSQL]failed to update mem: %v", r.Error)
			return fmt.Errorf("failed to update mem: %w", r.Error)
		}
		return nil
	}

	// 如果是创建新记录
	if r := s.gormDB.Model(mem).Create(mem); r.Error != nil {
		log.Printf("[MYSQL]failed to create mem: %v", r.Error)
		return fmt.Errorf("failed to create mem: %w", r.Error)
	}
	return nil
}

func (s *MySQLStorage) LoadMem() ([]*MemEntity, error) {
	var result []*MemEntity
	query := s.gormDB.Model(&MemEntity{}).Order("id desc")
	if r := query.Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query mem: %v", r.Error)
		return nil, fmt.Errorf("failed to query mem: %w", r.Error)
	}
	return result, nil
}

func (s *MySQLStorage) FindTool(tool *ToolEntity) error {
	query := s.gormDB.Where("uuid = ?", tool.UUID)
	if result := query.First(&tool); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load tool: %w", result.Error)
		} else {
			log.Printf("[MYSQL]failed to load tool: %v", result.Error)
			return fmt.Errorf("failed to load tool: %w", result.Error)
		}
	}
	return nil
}

func (s *MySQLStorage) SaveTool(tool *ToolEntity) error {
	// 如果是删除操作
	if !tool.DeletedAt.Time.IsZero() {
		if r := s.gormDB.Model(tool).Where("uuid = ?", tool.UUID).Delete(tool); r.Error != nil {
			log.Printf("[MYSQL]failed to delete tool: %v", r.Error)
			return fmt.Errorf("failed to delete tool: %w", r.Error)
		}
		return nil
	}

	// 构建更新数据
	updates := map[string]any{
		"uuid": tool.UUID, "type": tool.Type,
		"name": tool.Name, "desc": tool.Desc,
		"code": tool.Code, "deps": tool.Deps,
	}

	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(updates)),
	}
	query := s.gormDB.Model(tool).Clauses(clauses).Assign(updates)
	if r := query.Create(tool); r.Error != nil {
		log.Printf("[MYSQL]failed to save tool: %v", r.Error)
		return fmt.Errorf("failed to save tool: %w", r.Error)
	}
	return nil
}

func (s *MySQLStorage) LoadTool() ([]*ToolEntity, error) {
	var result []*ToolEntity
	query := s.gormDB.Model(&ToolEntity{}).Order("id desc")
	if r := query.Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query tools: %v", r.Error)
		return nil, fmt.Errorf("failed to query tools: %w", r.Error)
	}
	return result, nil
}

// TodoEntity 相关方法
func (s *MySQLStorage) FindTodo(todo *TodoEntity) error {
	query := s.gormDB.Where("uuid = ?", todo.UUID)
	if result := query.First(&todo); result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to load todo: %w", result.Error)
		}
		log.Printf("[MYSQL]failed to load todo: %v", result.Error)
		return fmt.Errorf("failed to load todo: %w", result.Error)
	}
	return nil
}

func (s *MySQLStorage) SaveTodo(todo *TodoEntity) error {
	updates := map[string]any{
		"uuid": todo.UUID, "task": todo.Task,
		"time": todo.Time, "todo": todo.Todo,
		"done": todo.Done,
	}
	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.AssignmentColumns(maputil.Keys(updates)),
	}
	query := s.gormDB.Model(todo).Clauses(clauses).Assign(updates)
	if r := query.Create(todo); r.Error != nil {
		log.Printf("[MYSQL]failed to save todo: %v", r.Error)
		return fmt.Errorf("failed to save todo: %w", r.Error)
	}
	return nil
}

func (s *MySQLStorage) LoadTodo() ([]*TodoEntity, error) {
	var result []*TodoEntity
	query := s.gormDB.Model(&TodoEntity{}).Order("id DESC")
	if r := query.Where("done = ?", 0).Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query todos: %v", r.Error)
		return nil, fmt.Errorf("failed to query todos: %w", r.Error)
	}
	return result, nil
}

func (s *MySQLStorage) LoadDone() ([]*TodoEntity, error) {
	var result []*TodoEntity
	query := s.gormDB.Model(&TodoEntity{}).Order("id DESC")
	if r := query.Where("done = ?", 1).Find(&result); r.Error != nil {
		log.Printf("[MYSQL]failed to query done todos: %v", r.Error)
		return nil, fmt.Errorf("failed to query done todos: %w", r.Error)
	}
	return result, nil
}
