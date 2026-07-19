package server

import (
	"net/http"

	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func (s *Server) listCronJobs(w http.ResponseWriter, r *http.Request) {
	list, err := s.st.ListCronJobs(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
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
		writeErr(w, http.StatusBadRequest, ErrCronFieldsRequired)
		return
	}
	if _, err := s.st.GetAgentByKey(r.Context(), in.Agent); err != nil {
		writeErr(w, http.StatusBadRequest, ErrUnknownAgent)
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
		writeErr(w, http.StatusConflict, ErrCreateFailed)
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrCronReloadFailed)
		return
	}
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) updateCronJob(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
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
			writeErr(w, http.StatusBadRequest, ErrUnknownAgent)
			return
		}
	}
	if err := s.st.UpdateCronJob(r.Context(), id, fields); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrCronReloadFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteCronJob(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if err := s.st.DeleteCronJob(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrDeleteFailed)
		return
	}
	if err := s.cron.Reload(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrCronReloadFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) reloadCron(w http.ResponseWriter, r *http.Request) {
	if err := s.cron.Reload(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrReloadFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reloaded"})
}
