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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"player": player,
		"queue":  queue,
		"bluetooth": map[string]any{
			"scanning":   false,
			"defaultMac": "",
			"connected":  nil,
		},
		"tasks": map[string]any{
			"items": []any{},
		},
	})
}
