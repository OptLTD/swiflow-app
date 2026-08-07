package server

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/OptLTD/swiflow/internal/auth"
	"github.com/OptLTD/swiflow/internal/tenant"
	"github.com/OptLTD/swiflow/library/support"
)

// publicAPI paths that skip auth (health + login/register).
func isPublicAPI(path string) bool {
	switch path {
	case "/api/health", "/api/auth/login", "/api/auth/register", "/api/auth/mode":
		return true
	default:
		return false
	}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if isPublicAPI(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if s.cfg.LocalMode {
			ctx := tenant.WithID(r.Context(), tenant.DefaultID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		tok := bearerToken(r)
		sess := s.sessions.Get(tok)
		if sess == nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx := tenant.WithID(r.Context(), sess.Tid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if c, err := r.Cookie("swiflow_token"); err == nil {
		return c.Value
	}
	return ""
}

func (s *Server) authMode(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"local_mode": s.cfg.LocalMode,
		"auth":       !s.cfg.LocalMode,
	})
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	tid := tenant.ID(r.Context())
	t, err := s.st.GetTenantByID(r.Context(), tid)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"tid": tid, "name": tid})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tid": t.ID, "name": t.Name})
}

type loginBody struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.LocalMode {
		writeErr(w, http.StatusBadRequest, "local mode: login not required")
		return
	}
	var in loginBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.Password == "" {
		writeErr(w, http.StatusBadRequest, "name and password required")
		return
	}
	t, err := s.st.GetTenantByName(r.Context(), in.Name)
	if err != nil || t == nil || !t.Enabled {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if t.PasswordHash == "" || !auth.CheckPassword(t.PasswordHash, in.Password) {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	sess, err := s.sessions.Create(t.ID, t.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "session error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": sess.Token,
		"tid":   t.ID,
		"name":  t.Name,
	})
}

func (s *Server) authRegister(w http.ResponseWriter, r *http.Request) {
	if s.cfg.LocalMode {
		writeErr(w, http.StatusBadRequest, "local mode: register not required")
		return
	}
	var in loginBody
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "name and password (min 6) required")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash error")
		return
	}
	id := support.NewID()
	if err := s.st.CreateTenant(r.Context(), id, in.Name, hash); err != nil {
		writeErr(w, http.StatusConflict, "tenant name taken")
		return
	}
	// Ensure tenant disk roots exist.
	roots := s.cfg.RootsForTenant(id)
	_ = ensureDirs(roots.Workspace, roots.Skills, roots.LightApps)

	sess, err := s.sessions.Create(id, in.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "session error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": sess.Token,
		"tid":   id,
		"name":  in.Name,
	})
}

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	if tok := bearerToken(r); tok != "" {
		s.sessions.Delete(tok)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ensureDirs(dirs ...string) error {
	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}
// WarnLocalModeListen returns an error when LocalMode would listen on a non-loopback address.
func WarnLocalModeListen(addr string, localMode bool) error {
	if !localMode {
		return nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "127.0.0.1" || host == "localhost" || host == "::1" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return errLocalModeNonLoopback
}

var errLocalModeNonLoopback = errString("LocalMode refuses non-loopback listen address")

type errString string

func (e errString) Error() string { return string(e) }
