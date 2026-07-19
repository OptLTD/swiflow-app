package server

import (
	"net/http"
	"strings"
)

func (s *Server) providersAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getProvider(w, r)
	case "set":
		if !requireID(w, id) {
			return
		}
		s.updateProvider(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	case "del":
		if !requireID(w, id) {
			return
		}
		s.deleteProvider(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) agentsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getAgent(w, r)
	case "set":
		if !requireID(w, id) {
			return
		}
		s.updateAgent(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	default:
		unknownAct(w, act)
	}
}

func (s *Server) sessionsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getSession(w, r)
	case "del":
		if !requireID(w, id) {
			return
		}
		s.deleteSession(w, r)
	case "watch":
		if !requireID(w, id) {
			return
		}
		s.watchSession(w, r)
	case "children":
		if !requireID(w, id) {
			return
		}
		s.listSessionChildren(w, r)
	case "chat":
		if !requireID(w, id) {
			return
		}
		s.chat(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	case "abort":
		if !requireID(w, id) {
			return
		}
		s.abort(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) runsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getRun(w, r)
	case "watch":
		s.watchRuns(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) toolsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	name := rawString(raw, "name")
	r = withActValues(r, actValues{Name: name})
	switch act {
	case "set":
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
			return
		}
		s.setToolEnabled(w, withJSONBody(r, bodyWithoutAct(raw, "name")))
	default:
		unknownAct(w, act)
	}
}

func (s *Server) settingsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	switch act {
	case "search-get":
		s.getSearchSettings(w, r)
	case "search-set":
		s.putSearchSettings(w, withJSONBody(r, bodyWithoutAct(raw)))
	default:
		unknownAct(w, act)
	}
}

func (s *Server) skillsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	slug := rawString(raw, "slug")
	r = withActValues(r, actValues{ID: id, Slug: slug})
	switch act {
	case "set":
		if slug == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "slug required"})
			return
		}
		s.setSkillEnabled(w, withJSONBody(r, bodyWithoutAct(raw, "slug", "id")))
	case "reload":
		s.reloadSkills(w, r)
	case "draft-accept":
		if !requireID(w, id) {
			return
		}
		s.acceptSkillDraft(w, r)
	case "draft-del":
		if !requireID(w, id) {
			return
		}
		s.deleteSkillDraft(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) mcpAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getMCPServer(w, r)
	case "set":
		if !requireID(w, id) {
			return
		}
		s.updateMCPServer(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	case "del":
		if !requireID(w, id) {
			return
		}
		s.deleteMCPServer(w, r)
	case "capabilities":
		if !requireID(w, id) {
			return
		}
		s.getMCPServerCapabilities(w, r)
	case "reload":
		s.reloadMCP(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) cronAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	r = withActValues(r, actValues{ID: id})
	switch act {
	case "set":
		if !requireID(w, id) {
			return
		}
		s.updateCronJob(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	case "del":
		if !requireID(w, id) {
			return
		}
		s.deleteCronJob(w, r)
	case "reload":
		s.reloadCron(w, r)
	default:
		unknownAct(w, act)
	}
}

func (s *Server) workspaceAct(w http.ResponseWriter, r *http.Request) {
	// Multipart upload: act via query or form.
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(strings.ToLower(ct), "multipart/") {
		act := strings.TrimSpace(r.URL.Query().Get("act"))
		if act == "" {
			act = "upload"
		}
		if act != "upload" {
			unknownAct(w, act)
			return
		}
		s.uploadWorkspace(w, r)
		return
	}

	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	switch act {
	case "download":
		s.downloadFile(w, withJSONBody(r, bodyWithoutAct(raw)))
	case "upload":
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "upload requires multipart form"})
	default:
		unknownAct(w, act)
	}
}

func (s *Server) systemAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	switch act {
	case "open-data":
		s.openDataDir(w, r)
	case "open-log":
		s.openLogFile(w, r)
	case "open-url":
		s.openURL(w, withJSONBody(r, bodyWithoutAct(raw)))
	default:
		unknownAct(w, act)
	}
}

func (s *Server) lightAppsAct(w http.ResponseWriter, r *http.Request) {
	raw, act, ok := readActBody(w, r)
	if !ok {
		return
	}
	id := rawString(raw, "id")
	key := rawString(raw, "key")
	r = withActValues(r, actValues{ID: id, Key: key})
	switch act {
	case "get":
		if !requireID(w, id) {
			return
		}
		s.getLightApp(w, r)
	case "set":
		if !requireID(w, id) {
			return
		}
		s.updateLightApp(w, withJSONBody(r, bodyWithoutAct(raw, "id")))
	case "del":
		if !requireID(w, id) {
			return
		}
		s.deleteLightApp(w, r)
	case "launch":
		if !requireID(w, id) {
			return
		}
		s.launchLightApp(w, r)
	case "stop":
		if !requireID(w, id) {
			return
		}
		s.stopLightApp(w, r)
	case "env-list":
		s.listLightAppEnv(w, r)
	case "env-set":
		s.setLightAppEnv(w, withJSONBody(r, bodyWithoutAct(raw)))
	case "env-del":
		if key == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "key required"})
			return
		}
		s.deleteLightAppEnv(w, r)
	default:
		unknownAct(w, act)
	}
}
