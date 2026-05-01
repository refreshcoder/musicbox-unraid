package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleQueueGet(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()

	items, err := s.mpd.Queue(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
	})
}

type queueAddReq struct {
	Path string `json:"path"`
}

func (s *Server) handleQueueAdd(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}

	var req queueAddReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()

	if err := s.mpd.Add(ctx, req.Path); err != nil {
		writeError(w, http.StatusBadRequest, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleQueueClear(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()

	if err := s.mpd.Clear(ctx); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type queueRemoveReq struct {
	Pos int `json:"pos"`
}

func (s *Server) handleQueueRemove(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}

	var req queueRemoveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()

	if err := s.mpd.Delete(ctx, req.Pos); err != nil {
		writeError(w, http.StatusBadRequest, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

