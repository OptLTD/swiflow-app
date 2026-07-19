package server

import (
	"net/http"

	"github.com/OptLTD/swiflow/internal/harness"
	"github.com/OptLTD/swiflow/internal/store"
)

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	if s.harness == nil {
		writeJSON(w, http.StatusOK, map[string]any{"runs": []any{}})
		return
	}
	// Include finished roots so humans can review drift after a run ends.
	runs := s.harness.List(true)
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	if s.harness == nil {
		writeErr(w, http.StatusNotFound, ErrHarnessUnavailable)
		return
	}
	snap, ok := s.harness.Snapshot(id)
	if !ok {
		writeErr(w, http.StatusNotFound, ErrRunNotFound)
		return
	}
	children := s.harness.ListChildren(id)
	writeJSON(w, http.StatusOK, map[string]any{"run": snap, "children": children})
}

func (s *Server) watchRuns(w http.ResponseWriter, r *http.Request) {
	if s.harness == nil {
		writeErr(w, http.StatusServiceUnavailable, ErrHarnessUnavailable)
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

	ch, cancel := s.harness.SubscribeRuns()
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
			if _, err := w.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

type sessionChildDTO struct {
	Session *store.Session      `json:"session,omitempty"`
	Run     *harness.RunSnapshot `json:"run,omitempty"`
}

func (s *Server) listSessionChildren(w http.ResponseWriter, r *http.Request) {
	id := requestID(r)
	list, err := s.st.ListSessions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, ErrListFailed)
		return
	}
	byID := map[string]sessionChildDTO{}
	for i := range list {
		sess := list[i]
		if sess.Parent != id {
			continue
		}
		sCopy := sess
		dto := sessionChildDTO{Session: &sCopy}
		if s.harness != nil {
			if snap, ok := s.harness.Snapshot(sess.ID); ok {
				cp := snap
				dto.Run = &cp
			}
		}
		byID[sess.ID] = dto
	}
	if s.harness != nil {
		for _, snap := range s.harness.ListChildren(id) {
			if _, ok := byID[snap.SessionID]; ok {
				continue
			}
			cp := snap
			byID[snap.SessionID] = sessionChildDTO{Run: &cp}
		}
	}
	out := make([]sessionChildDTO, 0, len(byID))
	for _, dto := range byID {
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, map[string]any{"children": out})
}
