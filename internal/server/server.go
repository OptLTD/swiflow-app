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
	"github.com/OptLTD/swiflow/internal/auth"
	"github.com/OptLTD/swiflow/internal/config"
	"github.com/OptLTD/swiflow/internal/harness"
	"github.com/OptLTD/swiflow/internal/lightapp"
	"github.com/OptLTD/swiflow/internal/mcpclient"
	"github.com/OptLTD/swiflow/internal/schedule"
	"github.com/OptLTD/swiflow/internal/skill"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/internal/tool"
	appversion "github.com/OptLTD/swiflow/internal/version"
	"github.com/OptLTD/swiflow/library/support"
	"github.com/OptLTD/swiflow/library/window"
)

// Server is the HTTP API server.
type Server struct {
	cfg      config.Config
	st       store.Store
	runner   *agent.Runner
	tools    *tool.Registry
	skills   *skill.Catalog
	mcp      *mcpclient.Manager
	cron     *schedule.Scheduler
	events   *SessionHub
	harness  *harness.Tracker
	window   *window.Bridge
	webOpts  *tool.WebOptions
	lightMgr *lightapp.Manager
	sessions *auth.Store
}

// New constructs a server. webOpts may be nil; search settings API needs a shared pointer to update live.
// tracker may be nil (runs API returns empty).
func New(cfg config.Config, st store.Store, runner *agent.Runner, tools *tool.Registry, skills *skill.Catalog, mcp *mcpclient.Manager, cron *schedule.Scheduler, events *SessionHub, win *window.Bridge, webOpts *tool.WebOptions, tracker *harness.Tracker, lightMgr *lightapp.Manager) *Server {
	if webOpts == nil {
		webOpts = &tool.WebOptions{}
	}
	s := &Server{cfg: cfg, st: st, runner: runner, tools: tools, skills: skills, mcp: mcp, cron: cron, events: events, harness: tracker, window: win, webOpts: webOpts, lightMgr: lightMgr, sessions: auth.NewStore()}
	if win != nil && events != nil {
		win.SetFallback(func(sessionID string, ev window.Event) {
			events.Publish(sessionID, agent.Event{
				Type: ev.Type, ID: ev.ID, Name: ev.Name, Arguments: ev.Arguments,
			})
		})
	}
	return s
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/auth/mode", s.authMode)
	mux.HandleFunc("POST /api/auth/login", s.authLogin)
	mux.HandleFunc("POST /api/auth/register", s.authRegister)
	mux.HandleFunc("POST /api/auth/logout", s.authLogout)
	mux.HandleFunc("GET /api/auth/me", s.authMe)
	mux.HandleFunc("GET /api/runtime", s.getRuntime)
	mux.HandleFunc("POST /api/runtime/install", s.installRuntime)

	mux.HandleFunc("GET /api/providers", s.listProviders)
	mux.HandleFunc("POST /api/providers", s.createProvider)
	mux.HandleFunc("POST /api/providers/act", s.providersAct)

	mux.HandleFunc("GET /api/agents", s.listAgents)
	mux.HandleFunc("POST /api/agents", s.createAgent)
	mux.HandleFunc("POST /api/agents/act", s.agentsAct)

	mux.HandleFunc("GET /api/sessions", s.listSessions)
	mux.HandleFunc("POST /api/sessions/act", s.sessionsAct)

	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("POST /api/runs/act", s.runsAct)

	mux.HandleFunc("GET /api/tools", s.listTools)
	mux.HandleFunc("POST /api/tools/act", s.toolsAct)

	mux.HandleFunc("POST /api/settings/act", s.settingsAct)

	mux.HandleFunc("GET /api/skills", s.listSkills)
	mux.HandleFunc("GET /api/skills/drafts", s.listSkillDrafts)
	mux.HandleFunc("POST /api/skills/act", s.skillsAct)

	mux.HandleFunc("GET /api/mcp/servers", s.listMCPServers)
	mux.HandleFunc("POST /api/mcp/servers", s.createMCPServer)
	mux.HandleFunc("POST /api/mcp/act", s.mcpAct)

	mux.HandleFunc("GET /api/cron/jobs", s.listCronJobs)
	mux.HandleFunc("POST /api/cron/jobs", s.createCronJob)
	mux.HandleFunc("POST /api/cron/act", s.cronAct)

	mux.HandleFunc("GET /api/workspace/list", s.listWorkspace)
	mux.HandleFunc("GET /api/workspace/read", s.readWorkspaceFile)
	mux.HandleFunc("POST /api/workspace/act", s.workspaceAct)

	mux.HandleFunc("GET /api/paths", s.getPaths)
	mux.HandleFunc("POST /api/system/act", s.systemAct)

	mux.HandleFunc("POST /api/window/reply", s.windowReply)

	mux.HandleFunc("GET /api/light-apps", s.listLightApps)
	mux.HandleFunc("POST /api/light-apps", s.createLightApp)
	mux.HandleFunc("POST /api/light-apps/act", s.lightAppsAct)

	var h http.Handler = mux
	h = s.authMiddleware(h)
	h = s.requestLogMiddleware(h)
	h = s.corsMiddleware(s.cfg.AllowedOrigins)(h)
	h = s.staticMiddleware(h)
	return h
}

func (s *Server) requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = support.NewID()
		}
		w.Header().Set("X-Request-Id", rid)
		rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		if r.URL.Path != "/api/sessions/act" {
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

func (s *Server) corsMiddleware(allowed []string) func(http.Handler) http.Handler {
	// LocalMode may reflect any origin (desktop/dev). Serve requires explicit allow-list;
	// empty list means no cross-origin browser access (same-origin UI still works).
	allowAll := s.cfg.LocalMode && len(allowed) == 0
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAll || set[origin] || set["*"]) {
				if set["*"] && !allowAll {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
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
		writeErr(w, http.StatusBadRequest, ErrInvalidJSON)
		return false
	}
	return true
}

// --- health ---

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"version": appversion.Version,
	})
}

// --- providers ---

func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListProviders(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

func (s *Server) createProvider(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		Display string `json:"display"`
		ApiBase string `json:"api_base"`
		ApiKey  string `json:"api_key"`
		Model   string `json:"model"`
		Enabled *bool  `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.ApiKey == "" {
		writeErr(w, http.StatusBadRequest, ErrNameAPIKeyRequired)
		return
	}
	if in.ApiBase == "" {
		in.ApiBase = "https://api.openai.com/v1"
	}
	if in.Model == "" {
		in.Model = "gpt-4o-mini"
	}
	if err := support.ValidateHTTPURL(in.ApiBase); err != nil {
		writeErr(w, http.StatusBadRequest, ErrURLMustBeHTTP, err.Error())
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	p := &store.Provider{
		ID:      support.NewID(),
		Name:    in.Name,
		Display: in.Display,
		ApiBase: in.ApiBase,
		ApiKey:  in.ApiKey,
		Model:   in.Model,
		Enabled: enabled,
	}
	if err := s.st.CreateProvider(r.Context(), p); err != nil {
		writeErr(w, http.StatusConflict, ErrCreateFailed)
		return
	}
	s.runner.InvalidateAll()
	p.ApiKey = ""
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) getProvider(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	p, err := s.st.GetProviderByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	var in map[string]any
	if !bindJSON(w, r, &in) {
		return
	}
	allowed := map[string]bool{"display": true, "api_base": true, "api_key": true, "model": true, "enabled": true}
	fields := map[string]any{}
	for k, v := range in {
		if allowed[k] {
			fields[k] = v
		}
	}
	if err := s.st.UpdateProvider(r.Context(), id, fields); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if err := s.st.DeleteProvider(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrDeleteFailed)
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- agents ---

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListAgents(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": list})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Key       string `json:"key"`
		Display   string `json:"display"`
		TxtModel  string `json:"txt_model"`
		ImgModel  string `json:"img_model"`
		Prompt    string `json:"prompt"`
		Charter   string `json:"charter"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Key == "" {
		writeErr(w, http.StatusBadRequest, ErrKeyRequired)
		return
	}
	if in.TxtModel == "" {
		in.TxtModel = "default"
	}
	if _, err := s.st.GetProviderByName(r.Context(), in.TxtModel); err != nil {
		writeErr(w, http.StatusBadRequest, ErrUnknownTxtModelProvider)
		return
	}
	if in.ImgModel != "" {
		if _, err := s.st.GetProviderByName(r.Context(), in.ImgModel); err != nil {
			writeErr(w, http.StatusBadRequest, ErrUnknownImgModelProvider)
			return
		}
	}
	a := &store.Agent{
		ID:       support.NewID(),
		Key:      in.Key,
		Display:  in.Display,
		TxtModel: in.TxtModel,
		ImgModel: in.ImgModel,
		Prompt:   in.Prompt,
		Charter:  in.Charter,
	}
	if err := s.st.CreateAgent(r.Context(), a); err != nil {
		writeErr(w, http.StatusConflict, ErrCreateFailed)
		return
	}
	s.runner.InvalidateAll()
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	key := requestID(r)
	a, err := s.st.GetAgentByKey(r.Context(), key)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrAgentNotFound)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	key := requestID(r)
	a, err := s.st.GetAgentByKey(r.Context(), key)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrAgentNotFound)
		return
	}
	var in struct {
		Display   *string `json:"display"`
		TxtModel  *string `json:"txt_model"`
		ImgModel  *string `json:"img_model"`
		Prompt    *string `json:"prompt"`
		Charter   *string `json:"charter"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	fields := map[string]any{}
	if in.Display != nil {
		fields["display"] = *in.Display
	}
	if in.TxtModel != nil {
		if _, err := s.st.GetProviderByName(r.Context(), *in.TxtModel); err != nil {
			writeErr(w, http.StatusBadRequest, ErrUnknownTxtModelProvider)
			return
		}
		fields["txt_model"] = *in.TxtModel
	}
	if in.ImgModel != nil {
		if *in.ImgModel != "" {
			if _, err := s.st.GetProviderByName(r.Context(), *in.ImgModel); err != nil {
				writeErr(w, http.StatusBadRequest, ErrUnknownImgModelProvider)
				return
			}
		}
		fields["img_model"] = *in.ImgModel
	}
	if in.Prompt != nil {
		fields["prompt"] = *in.Prompt
	}
	if in.Charter != nil {
		fields["charter"] = *in.Charter
	}
	if len(fields) == 0 {
		writeErr(w, http.StatusBadRequest, ErrNoFieldsToUpdate)
		return
	}
	if err := s.st.UpdateAgent(r.Context(), a.ID, fields); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
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
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
		return
	}
	// Hide subagent (child) sessions: only top-level sessions (no parent) appear
	// in the list. Child sessions are still reachable individually via getSession.
	roots := make([]store.Session, 0, len(list))
	for _, sess := range list {
		if sess.Parent == "" {
			roots = append(roots, sess)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": roots})
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	sess, err := s.st.GetSessionByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	if !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	msgs, err := s.st.ListMessages(r.Context(), sess.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrLoadMessagesFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": sess, "messages": msgs})
}

func (s *Server) deleteSession(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if id == "" {
		writeErr(w, http.StatusBadRequest, ErrIDRequired)
		return
	}
	sess, err := s.st.GetSessionByID(r.Context(), id)
	if err != nil || !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	_ = s.runner.Abort(id)
	if err := s.st.DeleteSession(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrDeleteFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func sessionOwnedByTenant(sess *store.Session, r *http.Request) bool {
	if sess == nil {
		return false
	}
	tid := tenant.ID(r.Context())
	sessTid := sess.Tid
	if sessTid == "" {
		sessTid = tenant.DefaultID
	}
	return sessTid == tid
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	var in struct {
		Message string `json:"message"`
		Agent   string `json:"agent"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Message == "" {
		writeErr(w, http.StatusBadRequest, ErrMessageRequired)
		return
	}
	// Ownership: existing sessions must belong to this tenant; new ids are created on Run.
	if sess, err := s.st.GetSessionByID(r.Context(), id); err == nil && !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}

	// Mid-run: enqueue instead of 409 when session is busy.
	if s.runner.IsBusy(id) {
		pos := s.runner.Enqueue(id, in.Agent, in.Message)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"queued":   true,
			"position": pos,
		})
		return
	}
	if s.runner.AtCapacity(tenant.ID(r.Context())) {
		writeErr(w, http.StatusConflict, ErrTooManyConcurrentRuns)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, ErrStreamingUnsupported)
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
		s.window.BindEmit(id, func(we window.Event) {
			ae := agent.Event{
				Type: we.Type, ID: we.ID, Name: we.Name, Arguments: we.Arguments,
			}
			emit(ae)
			if s.events != nil {
				s.events.Publish(id, ae)
			}
		})
		defer s.window.UnbindEmit(id)
	}
	err := s.runner.Run(ctx, id, in.Agent, in.Message, emit)
	if err == agent.ErrConcurrent {
		emit(agent.Event{Type: "error", Error: err.Error()})
		return
	}
	if err != nil && err != agent.ErrBusy {
		slog.Error("chat.run", "session", id, "error", err)
		emit(agent.Event{Type: "error", Error: err.Error()})
	}
}

func (s *Server) windowReply(w http.ResponseWriter, r *http.Request) {
	if s.window == nil {
		writeErr(w, http.StatusServiceUnavailable, ErrWindowBridgeUnavailable)
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
		writeErr(w, http.StatusBadRequest, ErrIDRequired)
		return
	}
	sessionID, ok := s.window.PendingSession(in.ID)
	if !ok {
		writeErr(w, http.StatusBadRequest, ErrInternalError, "unknown request id")
		return
	}
	sess, err := s.st.GetSessionByID(r.Context(), sessionID)
	if err != nil || !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	if err := s.window.Reply(in.ID, in.Result, in.Error); err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) watchSession(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if sess, err := s.st.GetSessionByID(r.Context(), id); err == nil && !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	if s.events == nil {
		writeErr(w, http.StatusServiceUnavailable, ErrSessionWatchUnavailable)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, ErrStreamingUnsupported)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch, cancel := s.events.Subscribe(id)
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
	id := requestID(r)
	if sess, err := s.st.GetSessionByID(r.Context(), id); err == nil && !sessionOwnedByTenant(sess, r) {
		writeErr(w, http.StatusNotFound, ErrSessionNotFound)
		return
	}
	ok := s.runner.Abort(id)
	writeJSON(w, http.StatusOK, map[string]bool{"aborted": ok})
}

// --- tools ---

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	infos := s.tools.All()
	pol, _ := s.st.ListToolPolicy(r.Context())
	enabled := map[string]bool{}
	for _, p := range pol {
		enabled[p.ToolName] = p.Enabled
	}
	out := make([]tool.Info, 0, len(infos))
	for _, t := range infos {
		if strings.HasPrefix(t.Name, "mcp_") {
			continue
		}
		if e, ok := enabled[t.Name]; ok {
			t.Enabled = e
		}
		out = append(out, t)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tools":            out,
		"exec_enabled":     s.cfg.Tools.ExecEnabled,
		"browser_enabled":  s.cfg.Tools.BrowserEnabled,
		"document_enabled": s.cfg.Tools.DocumentEnabled,
	})
}

func (s *Server) setToolEnabled(w http.ResponseWriter, r *http.Request) {
	name := requestName(r)
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Enabled && tool.IsRuntimeTool(name) && !s.cfg.Tools.ExecEnabled {
		writeErr(w, http.StatusBadRequest, ErrExecToolsDisabled)
		return
	}
	if in.Enabled && tool.IsBrowserTool(name) && !s.cfg.Tools.BrowserEnabled {
		writeErr(w, http.StatusBadRequest, ErrBrowserToolDisabled)
		return
	}
	if in.Enabled && tool.IsContentTool(name) && !s.cfg.Tools.DocumentEnabled {
		writeErr(w, http.StatusBadRequest, ErrDocumentToolDisabled)
		return
	}
	if err := s.st.SetToolEnabled(r.Context(), name, in.Enabled); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
		return
	}
	// LocalMode is single-tenant: keep in-memory registry in sync for Definitions().
	// Serve multi-tenant reads enable state from store per request/run.
	if s.cfg.LocalMode {
		s.tools.SetEnabled(name, in.Enabled)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// --- skills ---

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	roots := s.cfg.RootsForTenant(tenant.ID(r.Context()))
	cat := s.skills.ForUserDir(roots.Skills)
	all := cat.Discover(r.Context())
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
	slug := requestSlug(r)
	var in struct {
		Enabled bool `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if err := s.st.SetSkillEnabled(r.Context(), slug, in.Enabled); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) reloadSkills(w http.ResponseWriter, _ *http.Request) {
	// Discovery is on-demand; nothing to cache-clear here besides the disabled
	// set, which is read fresh. Acknowledge.
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

func (s *Server) listSkillDrafts(w http.ResponseWriter, r *http.Request) {
	roots := s.cfg.RootsForTenant(tenant.ID(r.Context()))
	cat := s.skills.ForUserDir(roots.Skills)
	drafts, err := cat.ListDrafts()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	if drafts == nil {
		drafts = []skill.Draft{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"drafts": drafts})
}

func (s *Server) acceptSkillDraft(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if id == "" {
		writeErr(w, http.StatusBadRequest, ErrIDRequired)
		return
	}
	roots := s.cfg.RootsForTenant(tenant.ID(r.Context()))
	cat := s.skills.ForUserDir(roots.Skills)
	if err := cat.AcceptDraft(id); err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) deleteSkillDraft(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if id == "" {
		writeErr(w, http.StatusBadRequest, ErrIDRequired)
		return
	}
	roots := s.cfg.RootsForTenant(tenant.ID(r.Context()))
	cat := s.skills.ForUserDir(roots.Skills)
	if err := cat.DeleteDraft(id); err != nil {
		writeErr(w, http.StatusBadRequest, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
