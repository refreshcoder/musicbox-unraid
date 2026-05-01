package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

type Server struct {
	mux *http.ServeMux
	ws  *ws.Hub
}

type Options struct {
	StaticDir string
}

func NewServer(opts Options) (*Server, error) {
	mux := http.NewServeMux()
	s := &Server{
		mux: mux,
		ws:  ws.NewHub(),
	}

	s.routes()

	if opts.StaticDir != "" {
		mux.Handle("GET /", StaticDir(opts.StaticDir).Handler())
	}

	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	s.mux.HandleFunc("GET /ws", s.handleWS)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Hub() *ws.Hub {
	return s.ws
}

func (s *Server) Start(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.ws.Broadcast(ws.NewEvent("player.progress", map[string]any{
				"positionMs": 0,
				"durationMs": 0,
				"bitrateKbps": 0,
			}))
		}
	}
}

