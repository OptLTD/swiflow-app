package httpd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"runtime"
	"strings"
	"swiflow/ability"
	"swiflow/amcp"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/initial"
	"swiflow/storage"
	"swiflow/support"
	"time"
)

type HttpServie struct {
	store storage.MyStore
}

func (h *HttpServie) Run(task *entity.TaskEntity, cmd string) error {
	if task.Command == "" {
		log.Println("error empty command")
		return fmt.Errorf("empty command")
	}

	dev := ability.DevCommandAbility{
		Home: task.Home,
	}

	// operate
	switch cmd {
	case "stop":
		if err := dev.Stop(task.Process); err != nil {
			log.Default().Printf("stop program: %v", err)
			return fmt.Errorf("stop program: %v", err)
		} else {
			task.Process = 0
		}
	case "start":
		if pid := task.Process; pid > 0 {
			dev.Stop(task.Process)
		}
		if pid, err := dev.Start(task.Command); err != nil {
			log.Default().Printf("start program: %v", dev.Logs())
			return fmt.Errorf("start program: %v", err)
		} else {
			log.Default().Printf("start: %v %d", task.Command, pid)
			task.Process = pid
		}
	case "status":
		status := dev.Status(task.Process)
		task.Process = support.If(status, task.Process, 0)
	default:
		return fmt.Errorf("undefined command")
	}
	if err := h.store.SaveTask(task); err != nil {
		return fmt.Errorf("save task fail: %v", err)
	}
	return nil
}

func (h *HttpServie) GetMcpEnv() any {
	var result = map[string]any{}
	var python = "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}

	dev := ability.DevCommandAbility{
		Home: config.GetWorkPath(""),
	}
	// python env check
	if data, err := dev.Run(python, 30*time.Second, "-V"); err == nil {
		result["python"] = strings.TrimSpace(string(data))
	}
	// uvx env check
	if data, err := dev.Run("uvx", 30*time.Second, "-V"); err == nil {
		env := strings.TrimSpace(string(data))
		result["uvx"] = strings.Split(env, "(")[0]
	}

	// node.js env check
	if data, err := dev.Run("node", 30*time.Second, "-v"); err == nil {
		result["nodejs"] = strings.TrimSpace(string(data))
	}

	// npx env check
	if data, err := dev.Run("npx", 30*time.Second, "-v"); err == nil {
		result["npx"] = strings.TrimSpace(string(data))
	}

	file := ability.FileSystemAbility{
		Base: config.GetWorkPath(""),
		Path: "install.lock",
	}
	if data, err := file.Read(); err == nil {
		result["running"] = strings.TrimSpace(string(data))
	}
	if config.IsWindows() {
		result["windows"] = true
	}
	return result
}

func (h *HttpServie) GetNetEnv() any {
	var domains = []string{
		"google.com",
		"youtube.com",
		"facebook.com",
		"instagram.com",
	}
	result := BatchCheck(domains)
	if len(result) > 2 {
		return "mainland"
	}
	return "standard"
}

func (h *HttpServie) GetRelease() any {
	releaseUrl := "https://dl.swiflow.cc/release.json"
	resp, err := http.Get(releaseUrl)
	if err != nil {
		return fmt.Errorf("fetch release failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("fetch release status: %v", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read release body failed: %v", err)
	}

	var rel map[string]any
	if err := json.Unmarshal(data, &rel); err != nil {
		return fmt.Errorf("unmarshal releases failed: %v", err)
	}

	var release struct {
		Body string `json:"body"`
		Name string `json:"name"`
		Type string `json:"type"`
		Hash string `json:"hash"`
		Url  string `json:"url"`
		Tag  string `json:"tag"`
	}

	currVer := config.GetVersion()
	release.Tag, _ = rel["tag"].(string)
	release.Body, _ = rel["body"].(string)
	assets, ok := rel["assets"].([]any)
	if !ok || !support.IsNewVer(release.Tag, currVer) {
		return nil
	}

	archMap := map[string]string{
		"darwin/amd64":  "x64.dmg",
		"darwin/arm64":  "aarch64.dmg",
		"windows/amd64": "x64-setup.exe",
	}
	key := runtime.GOOS + "/" + runtime.GOARCH
	for _, a := range assets {
		release.Name = ""
		asset, ok := a.(map[string]any)
		if !ok {
			continue
		}
		if val, ok := asset["name"]; ok {
			release.Name = val.(string)
		}
		if val, ok := asset["hash"]; ok {
			release.Hash = val.(string)
		}
		if val, ok := asset["type"]; ok {
			release.Type = val.(string)
		}
		if val, ok := asset["url"]; ok {
			release.Url = val.(string)
		}
		if strings.HasSuffix(release.Name, archMap[key]) {
			return release
		}
	}
	return nil
}

func (h *HttpServie) ReadTo(r io.Reader, v any) error {
	val, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(val, &v)
	if err != nil {
		return err
	}
	return nil
}

func (h *HttpServie) ReadMap(r io.Reader) any {
	var data = map[string]any{}
	val, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	err = json.Unmarshal(val, &data)
	if err != nil {
		return err
	}
	return data
}

func (h *HttpServie) InitMcpEnv(name string, env string) any {
	log.Printf("InitMcpEnv started: name=%s, env=%s", name, env)

	// Validate environment name
	if name != "js-npx" && name != "uvx-py" {
		log.Printf("Invalid environment name: %s", name)
		return fmt.Errorf("wrong name")
	}

	// Initialize file system ability
	file := ability.FileSystemAbility{
		Base: config.GetWorkPath(""),
	}

	var cmd string
	var args []string
	env = support.Or(env, "mainland")

	log.Printf("Using path: %s, env mode: %s", file.Base, env)
	// Determine platform-specific script and command
	switch runtime.GOOS {
	case "windows":
		file.Path = fmt.Sprintf("win-%s.ps1", name)
		cmd, args = "powershell", []string{
			"-NoProfile", "-ExecutionPolicy",
			"Bypass", "-File", file.Path, "-mode", env,
		}
	case "darwin":
		file.Path = fmt.Sprintf("mac-%s.sh", name)
		cmd, args = "sh", []string{file.Path, env}
	default:
		file.Path = fmt.Sprintf("linux-%s.sh", name)
		cmd, args = "sh", []string{file.Path, env}
	}
	// bin, err := initial.GetScript(file.Path)
	if bin, err := initial.GetScript(file.Path); len(bin) > 0 {
		log.Printf("Script content retrieved, size: %d bytes", len(bin))
		if err := file.Write(string(bin)); err != nil {
			log.Printf("Failed to write script file: %v", err)
			return err
		}
		log.Printf("Script file written successfully: %s", file.Path)
	} else if err != nil {
		log.Printf("Failed to retrieve script content: %v", err)
		return err
	}

	// Initialize command execution ability
	dev := ability.DevCommandAbility{
		Home: config.GetWorkPath(""),
	}
	log.Printf("Command execution home: %s", dev.Home)

	// Build command string for execution
	cmdStr := cmd + " " + strings.Join(args, " ")
	logFile := fmt.Sprintf("%s.log", file.Path)

	log.Printf("Executing command: %s > %s", cmdStr, logFile)
	if _, err := dev.Start(cmdStr, logFile); err != nil {
		log.Printf("Failed to start install command: %v", err)
		return fmt.Errorf("failed to start install: %v", err)
	}
	return nil
}

func (h *HttpServie) LoadGlobal() map[string]any {
	result := map[string]any{}
	if list, _ := h.store.LoadBot(); len(list) > 0 {
		bots := []map[string]any{}
		for _, r := range list {
			bots = append(bots, r.ToMap())
		}
		result["bots"] = bots
	}

	// default bot config
	if list, err := h.store.LoadCfg(); err == nil {
		for _, item := range list {
			switch item.Type {
			case entity.KEY_LOGIN_USER:
				result["login"] = item.Data
			case entity.KEY_APP_SETUP:
				result["setup"] = item.Data
			case entity.KEY_ACTIVE_BOT:
				result["active"] = item.Data
			case entity.KEY_EPIGRAPH:
				result["epigraph"], _ = item.Data["text"]
			case entity.KEY_USE_MODEL:
				result["useModel"], _ = item.Data["provider"]
			}
		}
	}
	result["mcpEnv"] = h.GetMcpEnv()
	result["authGate"] = config.GetAuthGate()
	return result
}

func (h *HttpServie) DoLaunch(base, path string) error {
	dev := ability.DevCommandAbility{Home: base}

	// Use platform-specific commands to open file/directory
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = "open"
		args = []string{path}
	case "windows":
		cmd = "start"
		args = []string{path}
	case "linux":
		// Try different Linux file managers in order of preference
		if _, err := dev.Exec("which xdg-open", 10*time.Second); err == nil {
			cmd = "xdg-open"
			args = []string{path}
		} else if _, err := dev.Exec("which nautilus", 10*time.Second); err == nil {
			cmd = "nautilus"
			args = []string{path}
		} else if _, err := dev.Exec("which dolphin", 10*time.Second); err == nil {
			cmd = "dolphin"
			args = []string{path}
		} else if _, err := dev.Exec("which thunar", 10*time.Second); err == nil {
			cmd = "thunar"
			args = []string{path}
		} else {
			return fmt.Errorf("no suitable file manager found on Linux system")
		}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Execute the launch command
	if _, err := dev.Run(cmd, 10*time.Second, args...); err != nil {
		return fmt.Errorf("launch failed: %v", err)
	}
	return nil
}

func (h *HttpServie) DoUpload(base string, files []*multipart.FileHeader) error {
	dev := ability.FileSystemAbility{Base: base}
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			continue
		}
		defer file.Close()
		dev.Path = header.Filename
		if err := dev.Copy(file); err != nil {
			return err
		}
	}
	return nil
}

func (h *HttpServie) HasProvider(name string) bool {
	cfg := &entity.CfgEntity{
		Name: name, Type: entity.KEY_PROVIDER,
	}
	if err := h.store.FindCfg(cfg); err == nil {
		return true
	}
	return false
}

func (h *HttpServie) SaveProvider(data *entity.CfgEntity) error {
	if !support.Bool(data.Data) {
		return fmt.Errorf("invalid data")
	}
	// 同步更新 provider 配置
	cfg := &entity.CfgEntity{Data: data.Data}
	cfg.Name, _ = cfg.Data["provider"].(string)
	cfg.Type = entity.KEY_PROVIDER
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) LoadSetupCfg() any {
	cfg := &entity.CfgEntity{
		Name: entity.KEY_APP_SETUP,
		Type: entity.KEY_APP_SETUP,
	}
	workHome := config.GetWorkHome()
	if err := h.store.FindCfg(cfg); err == nil {
		cfg.Data["dataPath"] = workHome
		return cfg.Data
	}
	return map[string]any{"dataPath": workHome}
}

func (h *HttpServie) SaveSetupCfg(cfg *entity.CfgEntity) error {
	if !support.Bool(cfg.Data) {
		return fmt.Errorf("invalid data")
	}
	cfg.Name = entity.KEY_APP_SETUP
	cfg.Type = entity.KEY_APP_SETUP
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) LoadModelCfg() any {
	result := map[string]any{}
	if list, err := h.store.LoadCfg(); err == nil {
		models := map[string]any{}
		result["models"] = &models
		for _, item := range list {
			switch item.Type {
			case entity.KEY_USE_MODEL:
				result["useModel"] = item.Data
			case entity.KEY_PROVIDER:
				models[item.Name] = item.Data
			}
		}
	}
	return result
}

func (h *HttpServie) SaveUseModel(cfg *entity.CfgEntity) error {
	if !support.Bool(cfg.Data) {
		return fmt.Errorf("invalid")
	}
	cfg.Name = entity.KEY_USE_MODEL
	cfg.Type = entity.KEY_USE_MODEL
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) InitBot() []*entity.BotEntity {
	list, _ := h.store.LoadBot()
	if len(list) > 0 {
		return list
	}
	var bots = initial.GetBots()
	if len(bots) == 0 {
		return nil
	}
	service := amcp.GetMcpService(h.store)
	for _, item := range bots {
		bot := &storage.BotEntity{
			UUID: item.UUID, Name: item.Name,
			UsePrompt: item.Desc,
		}
		// 如果有mcps，Tools为[uuid:*]格式
		if len(item.Mcps) > 0 {
			tools := make([]string, 0, len(item.Mcps))
			for uuid := range item.Mcps {
				tools = append(tools, uuid+":*")
			}
			bot.Tools = tools
		}
		if err := h.store.SaveBot(bot); err != nil {
			continue
		}
		go service.LoadMcpServer(item.Mcps)
	}
	list, _ = h.store.LoadBot()
	if len(list) > 0 {
		h.UseBot(list[0])
	}
	return list
}

func (h *HttpServie) LoadBot() []*entity.BotEntity {
	bots, _ := h.store.LoadBot()
	return bots
}

func (h *HttpServie) FindBot(uuid string) *entity.BotEntity {
	bot := &entity.BotEntity{UUID: uuid}
	if err := h.store.FindBot(bot); err == nil {
		return bot
	}
	return nil
}

func (h *HttpServie) SaveBot(bot *entity.BotEntity) error {
	return h.store.SaveBot(bot)
}

func (h *HttpServie) UseBot(bot *entity.BotEntity) error {
	cfg := &entity.CfgEntity{
		Name: entity.KEY_ACTIVE_BOT,
		Type: entity.KEY_ACTIVE_BOT,
		Data: bot.ToMap(),
	}
	config.Set("ACTIVE_BOT", bot.UUID)
	config.Set("CURRENT_HOME", bot.Home)
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) LoadMem() []*entity.MemEntity {
	list, _ := h.store.LoadMem()
	return list
}

func (h *HttpServie) FindMem(id uint) *entity.MemEntity {
	mem := &entity.MemEntity{ID: id}
	if h.store.FindMem(mem) == nil {
		return mem
	}
	return nil
}

func (h *HttpServie) SaveMem(mem *entity.MemEntity) error {
	return h.store.SaveMem(mem)
}

func (h *HttpServie) SaveState(task *entity.TaskEntity, state string) error {
	task.State = state
	return h.store.SaveTask(task)
}

func (h *HttpServie) LoadMsg(task *entity.TaskEntity, msgid string) *storage.MsgEntity {
	msg := &entity.MsgEntity{MsgId: msgid}
	if err := h.store.FindMsg(msg); err == nil {
		return msg
	}
	return nil
}

func (h *HttpServie) LoadCache(key string) map[string]any {
	list, err := h.store.LoadCfg()
	if err != nil || len(list) == 0 {
		return nil
	}
	for _, item := range list {
		if item.Type != entity.KEY_MY_CACHE {
			continue
		}
		if key == item.Name {
			return item.Data
		}
	}
	return nil
}

func (h *HttpServie) SaveCache(key string, data map[string]any) error {
	cfg := &entity.CfgEntity{
		Name: key, Data: data,
		Type: entity.KEY_MY_CACHE,
	}
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) SaveProfile(data map[string]any) error {
	cfg := &entity.CfgEntity{
		Data: data,
		Name: entity.KEY_LOGIN_USER,
		Type: entity.KEY_LOGIN_USER,
	}
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) ClearProfile() error {
	cfg := &entity.CfgEntity{
		Name: entity.KEY_LOGIN_USER,
		Type: entity.KEY_LOGIN_USER,
	}
	cfg.DeletedAt.Time = time.Now()
	err := h.store.SaveCfg(cfg)
	if err != nil {
		return err
	}
	cfg = &entity.CfgEntity{
		Name: entity.KEY_USE_MODEL,
		Type: entity.KEY_USE_MODEL,
	}
	cfg.DeletedAt.Time = time.Now()
	return h.store.SaveCfg(cfg)
}

// VerifyToken 通过 auth.swiflow.cc 验证认证token
func (h *HttpServie) VerifyToken(req any) (map[string]any, error) {
	var result map[string]any
	var url = config.GetAuthGate() + "/v1/verify-token"
	var body = strings.NewReader(support.ToJson(req))
	log.Println("[AUTH] verify-token request:", url, support.ToJson(req))
	if resp, err := http.Post(url, "application/json", body); err != nil {
		return nil, fmt.Errorf("%v", err)
	} else {
		defer resp.Body.Close()
		decode := json.NewDecoder(resp.Body)
		if err := decode.Decode(&result); err != nil {
			return nil, fmt.Errorf("%v", err)
		}
		if msg, _ := result["error"].(string); msg != "" {
			return nil, fmt.Errorf("%v", msg)
		}
	}
	log.Println("[AUTH] verify-token respond:", url, support.ToJson(result))
	return result, nil
}

// GetProfile 调用 auth.swiflow.cc 验证API token
func (h *HttpServie) GetProfile(apiKey string) (map[string]any, error) {
	var url = config.GetAuthGate() + "/v1/user-profile"
	var req, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %v", err)
	}

	var result map[string]any
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	log.Println("[AUTH] user-profile request:", url, apiKey)
	if resp, err := client.Do(req); err != nil {
		return nil, fmt.Errorf("call auth service failed: %v", err)
	} else {
		defer resp.Body.Close()
		decode := json.NewDecoder(resp.Body)
		if err := decode.Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response failed: %v", err)
		}
	}
	log.Println("[AUTH] user-profile respond:", url, support.ToJson(result))
	return result, nil
}
