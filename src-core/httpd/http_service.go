package httpd

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"swiflow/ability"
	"swiflow/agent"
	"swiflow/amcp"
	"swiflow/builtin"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/initial"
	"swiflow/model"
	"swiflow/storage"
	"swiflow/support"
	"time"

	"gorm.io/gorm"
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
	if data, err := dev.Run(python, 3*time.Second, "-V"); err == nil {
		result["python"] = strings.TrimSpace(string(data))
	}
	// uvx env check
	if data, err := dev.Run("uvx", 3*time.Second, "-V"); err == nil {
		env := strings.TrimSpace(string(data))
		result["uvx"] = strings.Split(env, "(")[0]
	}

	// uv env check
	if data, err := dev.Run("uv", 3*time.Second, "-V"); err == nil {
		env := strings.TrimSpace(string(data))
		result["uv"] = strings.Split(env, "(")[0]
	}

	// node.js env check
	if data, err := dev.Run("node", 3*time.Second, "-v"); err == nil {
		result["nodejs"] = strings.TrimSpace(string(data))
	}

	// npx env check
	if data, err := dev.Run("npx", 3*time.Second, "-v"); err == nil {
		result["npx"] = strings.TrimSpace(string(data))
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
	if list, err := h.store.LoadCfg(); err == nil { // Call without parameters to maintain existing behavior
		for _, item := range list {
			switch item.Type {
			case entity.KEY_LOGIN_USER:
				result["login"] = item.Data
			case entity.KEY_APP_SETUP:
				result["setup"] = item.Data
			case entity.KEY_USE_WORKER:
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
	if list, err := h.store.LoadCfg(); err == nil { // Call without parameters to maintain existing behavior
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
	bots, _ := h.store.LoadBot() // Call without parameters to maintain existing behavior
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
		Name: entity.KEY_USE_WORKER,
		Type: entity.KEY_USE_WORKER,
		Data: bot.ToMap(),
	}
	config.Set("USE_WORKER", bot.UUID)
	return h.store.SaveCfg(cfg)
}

func (h *HttpServie) LoadMem() []*entity.MemEntity {
	list, _ := h.store.LoadMem() // Call without parameters to maintain existing behavior
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
	msg := &entity.MsgEntity{UniqId: msgid}
	if err := h.store.FindMsg(msg); err == nil {
		return msg
	}
	return nil
}

func (h *HttpServie) LoadCache(key string) map[string]any {
	list, err := h.store.LoadCfg() // Call without parameters to maintain existing behavior
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

// InitMcpEnvAsync starts the MCP environment installation asynchronously and checks status periodically
func (h *HttpServie) InitMcpEnvAsync(name string, env string) any {
	log.Printf("InitMcpEnvAsync started: name=%s, env=%s", name, env)

	// Start the installation process first
	if err := h.InitMcpEnv(name, env); err != nil {
		log.Printf("Failed to start MCP environment installation: %v", err)
		return err
	}

	// Start asynchronous checking in a goroutine
	go h.checkMcpEnvPeriodically(name)

	return map[string]any{
		"status":  "started",
		"message": "MCP environment installation started, checking every 5 seconds",
	}
}

// checkMcpEnvPeriodically checks MCP environment status every 5 seconds for up to 10 minutes
func (h *HttpServie) checkMcpEnvPeriodically(name string) {
	const checkInterval = 5 * time.Second
	const maxDuration = 10 * time.Minute

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	timeout := time.After(maxDuration)

	log.Printf("Starting periodic MCP environment check for %s (5s intervals, 10min timeout)", name)

	for {
		select {
		case <-ticker.C:
			// Check environment status
			env := h.GetMcpEnv().(map[string]any)

			// Check if installation is complete based on environment type
			var isComplete bool
			switch name {
			case "uvx-py":
				// For Python environment, check both python and uvx
				python, hasPython := env["python"]
				uvx, hasUvx := env["uvx"]
				isComplete = hasPython && hasUvx && python != "" && uvx != ""
				log.Printf("Python env check: python=%v, uvx=%v, complete=%v", python, uvx, isComplete)
			case "js-npx":
				// For Node.js environment, check both nodejs and npx
				nodejs, hasNodejs := env["nodejs"]
				npx, hasNpx := env["npx"]
				isComplete = hasNodejs && hasNpx && nodejs != "" && npx != ""
				log.Printf("Node.js env check: nodejs=%v, npx=%v, complete=%v", nodejs, npx, isComplete)
			}

			if isComplete {
				log.Printf("MCP environment installation completed successfully for %s", name)
				return
			}

		case <-timeout:
			log.Printf("MCP environment installation timeout reached (10 minutes) for %s", name)
			return
		}
	}
}

// extractZipFile extracts a zip file to the specified destination directory
func (h *HttpServie) extractZipFile(zipPath, destDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		// Create the full file path
		path := filepath.Join(destDir, file.Name)

		// Ensure the file path is within the destination directory (security check)
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// Create directory
			os.MkdirAll(path, file.FileInfo().Mode())
			continue
		}

		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Extract file
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// ImportWorkers saves discovered workers to database and loads MCP servers
func (h *HttpServie) ImportWorkers(workers []*entity.BotEntity) error {
	service := amcp.GetMcpService(h.store)
	for _, worker := range workers {
		// If worker has MCP servers
		// set tools in [uuid:*] format
		if len(worker.McpServers) > 0 {
			tools := make([]string, 0, len(worker.McpServers))
			for uuid := range worker.McpServers {
				tools = append(tools, uuid+":*")
			}
			worker.Tools = tools
		}

		if err := h.store.SaveBot(worker); err != nil {
			log.Printf("[IMPORT] Failed to save bot %s: %v", worker.Name, err)
			return err
		}

		// Load MCP servers if available
		if len(worker.McpServers) > 0 {
			go service.LoadMcpServer(worker.McpServers)
		}
	}

	return nil
}

// DoImport processes uploaded .agent files and returns all imported bot
func (h *HttpServie) DoImport(files []*multipart.FileHeader) ([]*entity.BotEntity, error) {
	// Filter only .agent files
	var agentFiles []*multipart.FileHeader
	for _, header := range files {
		if strings.HasSuffix(header.Filename, ".agent") {
			agentFiles = append(agentFiles, header)
		} else {
			log.Printf("[IMPORT] Skipping non-.agent file: %s", header.Filename)
		}
	}

	if len(agentFiles) == 0 {
		return nil, fmt.Errorf("no valid .agent files found")
	}

	// Create temp directory for uploaded files
	importDir := filepath.Join(os.TempDir(), "swiflow-import")
	if err := os.MkdirAll(importDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp upload dir: %v", err)
	}
	defer os.RemoveAll(importDir)

	// Use DoUpload to handle file uploading
	if err := h.DoUpload(importDir, agentFiles); err != nil {
		return nil, fmt.Errorf("failed to upload files: %v", err)
	}

	var allWorkers []*entity.BotEntity
	for _, header := range agentFiles {
		agentFilePath := filepath.Join(importDir, header.Filename)
		targetDir := config.GetWorkPath("agents", header.Filename)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Printf("[IMPORT] Failed to create extract dir for %s: %v", header.Filename, err)
			continue
		}
		defer os.RemoveAll(targetDir)

		// Extract zip file to target directory
		if err := h.extractZipFile(agentFilePath, targetDir); err != nil {
			log.Printf("[IMPORT] Failed to extract %s: %v", header.Filename, err)
			continue
		}

		// Discover and import agents from extracted directory
		workers, err := agent.DiscoverWorkers(targetDir)
		if err != nil {
			log.Printf("[IMPORT] Failed to discover workers in %s: %v", targetDir, err)
			continue
		}

		// Save discovered workers to database
		if err := h.ImportWorkers(workers); err != nil {
			log.Printf("[IMPORT] Failed to save workers: %v", err)
			continue
		}

		allWorkers = append(allWorkers, workers...)
	}

	if len(allWorkers) == 0 {
		return nil, fmt.Errorf("no valid .agent files were imported")
	}
	return allWorkers, nil
}

func (h *HttpServie) GetContext(session string) string {
	var builder strings.Builder

	workers, err := h.store.LoadBot()
	if err == nil && len(workers) > 0 {
		builder.WriteString("\n\n---------------\n\n")
		builder.WriteString("# Available Workers\n\n")
		for _, worker := range workers {
			if worker.Leader == "" {
				continue
			}
			if worker.Type == "leader" {
				continue
			}
			builder.WriteString(fmt.Sprintf(
				"- **%s(id:%s)**\n",
				worker.Name, worker.UUID,
			))
			builder.WriteString(fmt.Sprintf(
				"Desc:   \n%s\n", worker.Desc,
			))
		}
	}

	tasks, err := h.store.LoadTask("session", session)
	if err == nil && len(tasks) > 0 {
		builder.WriteString("\n\n---------------\n\n")
		builder.WriteString("# Recent Tasks     \n\n")
		for _, task := range tasks {
			builder.WriteString(fmt.Sprintf(
				"## - **%s(id:%s)**\n",
				task.Name, task.UUID,
			))
			builder.WriteString(fmt.Sprintf(
				"**%s**:\n```md\n%s\n```\n\n",
				"Context", task.Context,
			))
		}
	}

	return builder.String()
}

func (h *HttpServie) GetHistory(req *builtin.IntentRequest) []model.Message {
	result := []model.Message{
		{Role: "user", Content: h.GetContext(req.Session)},
		{Role: "assistant", Content: support.TrimIndent(`
			<intent>
				<type>talk</type>
				<emoji>Typing</emoji>
				<message>很开心能给你带来帮助，有什么需要我做的吗？</message>
			</intent>`,
		)},
	}
	cfg := &entity.CfgEntity{
		Type: entity.KEY_INTENT_MSG,
		Name: req.Session,
	}
	if err := h.store.FindCfg(cfg); err != nil {
		return result
	}
	if len(cfg.Data) == 0 || cfg.Data == nil {
		cfg.Data = map[string]any{}
	}
	if arr, ok := cfg.Data["msgs"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				role, _ := m["role"].(string)
				utime, _ := m["time"].(float64)
				content, _ := m["content"].(string)
				if strings.TrimSpace(content) != "" {
					continue
				}
				// 过滤30分钟之前的数据
				if time.Now().Unix()-int64(utime) > 30*60 {
					continue
				}
				result = append(result, model.Message{
					Role: role, Content: content,
				})
			}
		}
	}
	return result
}

func (h *HttpServie) AppendHistory(req *builtin.IntentRequest, res *builtin.IntentResult) error {
	cfg := &entity.CfgEntity{
		Type: entity.KEY_INTENT_MSG,
		Name: req.Session, Data: map[string]any{},
	}
	if err := h.store.FindCfg(cfg); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	if len(cfg.Data) == 0 || cfg.Data == nil {
		cfg.Data = map[string]any{}
	}
	msgs := []map[string]any{}
	if arr, ok := cfg.Data["msgs"].([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				utime, _ := m["time"].(float64)
				if time.Now().Unix()-int64(utime) > 30*60 {
					continue
				}
				msgs = append(msgs, m)
			}
		}
	}
	utime := time.Now().Unix()
	msgs = append(msgs, map[string]any{
		"role": "user", "time": utime,
		"content": req.Content,
	})
	msgs = append(msgs, map[string]any{
		"role": "assistant", "time": utime,
		"content": support.ToXML(res, nil),
	})
	cfg.Data["msgs"] = msgs
	return h.store.SaveCfg(cfg)
}
