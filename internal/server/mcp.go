package server

import (
	"net/http"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func (s *Server) listMCPServers(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListMCPServers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": list})
}

func (s *Server) createMCPServer(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string   `json:"name"`
		Type string   `json:"type"`
		Cmd  string   `json:"cmd"`
		Args []string `json:"args"`
		URL  string   `json:"url"`

		Env map[string]string `json:"env"`

		Enabled *bool `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and type required"})
		return
	}
	switch in.Type {
	case "stdio":
		if in.Cmd == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cmd required for stdio"})
			return
		}
	case "sse", "streamable":
		if in.URL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required for " + in.Type})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "type must be stdio, sse, or streamable"})
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	srv := &store.MCPServer{
		ID: support.NewID(), Name: in.Name,
		Type: in.Type, Cmd: in.Cmd, Args: in.Args,
		URL: in.URL, Env: in.Env, Enabled: enabled,
	}
	if err := s.st.CreateMCPServer(r.Context(), srv); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "create failed"})
		return
	}
	if err := s.mcp.Sync(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mcp sync failed"})
		return
	}
	writeJSON(w, http.StatusCreated, srv)
}

func (s *Server) getMCPServer(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	srv, err := s.st.GetMCPServerByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, srv)
}

func (s *Server) updateMCPServer(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	var in map[string]any
	if !bindJSON(w, r, &in) {
		return
	}
	allowed := map[string]bool{
		"type": true, "cmd": true,
		"args": true, "url": true, "env": true, "enabled": true,
	}
	fields := map[string]any{}
	for k, v := range in {
		if allowed[k] {
			fields[k] = v
		}
	}
	if err := s.st.UpdateMCPServer(r.Context(), id, fields); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	if err := s.mcp.Sync(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mcp sync failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteMCPServer(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	srv, err := s.st.GetMCPServerByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err := s.st.DeleteMCPServer(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	_ = srv
	if err := s.mcp.Sync(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mcp sync failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) reloadMCP(w http.ResponseWriter, r *http.Request) {
	if err := s.mcp.Sync(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "sync failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}

func (s *Server) getMCPServerCapabilities(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if _, err := s.st.GetMCPServerByID(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	caps, err := s.mcp.ServerCapabilities(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, caps)
}
