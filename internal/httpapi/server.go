package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/refreshcoder/musicbox-unraid/internal/mpd"
	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

type Server struct {
	mux      *http.ServeMux
	ws       *ws.Hub
	mpd      mpd.Client
	mpdReady bool
}

type Options struct {
	StaticDir string
	MPDAddr   string
}

func NewServer(opts Options) (*Server, error) {
	mux := http.NewServeMux()
	s := &Server{
		mux: mux,
		ws:  ws.NewHub(),
		mpd: mpd.Client{Addr: opts.MPDAddr},
	}
	s.mpdReady = opts.MPDAddr != ""

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
			if !s.mpdReady {
				s.ws.Broadcast(ws.NewEvent("player.progress", map[string]any{
					"positionMs": 0,
					"durationMs": 0,
					"bitrateKbps": 0,
				}))
				continue
			}

			reqCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
			st, err := s.mpd.Status(reqCtx)
			cancel()
			if err != nil {
				s.ws.Broadcast(ws.NewEvent("player.progress", map[string]any{
					"positionMs": 0,
					"durationMs": 0,
					"bitrateKbps": 0,
				}))
				continue
			}

			s.ws.Broadcast(ws.NewEvent("player.progress", map[string]any{
				"positionMs": st.ElapsedMs,
				"durationMs": st.DurationMs,
				"bitrateKbps": st.BitrateKbps,
			}))
		}
	}
}
