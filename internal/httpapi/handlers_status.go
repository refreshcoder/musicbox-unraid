package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"player": map[string]any{},
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

