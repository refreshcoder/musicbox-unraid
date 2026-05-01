package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

type tasksListResp struct {
	Items any `json:"items"`
}

func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, tasksListResp{
		Items: s.tasks.List(),
	})
}

type tasksBVReq struct {
	BV string `json:"bv"`
}

func (s *Server) handleTasksBV(w http.ResponseWriter, r *http.Request) {
	var req tasksBVReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}

	t, err := s.tasks.EnqueueBV(req.BV)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_bv", err.Error())
		return
	}

	s.ws.Broadcast(ws.NewEvent("task.update", t))
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleTasksCancel(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_id", "Missing task id")
		return
	}

	if !s.tasks.Cancel(id) {
		writeError(w, http.StatusNotFound, "not_found", "Task not found or not cancelable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
