package httpd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"swiflow/ability"
	"swiflow/action"
	"swiflow/agent"
	"swiflow/builtin"
	"swiflow/config"
	"swiflow/entity"
	"swiflow/initial"
	"swiflow/model"
	"swiflow/support"
	"time"

	"github.com/duke-git/lancet/v2/convertor"
)

type HttpHandler struct {
	service *HttpServie
	manager *agent.Manager
}

func NewHttpHandler(m *agent.Manager) *HttpHandler {
	var s = new(HttpServie)
	store, e := m.GetStorage()
	if store != nil && e == nil {
		s.store = store
	}
	return &HttpHandler{s, m}
}

func JsonResp(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	if err, ok := data.(error); ok && err != nil {
		result := map[string]any{"errmsg": err.Error()}
		return json.NewEncoder(w).Encode(result)
	}

	if str, ok := data.(string); ok && str != "" {
		result := map[string]any{"result": str}
		return json.NewEncoder(w).Encode(result)
	}

	if err := json.NewEncoder(w).Encode(data); err == nil {
		return nil
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"errmsg":"something wrong"}`))
	return nil
}

func (h *HttpHandler) Static(w http.ResponseWriter, r *http.Request) {
	embedFs := initial.GeEmbedFS()
	subfs, err := fs.Sub(embedFs, "html")
	if err != nil {
		if err = JsonResp(w, err); err != nil {
			log.Println("[HTTP] resp error", err)
		}
		return
	}
	http.FileServer(http.FS(subfs)).ServeHTTP(w, r)
}

func (h *HttpHandler) Intent(w http.ResponseWriter, r *http.Request) {
	// Parse request body to get input data
	var request builtin.IntentRequest
	if err := h.service.ReadTo(r.Body, &request); err != nil {
		JsonResp(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	var manager = builtin.GetManager()
	var store, _ = h.manager.GetStorage()
	if len(manager.AllTools()) == 0 {
		tools, _ := store.LoadTool()
		manager.Init(tools)
	}

	// 使用历史消息进行意图识别（按 session 从 cfg 读取）
	var history = h.service.GetHistory(&request)
	var intent, err = manager.GetIntent(&request, history)
	if err != nil || intent == nil {
		JsonResp(w, fmt.Errorf("get intent error: %w", err))
		return
	} else if intent.Intent != "task" {
		JsonResp(w, intent)
		return
	}

	// 完成时写入当前用户输入到历史
	if h.service.AppendHistory(&request, intent) != nil {
		log.Println("[HTTP] append history error", err)
	}

	worker, err := h.manager.GetWorker(intent.Worker)
	if err != nil || worker == nil {
		JsonResp(w, fmt.Errorf("get worker error: %w", err))
		return
	}

	// Handle task creation/retrieval logic
	var task *agent.MyTask
	if intent.TaskID != "" {
		task, err = h.manager.QueryTask(intent.TaskID)
	}
	if task == nil {
		intent.TaskID = fmt.Sprintf("%s-%d", request.Session, time.Now().Unix())
		task, err = h.manager.InitTask(request.Content, intent.TaskID)
	}

	if config.Get("DEBUG_MODE") == "yes" {
		task.IsDebug = true
	}
	task.Home = config.CurrentHome()
	input := &action.UserInput{
		Content: request.Content,
		Uploads: []string{},
	}
	for _, file := range request.Uploads {
		if strings.HasPrefix(file, "[") {
			input.Uploads = append(input.Uploads, file)
		} else {
			filename := fmt.Sprintf("[%s](%s)", file, file)
			input.Uploads = append(input.Uploads, filename)
		}
	}
	go h.manager.Handle(input, task, worker)
	JsonResp(w, intent)
}

// Start chat handle
func (h *HttpHandler) Start(w http.ResponseWriter, r *http.Request) {
	// Parse request body to get input data
	var request struct {
		Content  string   `json:"content"`
		Uploads  []string `json:"uploads"`
		StartNew string   `json:"startNew"`
		TaskUUID string   `json:"taskUUID"`
		WorkerId string   `json:"workerId"`
		HomePath string   `json:"homePath"`
	}
	if err := h.service.ReadTo(r.Body, &request); err != nil {
		JsonResp(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Create UserInput from request data
	input := &action.UserInput{
		Content: strings.TrimSpace(request.Content),
		Uploads: request.Uploads,
	}

	// Get worker UUID from request data or config
	uuid := config.GetStr("USE_WORKER", "")
	if request.WorkerId != "" {
		uuid = request.WorkerId
	}
	worker, err := h.manager.GetWorker(uuid)
	if err != nil || worker == nil {
		JsonResp(w, fmt.Errorf("get worker error: %w", err))
		return
	}

	// Handle task creation/retrieval logic
	var task *agent.MyTask
	if request.StartNew == "yes" {
		task, err = h.manager.InitTask(input.Content, request.TaskUUID)
	} else if strings.HasPrefix(request.TaskUUID, "#debug#") {
		worker = convertor.DeepClone(worker)
		task, err = h.manager.NewMcpTask(request.TaskUUID)
	} else {
		task, err = h.manager.QueryTask(request.TaskUUID)
	}
	if task == nil || err != nil {
		JsonResp(w, fmt.Errorf("query task error: %w", err))
		return
	}

	if worker.Home == "" {
		task.Home = config.CurrentHome()
	}

	// Set home path if provided
	if request.HomePath != "" {
		task.IsDebug = true
		task.Home = request.HomePath
	}

	if config.Get("DEBUG_MODE") == "yes" {
		task.IsDebug = true
	}

	if config.Get("USE_SUBAGENT") != "yes" {
		go h.manager.Handle(input, task, worker)
	} else {
		go h.manager.Start(input, task, worker)
	}

	response := map[string]any{
		"success": true, "taskUUID": task.UUID,
		"message": "Task started successfully",
	}
	JsonResp(w, response)
}

// Global info handle
func (h *HttpHandler) Global(w http.ResponseWriter, r *http.Request) {
	if bots := h.service.LoadBot(); len(bots) == 0 {
		if len(h.service.InitBot()) == 0 {
			JsonResp(w, fmt.Errorf("init bot fail"))
			return
		}
		if err := h.manager.Initial(); err != nil {
			JsonResp(w, fmt.Errorf("init bot: %w", err))
			return
		}
	}

	result := h.service.LoadGlobal()
	if err := JsonResp(w, result); err != nil {
		log.Println("[HTTP] resp error", err)
	}
}

func (h *HttpHandler) Upload(w http.ResponseWriter, r *http.Request) {
	// 限制上传文件大小 (例如10MB)
	r.ParseMultipartForm(32 << 20)
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		JsonResp(w, fmt.Errorf("error upload data"))
		return
	}
	uuid := r.URL.Query().Get("uuid")
	bot, err := h.manager.QueryWorker(uuid)
	if bot == nil || err != nil {
		err = fmt.Errorf("找不到 Bot: %v", err)
		if err = JsonResp(w, err); err != nil {
			log.Println("[HTTP] resp error", err)
		}
		return
	}

	home := config.CurrentHome()
	home = support.Or(bot.Home, home)
	files := r.MultipartForm.File["files"]
	err = h.service.DoUpload(home, files)
	if err != nil {
		JsonResp(w, err)
		return
	}

	var result = map[string]string{}
	for _, hd := range files {
		name := hd.Filename
		result[name] = name
	}
	JsonResp(w, result)
}

// Import handles .agent file imports by extracting zip files and processing agent definitions
func (h *HttpHandler) Import(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form with size limit (32MB)
	r.ParseMultipartForm(32 << 20)

	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		JsonResp(w, fmt.Errorf("no files uploaded"))
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		JsonResp(w, fmt.Errorf("no files found"))
		return
	}

	// Process .agent files through service layer
	allWorkers, err := h.service.DoImport(files)
	if err != nil {
		JsonResp(w, err)
		return
	}

	var leaderId string
	var imported = []string{}
	for _, worker := range allWorkers {
		imported = append(imported, worker.Name)
		if worker.Type == agent.AGENT_LEADER {
			leaderId = worker.UUID
		}
	}

	// Return results in consistent format
	var result = map[string]any{
		"leaderId": leaderId,
		"imported": imported,
	}
	JsonResp(w, result)
}

func (h *HttpHandler) Setting(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	switch act {
	case "get-model":
		result := h.service.LoadModelCfg()
		if err := JsonResp(w, result); err != nil {
			log.Println("resp error", err)
		}
	case "set-model":
		cfg := &entity.CfgEntity{}
		var data = h.service.ReadMap(r.Body)
		cfg.Data, _ = data.(map[string]any)
		if err := h.service.SaveUseModel(cfg); err == nil {
			h.service.SaveProvider(cfg)
			h.manager.InitConfig()
			JsonResp(w, cfg.ToMap())
		} else {
			JsonResp(w, err)
		}
	case "set-provider":
		cfg := &entity.CfgEntity{}
		var data = h.service.ReadMap(r.Body)
		cfg.Data, _ = data.(map[string]any)
		if err := h.service.SaveProvider(cfg); err == nil {
			h.manager.InitConfig()
			JsonResp(w, cfg.ToMap())
		} else {
			JsonResp(w, err)
		}
		return
	case "get-setup":
		result := h.service.LoadSetupCfg()
		if err := JsonResp(w, result); err != nil {
			log.Println("resp error", err)
		}
	case "put-setup":
		cfg := &entity.CfgEntity{}
		var data = h.service.ReadMap(r.Body)
		cfg.Data, _ = data.(map[string]any)
		err := h.service.SaveSetupCfg(cfg)
		if err == nil {
			h.manager.UpdateEnv(cfg)
			JsonResp(w, cfg.ToMap())
		} else {
			JsonResp(w, err)
		}
	}
}

func (h *HttpHandler) SignOut(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ClearProfile(); err != nil {
		JsonResp(w, fmt.Errorf("clear-profile failed: %v", err))
	}
	JsonResp(w, "success")
}

// 调用auth.swiflow.cc验证token
func (h *HttpHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var apiKey string
	var body = h.service.ReadMap(r.Body)
	if info, err := h.service.VerifyToken(body); err != nil {
		JsonResp(w, err)
		return
	} else if key, _ := info["apiKey"].(string); key == "" {
		JsonResp(w, fmt.Errorf("extract apiKey failed"))
		return
	} else {
		apiKey = key
	}

	var llmCfg = &model.LLMConfig{}
	var cfgData = &entity.CfgEntity{}
	var userData = &entity.UserEntity{}
	if info, err := h.service.GetProfile(apiKey); err != nil {
		JsonResp(w, err)
	} else if err := h.service.SaveProfile(info); err != nil {
		JsonResp(w, err)
	} else if err := userData.FromMap(info); err != nil {
		JsonResp(w, err)
	} else if userData.APIKey != "" {
		llmCfg.ApiKey, llmCfg.Provider = userData.APIKey, "swiflow"
		llmCfg.ApiUrl, llmCfg.UseModel = config.GetAuthGate()+"/v1", ""
		cfgData.Data, _ = convertor.StructToMap(llmCfg)
		if err := h.service.SaveProvider(cfgData); err != nil {
			log.Println("[SIGN] save provider: ", err.Error())
		}
		if err := h.service.SaveUseModel(cfgData); err != nil {
			log.Println("[SIGN] save llm cfg: ", err.Error())
		}
		JsonResp(w, "success")
	} else {
		JsonResp(w, "success")
	}
}

func (h *HttpHandler) Program(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	uuid := r.URL.Query().Get("uuid")
	task, err := h.manager.QueryTask(uuid)
	if err != nil || task == nil {
		err = JsonResp(w, err)
	}
	if err = h.service.Run(task, act); err != nil {
		err = JsonResp(w, err)
	} else {
		err = JsonResp(w, task.ToMap())
	}
}

func (h *HttpHandler) ToolEnv(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("act") {
	case "release":
		JsonResp(w, h.service.GetRelease())
	case "mcp-env":
		JsonResp(w, h.service.GetMcpEnv())
	case "net-env":
		JsonResp(w, h.service.GetNetEnv())
	case "install":
		name := r.URL.Query().Get("name")
		netEnv := r.URL.Query().Get("net-env")
		resp := h.service.InitMcpEnv(name, netEnv)
		if err := JsonResp(w, resp); err != nil {
			log.Println("[HTTP] resp error", err)
		}
	default:
		JsonResp(w, "undefined command")
	}
}

func (h *HttpHandler) Launch(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	bot, err := h.manager.QueryWorker(uuid)
	if bot == nil || err != nil {
		err = fmt.Errorf("找不到 Bot: %v", err)
		if err = JsonResp(w, err); err != nil {
			log.Println("[HTTP] resp error", err)
		}
		return
	}

	path := r.URL.Query().Get("path")
	baseHome := config.CurrentHome()
	bot.Home = support.Or(bot.Home, baseHome)
	// Construct the full path to open
	var targetPath string
	if path == "." || path == "" {
		targetPath = bot.Home
	} else {
		targetPath = filepath.Join(bot.Home, path)
	}
	// Verify the path exists
	if _, err = os.Stat(targetPath); err != nil {
		err = fmt.Errorf("path does not exist: %v", path)
		JsonResp(w, map[string]string{"errmsg": err.Error()})
		return
	}
	if err = h.service.DoLaunch(bot.Home, path); err != nil {
		JsonResp(w, map[string]string{"errmsg": err.Error()})
	} else {
		JsonResp(w, map[string]string{"success": "Directory launched successfully"})
	}
}

func (h *HttpHandler) Browser(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	bot, err := h.manager.QueryWorker(uuid)
	if bot == nil || err != nil {
		JsonResp(w, err)
		return
	}
	baseHome := config.CurrentHome()
	bot.Home = support.Or(bot.Home, baseHome)
	path := r.URL.Query().Get("path")
	file := ability.FileSystemAbility{
		Path: path, Base: bot.Home,
	}

	if file.IsDir() {
		// 目录：返回文件列表
		if data, err := file.List(); err != nil {
			JsonResp(w, err)
		} else {
			JsonResp(w, data)
		}
		return
	}

	// 文件：直接返回文件流
	fullPath := filepath.Join(file.Base, file.Path)
	if _, err := os.Stat(fullPath); err != nil {
		JsonResp(w, fmt.Errorf("文件不存在: %v", err))
		return
	}

	// 设置适当的 Content-Type
	ext := strings.ToLower(filepath.Ext(file.Path))
	contentType := "application/octet-stream"
	switch ext {
	case ".txt", ".md", ".py", ".js", ".ts", ".html", ".css",
		".json", ".yaml", ".yml", ".xml", ".csv", ".sql", ".sh",
		".bat", ".ps1", ".dockerfile", ".gitignore", ".env",
		".ini", ".cfg", ".conf", ".log", ".rst", ".toml", ".lock":
		contentType = "text/plain"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		"inline; filename=%s", filepath.Base(file.Path),
	))

	// 直接流式传输文件
	http.ServeFile(w, r, fullPath)
}

func (h *HttpHandler) Execute(w http.ResponseWriter, r *http.Request) {
	act := r.URL.Query().Get("act")
	uuid := r.URL.Query().Get("uuid")
	task, err := h.manager.QueryTask(uuid)
	if task == nil || err != nil {
		JsonResp(w, fmt.Errorf("找不到任务: %v", err))
		return
	}
	bot, err := h.manager.QueryWorker(task.BotId)
	if bot == nil || err != nil {
		JsonResp(w, fmt.Errorf("找不到Bot: %v", err))
		return
	}

	executor := h.manager.LoadExecutor(task, bot)
	if executor == nil && act == "resume" {
		err = fmt.Errorf("no active executor")
		if err = JsonResp(w, err); err != nil {
			log.Println("[HTTP] resp error", err)
		}
		return
	}
	if executor == nil && act != "resume" {
		executor = h.manager.GetExecutor(task, bot)
	}

	switch act {
	case "resume":
		err = executor.Resume()
	case "replay":
		msgid := r.URL.Query().Get("msgid")
		msg := h.service.LoadMsg(task, msgid)
		if msg == nil {
			JsonResp(w, fmt.Errorf("no message select"))
			return
		}
		respAction := action.Parse(msg.Request)
		respAction.Payload = &action.Payload{
			UUID: task.UUID,
			Time: time.Now(),
			Home: task.Home,
		}
		if val := executor.PlayAction(respAction); val != "" {
			if err = JsonResp(w, val); err != nil {
				log.Println("[HTTP] resp error", err)
			}
		}
		return
	case "stop":
		if err = model.Cancel(uuid); err == nil {
			err = executor.Terminate()
			h.service.SaveState(task, "canceled")
		}
	}

	if err = JsonResp(w, err); err != nil {
		log.Println("[HTTP] resp error", err)
	}
}
