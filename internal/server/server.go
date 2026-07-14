// Package server implements the HTTP REST + SSE layer. Spec §6.8, §10.
package server

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/OptLTD/swiflow/embed"
	"github.com/OptLTD/swiflow/internal/agent"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/secure"
	"github.com/OptLTD/swiflow/internal/sesshub"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tool"
	"github.com/OptLTD/swiflow/internal/util"
	"github.com/OptLTD/swiflow/internal/window"
)

// Server is the HTTP API server.
type Server struct {
	cfg    config.Config
	st     store.Store
	runner *agent.Runner
	tools  *tool.Registry
	skills *skill.Catalog
	mcp    *mcpclient.Manager
	cron   *schedule.Scheduler
	events *sesshub.Hub
	window *window.Bridge
}

// New constructs a server.
func New(cfg config.Config, st store.Store, runner *agent.Runner, tools *tool.Registry, skills *skill.Catalog, mcp *mcpclient.Manager, cron *schedule.Scheduler, events *sesshub.Hub, win *window.Bridge) *Server {
	return &Server{cfg: cfg, st: st, runner: runner, tools: tools, skills: skills, mcp: mcp, cron: cron, events: events, window: win}
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("GET /api/providers", s.listProviders)
	mux.HandleFunc("POST /api/providers", s.createProvider)
	mux.HandleFunc("GET /api/providers/{id}", s.getProvider)
	mux.HandleFunc("PUT /api/providers/{id}", s.updateProvider)
	mux.HandleFunc("DELETE /api/providers/{id}", s.deleteProvider)

	mux.HandleFunc("GET /api/agents", s.listAgents)
	mux.HandleFunc("POST /api/agents", s.createAgent)
	mux.HandleFunc("GET /api/agents/{key}", s.getAgent)
	mux.HandleFunc("PUT /api/agents/{key}", s.updateAgent)

	mux.HandleFunc("GET /api/sessions", s.listSessions)
	mux.HandleFunc("GET /api/sessions/{key}", s.getSession)
	mux.HandleFunc("GET /api/sessions/{key}/watch", s.watchSession)
	mux.HandleFunc("POST /api/sessions/{key}/chat", s.chat)
	mux.HandleFunc("POST /api/sessions/{key}/abort", s.abort)

	mux.HandleFunc("GET /api/tools", s.listTools)
	mux.HandleFunc("PUT /api/tools/{name}", s.setToolEnabled)

	mux.HandleFunc("GET /api/skills", s.listSkills)
	mux.HandleFunc("PUT /api/skills/{slug}", s.setSkillEnabled)
	mux.HandleFunc("POST /api/skills/reload", s.reloadSkills)

	mux.HandleFunc("POST /api/mcp/reload", s.reloadMCP)
	mux.HandleFunc("GET /api/mcp/servers", s.listMCPServers)
	mux.HandleFunc("POST /api/mcp/servers", s.createMCPServer)
	mux.HandleFunc("GET /api/mcp/servers/{id}", s.getMCPServer)
	mux.HandleFunc("GET /api/mcp/servers/{id}/capabilities", s.getMCPServerCapabilities)
	mux.HandleFunc("PUT /api/mcp/servers/{id}", s.updateMCPServer)
	mux.HandleFunc("DELETE /api/mcp/servers/{id}", s.deleteMCPServer)

	mux.HandleFunc("GET /api/cron/jobs", s.listCronJobs)
	mux.HandleFunc("POST /api/cron/jobs", s.createCronJob)
	mux.HandleFunc("POST /api/cron/reload", s.reloadCron)
	mux.HandleFunc("PUT /api/cron/jobs/{id}", s.updateCronJob)
	mux.HandleFunc("DELETE /api/cron/jobs/{id}", s.deleteCronJob)

	mux.HandleFunc("GET /api/workspace/list", s.listWorkspace)
	mux.HandleFunc("GET /api/workspace/read", s.readWorkspaceFile)
	mux.HandleFunc("GET /api/workspace/download", s.downloadFile)
	mux.HandleFunc("POST /api/workspace/download", s.downloadFile)
	mux.HandleFunc("POST /api/workspace/upload", s.uploadWorkspace)

	mux.HandleFunc("POST /api/window/reply", s.windowReply)

	var h http.Handler = mux
	h = s.requestLogMiddleware(h)
	h = s.authMiddleware(h)
	h = s.corsMiddleware(s.cfg.AllowedOrigins)(h)
	h = s.staticMiddleware(h)
	return h
}

func (s *Server) requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = util.NewID()
		}
		w.Header().Set("X-Request-Id", rid)
		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		if !strings.HasPrefix(r.URL.Path, "/api/sessions/") || !strings.HasSuffix(r.URL.Path, "/chat") {
			slog.Info("http",
				"request_id", rid,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher so SSE handlers work through the logging wrapper.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the underlying ResponseWriter for http.ResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// --- middleware ---

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && !s.cfg.SkipAuth {
			tok := r.Header.Get("Authorization")
			tok = strings.TrimPrefix(tok, "Bearer ")
			if tok == "" || tok != s.cfg.AuthToken {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(allowed []string) func(http.Handler) http.Handler {
	allowAll := len(allowed) == 0
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || set[origin]) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// staticMiddleware serves the Vue UI for non-/api routes.
func (s *Server) staticMiddleware(next http.Handler) http.Handler {
	dist, _ := embed.GetFrontendDist()
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(dist, p); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func bindJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return false
	}
	return true
}

// --- health ---

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"skip_auth": s.cfg.SkipAuth,
	})
}

// --- providers ---

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListProviders(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		APIBase     string `json:"api_base"`
		APIKey      string `json:"api_key"`
		Enabled     *bool  `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.APIKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, api_key required"})
		return
	}
	if in.APIBase == "" {
		in.APIBase = "https://api.openai.com/v1"
	}
	if err := secure.ValidateHTTPURL(in.APIBase); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	p := &store.Provider{
		ID:          util.NewID(),
		Name:        in.Name,
		DisplayName: in.DisplayName,
		APIBase:     in.APIBase,
		APIKey:      in.APIKey,
		Enabled:     enabled,
	}
	if err := s.st.CreateProvider(r.Context(), p); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "create failed"})
		return
	}
	s.runner.InvalidateAll()
	p.APIKey = ""
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.st.GetProviderByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in map[string]any
	if !bindJSON(w, r, &in) {
		return
	}
	allowed := map[string]bool{"display_name": true, "api_base": true, "api_key": true, "enabled": true}
	fields := map[string]any{}
	for k, v := range in {
		if allowed[k] {
			fields[k] = v
		}
	}
	if err := s.st.UpdateProvider(r.Context(), id, fields); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.st.DeleteProvider(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- agents ---

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListAgents(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": list})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key         string `json:"key"`
		DisplayName string `json:"display_name"`
		Provider    string `json:"provider"`
		Model       string `json:"model"`
		SystemExtra string `json:"system_extra"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Key == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
		return
	}
	if in.Provider == "" {
		in.Provider = "openai"
	}
	if in.Model == "" {
		in.Model = "gpt-4o-mini"
	}
	if _, err := s.st.GetProviderByName(r.Context(), in.Provider); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider"})
		return
	}
	a := &store.Agent{
		ID:          util.NewID(),
		Key:         in.Key,
		DisplayName: in.DisplayName,
		Provider:    in.Provider,
		Model:       in.Model,
		SystemExtra: in.SystemExtra,
	}
	if err := s.st.CreateAgent(r.Context(), a); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "create failed"})
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	a, err := s.st.GetAgentByKey(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	a, err := s.st.GetAgentByKey(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}
	var in struct {
		DisplayName *string `json:"display_name"`
		Provider    *string `json:"provider"`
		Model       *string `json:"model"`
		SystemExtra *string `json:"system_extra"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	fields := map[string]any{}
	if in.DisplayName != nil {
		fields["display_name"] = *in.DisplayName
	}
	if in.Provider != nil {
		if _, err := s.st.GetProviderByName(r.Context(), *in.Provider); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown provider"})
			return
		}
		fields["provider"] = *in.Provider
	}
	if in.Model != nil {
		fields["model"] = *in.Model
	}
	if in.SystemExtra != nil {
		fields["system_extra"] = *in.SystemExtra
	}
	if len(fields) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no fields to update"})
		return
	}
	if err := s.st.UpdateAgent(r.Context(), a.ID, fields); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	s.runner.InvalidateAll()
	updated, _ := s.st.GetAgentByKey(r.Context(), key)
	writeJSON(w, http.StatusOK, updated)
}

// --- sessions ---

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListSessions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	sess, err := s.st.GetSessionByKey(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	msgs, err := s.st.ListMessages(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load messages failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": sess, "messages": msgs})
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var in struct {
		Message  string `json:"message"`
		AgentKey string `json:"agent_key"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message required"})
		return
	}
	if s.runner.IsBusy(key) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "session busy"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	emit := func(ev agent.Event) {
		data, _ := json.Marshal(ev)
		_, werr := w.Write([]byte("data: " + string(data) + "\n\n"))
		if werr == nil {
			flusher.Flush()
		}
	}
	if s.window != nil {
		s.window.BindEmit(key, func(we window.Event) {
			emit(agent.Event{
				Type:      we.Type,
				ID:        we.ID,
				Name:      we.Name,
				Arguments: we.Arguments,
			})
		})
		defer s.window.UnbindEmit(key)
	}
	err := s.runner.Run(ctx, key, in.AgentKey, in.Message, emit)
	if err != nil && err != agent.ErrBusy {
		slog.Error("chat.run", "session", key, "error", err)
		emit(agent.Event{Type: "error", Error: err.Error()})
	}
}

func (s *Server) windowReply(w http.ResponseWriter, r *http.Request) {
	if s.window == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "window bridge unavailable"})
		return
	}
	var in struct {
		ID     string `json:"id"`
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.ID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	if err := s.window.Reply(in.ID, in.Result, in.Error); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) watchSession(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if s.events == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session watch unavailable"})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, cancel := s.events.Subscribe(key)
	defer cancel()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case data, open := <-ch:
			if !open {
				return
			}
			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) abort(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	ok := s.runner.Abort(key)
	writeJSON(w, http.StatusOK, map[string]bool{"aborted": ok})
}

// --- tools ---

func (s *Server) listTools(w http.ResponseWriter, _ *http.Request) {
	infos := s.tools.All()
	out := make([]tool.Info, 0, len(infos))
	for _, t := range infos {
		if strings.HasPrefix(t.Name, "mcp_") {
			continue
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tools":           out,
		"exec_enabled":    s.cfg.Tools.ExecEnabled,
		"browser_enabled": s.cfg.Tools.BrowserEnabled,
	})
}

func (s *Server) setToolEnabled(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Enabled && tool.IsRuntimeTool(name) && !s.cfg.Tools.ExecEnabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "runtime tools require tools.exec_enabled or SWIFLOW_EXEC=true in config",
		})
		return
	}
	if in.Enabled && tool.IsBrowserTool(name) && !s.cfg.Tools.BrowserEnabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "browser tool requires tools.browser_enabled or SWIFLOW_BROWSER=true in config",
		})
		return
	}
	if err := s.st.SetToolEnabled(r.Context(), name, in.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	s.tools.SetEnabled(name, in.Enabled)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// --- skills ---

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	all := s.skills.Discover(r.Context())
	disabled, _ := s.st.DisabledSkills(r.Context())
	dset := map[string]bool{}
	for _, d := range disabled {
		dset[d] = true
	}
	out := make([]map[string]any, 0, len(all))
	for _, sk := range all {
		out = append(out, map[string]any{
			"slug":        sk.Slug,
			"name":        sk.Name,
			"description": sk.Description,
			"source":      sk.Source,
			"enabled":     !dset[sk.Slug],
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"skills": out})
}

func (s *Server) setSkillEnabled(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if err := s.st.SetSkillEnabled(r.Context(), slug, in.Enabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) reloadSkills(w http.ResponseWriter, _ *http.Request) {
	// Discovery is on-demand; nothing to cache-clear here besides the disabled
	// set, which is read fresh. Acknowledge.
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}
