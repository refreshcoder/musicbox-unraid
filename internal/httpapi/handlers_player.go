package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handlePlayerPlay(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.Play(ctx); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlayerPause(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.Pause(ctx, true); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlayerToggle(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	st, err := s.mpd.Status(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	if st.State == "play" {
		if err := s.mpd.Pause(ctx, true); err != nil {
			writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
			return
		}
	} else {
		if err := s.mpd.Play(ctx); err != nil {
			writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlayerNext(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.Next(ctx); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handlePlayerPrev(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.Prev(ctx); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type seekReq struct {
	PositionMs int64 `json:"positionMs"`
}

func (s *Server) handlePlayerSeek(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	var req seekReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.SeekMs(ctx, req.PositionMs); err != nil {
		writeError(w, http.StatusBadGateway, "mpd_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type volumeReq struct {
	Volume int `json:"volume"`
}

func (s *Server) handlePlayerVolume(w http.ResponseWriter, r *http.Request) {
	if !s.mpdReady {
		writeError(w, http.StatusServiceUnavailable, "mpd_unconfigured", "MPD is not configured")
		return
	}
	var req volumeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "Invalid JSON body")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
	defer cancel()
	if err := s.mpd.SetVol(ctx, req.Volume); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_volume", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

