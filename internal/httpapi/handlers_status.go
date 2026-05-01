package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	player := map[string]any{}
	if s.mpdReady {
		ctx, cancel := context.WithTimeout(r.Context(), 1500*time.Millisecond)
		st, err := s.mpd.Status(ctx)
		cancel()
		if err == nil {
			player = map[string]any{
				"status": st.State,
				"volume": st.Volume,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"player": player,
		"queue":  map[string]any{},
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
