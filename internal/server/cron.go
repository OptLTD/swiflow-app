package server

import (
	"net/http"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func (s *Server) listCronJobs(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListCronJobs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": list})
}

func (s *Server) createCronJob(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Agent    string `json:"agent"`
		Message  string `json:"message"`
		Schedule string `json:"schedule"`
		Enabled  *bool  `json:"enabled"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Name == "" || in.Agent == "" || in.Message == "" || in.Schedule == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, agent, message, schedule required"})
		return
	}
	if _, err := s.st.GetAgentByKey(r.Context(), in.Agent); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown agent"})
		return
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	job := &store.CronJob{
		ID: support.NewID(), Name: in.Name, Agent: in.Agent,
		Message: in.Message, Schedule: in.Schedule, Enabled: enabled,
	}
	if err := s.st.CreateCronJob(r.Context(), job); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "create failed"})
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cron reload failed"})
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) updateCronJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in map[string]any
	if !bindJSON(w, r, &in) {
		return
	}
	allowed := map[string]bool{"agent": true, "message": true, "schedule": true, "enabled": true}
	fields := map[string]any{}
	for k, v := range in {
		if allowed[k] {
			fields[k] = v
		}
	}
	if ak, ok := fields["agent"].(string); ok {
		if _, err := s.st.GetAgentByKey(r.Context(), ak); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown agent"})
			return
		}
	}
	if err := s.st.UpdateCronJob(r.Context(), id, fields); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cron reload failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteCronJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.st.DeleteCronJob(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cron reload failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) reloadCron(w http.ResponseWriter, r *http.Request) {
	if err := s.cron.Reload(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "reload failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}
