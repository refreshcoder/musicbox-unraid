package tasks

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusQueued   Status = "queued"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

type Task struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Input      string `json:"input"`
	Status     Status `json:"status"`
	Stage      string `json:"stage,omitempty"`
	Progress01 float64 `json:"progress01,omitempty"`
	ResultPath string `json:"resultPath,omitempty"`
	Error      string `json:"error,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`

	cancel context.CancelFunc
}

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (stdout string, err error)
}

type MPD interface {
	Update(ctx context.Context) error
}

type Manager struct {
	mu       sync.RWMutex
	items    []*Task
	runner   Runner
	mpd      MPD
	musicDir string
}

type Options struct {
	Runner   Runner
	MPD      MPD
	MusicDir string
}

func NewManager(opts Options) (*Manager, error) {
	if opts.Runner == nil {
		return nil, errors.New("missing runner")
	}
	musicDir := strings.TrimSpace(opts.MusicDir)
	if musicDir == "" {
		musicDir = "/srv/music"
	}
	return &Manager{
		items:    []*Task{},
		runner:   opts.Runner,
		mpd:      opts.MPD,
		musicDir: musicDir,
	}, nil
}

func (m *Manager) MusicDir() string {
	return m.musicDir
}

func (m *Manager) IncomingDir() string {
	return filepath.Join(m.musicDir, ".incoming")
}

func (m *Manager) List() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Task, 0, len(m.items))
	out = append(out, m.items...)
	return out
}

func (m *Manager) EnqueueBV(bv string) (*Task, error) {
	bv = strings.TrimSpace(bv)
	if bv == "" {
		return nil, errors.New("bv is empty")
	}

	now := time.Now().Unix()
	t := &Task{
		ID:        newID(),
		Type:      "bv",
		Input:     bv,
		Status:    StatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.mu.Lock()
	m.items = append(m.items, t)
	m.mu.Unlock()

	return t, nil
}

func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.items {
		if t.ID != id {
			continue
		}
		switch t.Status {
		case StatusQueued:
			t.Status = StatusCanceled
			t.UpdatedAt = time.Now().Unix()
			return true
		case StatusRunning:
			if t.cancel != nil {
				t.cancel()
				return true
			}
			return false
		default:
			return false
		}
	}
	return false
}

func (m *Manager) RunWorker(
	ctx context.Context,
	onUpdate func(*Task),
	onDone func(*Task),
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		t := m.nextQueued()
		if t == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(300 * time.Millisecond):
				continue
			}
		}

		taskCtx, cancel := context.WithCancel(ctx)
		m.markRunning(t, cancel)
		if onUpdate != nil {
			onUpdate(t)
		}

		err := m.runOne(taskCtx, t, onUpdate)
		cancel()

		m.finish(t, err)
		if t.Status == StatusSuccess || t.Status == StatusFailed || t.Status == StatusCanceled {
			if onDone != nil {
				onDone(t)
			}
		}
	}
}

func (m *Manager) nextQueued() *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, t := range m.items {
		if t.Status == StatusQueued {
			return t
		}
	}
	return nil
}

func (m *Manager) markRunning(t *Task, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.Status != StatusQueued {
		return
	}
	t.Status = StatusRunning
	t.Stage = "download"
	t.Progress01 = 0
	t.cancel = cancel
	t.UpdatedAt = time.Now().Unix()
}

func (m *Manager) finish(t *Task, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if errors.Is(err, context.Canceled) {
		t.Status = StatusCanceled
		t.Error = ""
		t.Stage = ""
		t.Progress01 = 0
		t.UpdatedAt = time.Now().Unix()
		t.cancel = nil
		return
	}

	if err != nil {
		t.Status = StatusFailed
		t.Error = err.Error()
		t.Stage = ""
		t.Progress01 = 0
		t.UpdatedAt = time.Now().Unix()
		t.cancel = nil
		return
	}

	t.Status = StatusSuccess
	t.Error = ""
	t.Stage = ""
	t.Progress01 = 1
	t.UpdatedAt = time.Now().Unix()
	t.cancel = nil
}

func (m *Manager) setStage(t *Task, stage string, progress float64, onUpdate func(*Task)) {
	m.mu.Lock()
	if t.Status == StatusRunning {
		t.Stage = stage
		t.Progress01 = progress
		t.UpdatedAt = time.Now().Unix()
	}
	m.mu.Unlock()
	if onUpdate != nil {
		onUpdate(t)
	}
}

func (m *Manager) runOne(ctx context.Context, t *Task, onUpdate func(*Task)) error {
	incoming := m.IncomingDir()
	if err := os.MkdirAll(incoming, 0o755); err != nil {
		return err
	}

	switch t.Type {
	case "bv":
		return m.runBV(ctx, t, onUpdate)
	default:
		return errors.New("unknown task type")
	}
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

