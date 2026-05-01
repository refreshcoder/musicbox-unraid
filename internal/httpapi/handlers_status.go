package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	player := map[string]any{}
	queue := map[string]any{}
	if s.mpdReady {
		ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		st, err := s.mpd.Status(ctx)
		cancel()
		if err == nil {
			player = map[string]any{
				"status": st.State,
				"volume": st.Volume,
			}
			queue = map[string]any{
				"length":     st.QueueLength,
				"currentPos": st.Song,
			}
		}
	}

	s.btMu.RLock()
	btScanning := s.btScanning
	btDefaultMac := s.btDefaultMac
	s.btMu.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"player": player,
		"queue":  queue,
		"bluetooth": map[string]any{
			"scanning":   btScanning,
			"defaultMac": btDefaultMac,
			"connected":  nil,
		},
		"tasks": map[string]any{
			"items": s.tasks.List(),
		},
	})
}
