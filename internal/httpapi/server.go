package httpapi

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/refreshcoder/musicbox-unraid/internal/bluetooth"
	"github.com/refreshcoder/musicbox-unraid/internal/mpd"
	"github.com/refreshcoder/musicbox-unraid/internal/tasks"
	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

type Server struct {
	mux      *http.ServeMux
	ws       *ws.Hub
	mpd      mpd.Client
	mpdReady bool

	btMu         sync.RWMutex
	btCtl        bluetooth.Ctl
	btScanning   bool
	btDefaultMac string

	tasks *tasks.Manager
}

type Options struct {
	StaticDir string
	MPDAddr   string
	MusicDir  string
}

func NewServer(opts Options) (*Server, error) {
	mux := http.NewServeMux()
	s := &Server{
		mux: mux,
		ws:  ws.NewHub(),
		mpd: mpd.Client{Addr: opts.MPDAddr},
		btCtl: bluetooth.Ctl{},
	}
	s.mpdReady = opts.MPDAddr != ""

	tm, err := tasks.NewManager(tasks.Options{
		Runner:   tasks.ExecRunner{},
		MPD:      func() tasks.MPD { if s.mpdReady { return s.mpd }; return nil }(),
		MusicDir: opts.MusicDir,
	})
	if err != nil {
		return nil, err
	}
	s.tasks = tm

	s.routes()

	if opts.StaticDir != "" {
		mux.Handle("GET /", StaticDir(opts.StaticDir).Handler())
	}

	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	s.mux.HandleFunc("POST /api/v1/player/play", s.handlePlayerPlay)
	s.mux.HandleFunc("POST /api/v1/player/pause", s.handlePlayerPause)
	s.mux.HandleFunc("POST /api/v1/player/toggle", s.handlePlayerToggle)
	s.mux.HandleFunc("POST /api/v1/player/next", s.handlePlayerNext)
	s.mux.HandleFunc("POST /api/v1/player/prev", s.handlePlayerPrev)
	s.mux.HandleFunc("POST /api/v1/player/seek", s.handlePlayerSeek)
	s.mux.HandleFunc("POST /api/v1/player/volume", s.handlePlayerVolume)
	s.mux.HandleFunc("GET /api/v1/queue", s.handleQueueGet)
	s.mux.HandleFunc("POST /api/v1/queue/add", s.handleQueueAdd)
	s.mux.HandleFunc("POST /api/v1/queue/clear", s.handleQueueClear)
	s.mux.HandleFunc("POST /api/v1/queue/remove", s.handleQueueRemove)
	s.mux.HandleFunc("GET /api/v1/bluetooth/status", s.handleBluetoothStatus)
	s.mux.HandleFunc("POST /api/v1/bluetooth/scan/start", s.handleBluetoothScanStart)
	s.mux.HandleFunc("POST /api/v1/bluetooth/scan/stop", s.handleBluetoothScanStop)
	s.mux.HandleFunc("GET /api/v1/bluetooth/devices", s.handleBluetoothDevices)
	s.mux.HandleFunc("POST /api/v1/bluetooth/devices/{mac}/connect", s.handleBluetoothConnect)
	s.mux.HandleFunc("POST /api/v1/bluetooth/devices/{mac}/disconnect", s.handleBluetoothDisconnect)
	s.mux.HandleFunc("PUT /api/v1/bluetooth/default", s.handleBluetoothDefaultSet)
	s.mux.HandleFunc("DELETE /api/v1/bluetooth/default", s.handleBluetoothDefaultClear)
	s.mux.HandleFunc("GET /api/v1/tasks", s.handleTasksList)
	s.mux.HandleFunc("POST /api/v1/tasks/bv", s.handleTasksBV)
	s.mux.HandleFunc("POST /api/v1/tasks/{id}/cancel", s.handleTasksCancel)
	s.mux.HandleFunc("GET /ws", s.handleWS)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) Hub() *ws.Hub {
	return s.ws
}

func (s *Server) Start(ctx context.Context) {
	go s.tasks.RunWorker(ctx,
		func(t *tasks.Task) {
			s.ws.Broadcast(ws.NewEvent("task.update", t))
		},
		func(t *tasks.Task) {
			s.ws.Broadcast(ws.NewEvent("task.done", t))
		},
	)

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
