package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/OptLTD/swiflow/internal/lightapp"
	"github.com/OptLTD/swiflow/internal/store"
	"github.com/OptLTD/swiflow/library/support"
)

func (s *Server) listLightApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.st.ListLightApps(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
		return
	}
	// Overlay live runtime status from the manager.
	if s.lightMgr != nil {
		for i := range apps {
			apps[i].Status = s.lightMgr.Status(apps[i].ID)
			if apps[i].Status == "running" {
				apps[i].Port = s.lightMgr.RunningPort(apps[i].ID)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"apps": apps})
}

func (s *Server) createLightApp(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Runtime     string `json:"runtime"`
		EntryPoint  string `json:"entry_point"`
	}
	if !bindJSON(w, r, &in) {
		return
	}
	if in.Name == "" {
		writeErr(w, http.StatusBadRequest, ErrNameRequired)
		return
	}
	if in.Runtime == "" {
		in.Runtime = "python"
	}
	if in.Runtime != "python" && in.Runtime != "static" {
		writeErr(w, http.StatusBadRequest, ErrInvalidLightAppRuntime)
		return
	}
	if in.EntryPoint == "" {
		if in.Runtime == "python" {
			in.EntryPoint = "app.py"
		} else {
			in.EntryPoint = "index.html"
		}
	}
	id := support.NewID()
	if s.lightMgr != nil {
		if err := s.lightMgr.EnsureDir(id); err != nil {
			writeErr(w, http.StatusInternalServerError, ErrCreateAppDirFailed)
			return
		}
	}
	a := &store.LightApp{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		Runtime:     in.Runtime,
		EntryPoint:  in.EntryPoint,
		Status:      "stopped",
	}
	if err := s.st.CreateLightApp(r.Context(), a); err != nil {
		writeErr(w, http.StatusConflict, ErrCreateFailed)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) getLightApp(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	a, err := s.st.GetLightAppByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if s.lightMgr != nil {
		a.Status = s.lightMgr.Status(id)
		if a.Status == "running" {
			a.Port = s.lightMgr.RunningPort(id)
		}
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) updateLightApp(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if _, err := s.st.GetLightAppByID(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	var in map[string]any
	if !bindJSON(w, r, &in) {
		return
	}
	allowed := map[string]bool{"name": true, "description": true, "runtime": true, "entry_point": true}
	fields := map[string]any{}
	for k, v := range in {
		if allowed[k] {
			fields[k] = v
		}
	}
	if err := s.st.UpdateLightApp(r.Context(), id, fields); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrUpdateFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteLightApp(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if _, err := s.st.GetLightAppByID(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if s.lightMgr != nil {
		s.lightMgr.Stop(id)
	}
	if err := s.st.DeleteLightApp(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrDeleteFailed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) launchLightApp(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	a, err := s.st.GetLightAppByID(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if s.lightMgr == nil {
		writeErr(w, http.StatusServiceUnavailable, ErrLightAppManagerUnavailable)
		return
	}
	entryPoint := a.EntryPoint
	if !filepath.IsAbs(entryPoint) {
		entryPoint = filepath.Join(s.lightMgr.AppDir(id), entryPoint)
	}
	extraEnv, _ := s.st.ListLightAppEnv(r.Context())
	url, port, err := s.lightMgr.Launch(r.Context(), id, lightapp.LaunchConfig{
		EntryPoint: a.EntryPoint,
		Runtime:    lightapp.Runtime(a.Runtime),
		ExtraEnv:   extraEnv,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	_ = s.st.UpdateLightApp(r.Context(), id, map[string]any{"status": "running", "port": port})
	writeJSON(w, http.StatusOK, map[string]any{"url": url, "port": port})
}

func (s *Server) stopLightApp(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if _, err := s.st.GetLightAppByID(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if s.lightMgr != nil {
		s.lightMgr.Stop(id)
	}
	_ = s.st.UpdateLightApp(r.Context(), id, map[string]any{"status": "stopped", "port": 0})
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Server) listLightAppEnv(w http.ResponseWriter, r *http.Request) {
	env, err := s.st.ListLightAppEnv(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"env": env})
}

func (s *Server) setLightAppEnv(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}
	if body.Key == "" {
		writeErr(w, http.StatusBadRequest, ErrKeyAndValueRequired)
		return
	}
	if err := s.st.SetLightAppEnv(r.Context(), body.Key, body.Value); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteLightAppEnv(w http.ResponseWriter, r *http.Request) {
	key := requestKey(r)
	if err := s.st.DeleteLightAppEnv(r.Context(), key); err != nil {
		writeErr(w, http.StatusInternalServerError, ErrInternalError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
