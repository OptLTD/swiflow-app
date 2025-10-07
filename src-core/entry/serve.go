package entry

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"slices"
	"swiflow/agent"
	"swiflow/httpd"

	"github.com/gorilla/websocket"
)

var server *http.Server
var manager *agent.Manager
var allows = []string{
	"tauri://localhost",
	"http://localhost:5173",
}

func StopWebServer(ctx context.Context) error {
	manager.ClearProcess()
	return server.Shutdown(ctx)
}

func StartWebServer(address string) (err error) {
	mux := http.NewServeMux()
	manager = agent.NewManager()
	handler := httpd.NewHttpHandler(manager)
	setting := httpd.NewSettingHandle(manager)
	mux.HandleFunc("/", handler.Static)
	mux.HandleFunc("/socket", startSocket)
	mux.HandleFunc("/api/bot", setting.BotSet)
	mux.HandleFunc("/api/mem", setting.MemSet)
	mux.HandleFunc("/api/mcp", setting.McpSet)
	mux.HandleFunc("/api/task", setting.TaskSet)
	mux.HandleFunc("/api/todo", setting.TodoSet)
	mux.HandleFunc("/api/tool", setting.ToolSet)
	mux.HandleFunc("/api/msgs", setting.GetMsgs)
	mux.HandleFunc("/api/tasks", setting.GetTasks)

	mux.HandleFunc("/api/start", handler.Start)
	mux.HandleFunc("/api/intent", handler.Intent)
	mux.HandleFunc("/api/global", handler.Global)
	mux.HandleFunc("/api/upload", handler.Upload)
	mux.HandleFunc("/api/import", handler.Import)
	mux.HandleFunc("/api/launch", handler.Launch)
	mux.HandleFunc("/api/browser", handler.Browser)
	mux.HandleFunc("/api/execute", handler.Execute)
	mux.HandleFunc("/api/toolenv", handler.ToolEnv)
	mux.HandleFunc("/api/program", handler.Program)
	mux.HandleFunc("/api/setting", handler.Setting)
	mux.HandleFunc("/api/sign-in", handler.SignIn)
	mux.HandleFunc("/api/sign-out", handler.SignOut)

	// 启动服务（在协程中运行）
	server = &http.Server{Addr: address, Handler: middle(mux)}
	log.Printf("[HTTP] Server is running on http://%s", address)
	if err = server.ListenAndServe(); err != nil {
		log.Fatalf("[HTTP] Server run error: %v", err)
	}
	return
}

func middle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err == nil && httpd.IsInternal(u.Hostname()) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if httpd.IsInternal(r.Host) || slices.Contains(allows, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var upgrader = &websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		if httpd.IsInternal(r.Host) {
			return true
		}
		u, err := url.Parse(r.Header.Get("Origin"))
		if err == nil && httpd.IsInternal(u.Hostname()) {
			return true
		}
		return false
	},
}

// websocket upgrade handle
func startSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("[WS] Error during connection upgradation:", err)
		return
	}

	sessid := r.URL.Query().Get("sessid")
	source := r.URL.Query().Get("source")
	session := httpd.NewWSSession(
		conn, manager, sessid, source,
	)
	defer session.Close()
	session.Handle()
}
