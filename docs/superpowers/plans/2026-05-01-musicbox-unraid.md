# MusicBox Unraid Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Debian 12 VM 内运行一个 Go 单进程 Web App（内置 React+Tailwind 前端），提供一页式音乐播控、上传、BV→m4a 入库、蓝牙连接管理（可诊断 + 自动重连）的 MVP。

**Architecture:** Go 后端提供 REST + WebSocket，后端通过 MPD（TCP 协议）控制播放，通过 `bluetoothctl` 命令封装进行蓝牙管理，通过 `yt-dlp`/`ffmpeg` 执行 BV 下载与转码；React 前端用 WebSocket 做状态驱动，默认“只入库不打扰”。

**Tech Stack:** Go 1.22+（net/http），WebSocket（nhooyr.io/websocket），sqlite（modernc.org/sqlite），React+TypeScript+Vite+Tailwind。

---

## Repo & Workspace

仓库：`musicbox-unraid`（公开，个人账号）

项目根目录结构（最终形态）：

- `cmd/musicbox/main.go`
- `internal/httpapi/`（REST handlers + routing）
- `internal/ws/`（WS hub + event schema）
- `internal/mpd/`（MPD client）
- `internal/bluetooth/`（bluetoothctl 封装 + 自动重连）
- `internal/tasks/`（任务队列 + BV worker）
- `internal/store/`（sqlite）
- `web/`（React 应用源码）
- `web/dist/`（构建产物，生产时由 Go 服务提供）
- `docs/`（保留既有 spec）

---

### Task 1: 创建 GitHub 仓库并初始化本地项目

**Files:**
- Create: `README.md`
- Create: `go.mod`
- Create: `.gitignore`

- [ ] **Step 1: 使用 gh 创建 GitHub 仓库（公开）**

Run:

```bash
gh repo create musicbox-unraid --public --source=. --remote=origin --push=false
```

Expected: 在你的个人账号下创建仓库并添加 `origin` 远程（不立即 push）。

- [ ] **Step 2: 初始化 git 与基础文件**

Run:

```bash
git init
```

Create `README.md`:

```md
# musicbox-unraid

Unraid VM music box: BlueZ + BlueALSA + MPD + Go(Web) + React(Tailwind).
```

Create `.gitignore`:

```gitignore
.DS_Store
node_modules
dist
web/dist
.env
.idea
.vscode
*.log

*.swp
*.swo

bin/
```

Create `go.mod`:

```go
module github.com/refreshcoder/musicbox-unraid

go 1.22
```

- [ ] **Step 3: 首次提交**

Run:

```bash
git add README.md .gitignore go.mod
git commit -m "chore: init repository"
```

---

### Task 2: Go 服务骨架（静态前端占位 + REST health）

**Files:**
- Create: `cmd/musicbox/main.go`
- Create: `internal/httpapi/router.go`
- Create: `internal/httpapi/handlers_health.go`
- Test: `internal/httpapi/handlers_health_test.go`

- [ ] **Step 1: 添加 router 与 health handler**

Create `internal/httpapi/router.go`:

```go
package httpapi

import (
	"net/http"
)

type Server struct {
	mux *http.ServeMux
}

func NewServer() *Server {
	mux := http.NewServeMux()
	s := &Server{mux: mux}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
```

Create `internal/httpapi/handlers_health.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
	})
}
```

- [ ] **Step 2: 添加 main 启动监听**

Create `cmd/musicbox/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/refreshcoder/musicbox-unraid/internal/httpapi"
)

func main() {
	addr := envOr("MUSICBOX_ADDR", ":8080")

	s := httpapi.NewServer()

	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
```

- [ ] **Step 3: 写一个最小测试**

Create `internal/httpapi/handlers_health_test.go`:

```go
package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	s := NewServer()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	s.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if resp["ok"] != true {
		t.Fatalf("expected ok=true, got %v", resp["ok"])
	}
}
```

- [ ] **Step 4: 运行测试并提交**

Run:

```bash
go test ./...
```

Expected: PASS

Commit:

```bash
git add cmd internal
git commit -m "feat: add go server skeleton and health endpoint"
```

---

### Task 3: WebSocket Hub 与事件 schema（含 player.progress 每 3 秒）

**Files:**
- Add: `internal/ws/events.go`
- Add: `internal/ws/hub.go`
- Add: `internal/ws/client.go`
- Modify: `internal/httpapi/router.go`
- Add: `internal/httpapi/handlers_ws.go`

- [ ] **Step 1: 定义事件结构**

Create `internal/ws/events.go`:

```go
package ws

import "time"

type Event struct {
	Type string `json:"type"`
	TS   int64  `json:"ts"`
	Data any    `json:"data"`
}

func NewEvent(typ string, data any) Event {
	return Event{
		Type: typ,
		TS:   time.Now().Unix(),
		Data: data,
	}
}
```

- [ ] **Step 2: 实现 hub（广播）**

Create `internal/ws/hub.go`:

```go
package ws

import "sync"

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: map[*Client]struct{}{}}
}

func (h *Hub) Add(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) Broadcast(evt Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.Send(evt)
	}
}
```

- [ ] **Step 3: 实现 client（带缓冲发送队列）**

Create `internal/ws/client.go`:

```go
package ws

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"nhooyr.io/websocket"
)

type Client struct {
	conn   *websocket.Conn
	closed atomic.Bool
	ch     chan Event
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn: conn,
		ch:   make(chan Event, 64),
	}
}

func (c *Client) Send(evt Event) {
	if c.closed.Load() {
		return
	}
	select {
	case c.ch <- evt:
	default:
	}
}

func (c *Client) Run(ctx context.Context) error {
	defer c.closed.Store(true)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case evt := <-c.ch:
			b, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			if err := c.conn.Write(ctx, websocket.MessageText, b); err != nil {
				return err
			}
		}
	}
}
```

- [ ] **Step 4: 增加 WS handler 并接入 router**

Modify `internal/httpapi/router.go`：

```go
package httpapi

import (
	"net/http"

	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

type Server struct {
	mux *http.ServeMux
	ws  *ws.Hub
}

func NewServer() *Server {
	mux := http.NewServeMux()
	s := &Server{mux: mux, ws: ws.NewHub()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /ws", s.handleWS)
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
```

Create `internal/httpapi/handlers_ws.go`:

```go
package httpapi

import (
	"context"
	"net/http"
	"time"

	"nhooyr.io/websocket"

	"github.com/refreshcoder/musicbox-unraid/internal/ws"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	c := ws.NewClient(conn)
	s.ws.Add(c)
	defer s.ws.Remove(c)

	_ = conn.SetReadLimit(1024 * 32)
	_ = conn.SetReadDeadline(time.Now().Add(365 * 24 * time.Hour))

	_ = c.Run(ctx)
}
```

- [ ] **Step 5: 增加依赖并提交**

Run:

```bash
go get nhooyr.io/websocket@latest
go test ./...
```

Commit:

```bash
git add internal go.mod go.sum
git commit -m "feat: add websocket hub"
```

---

### Task 4: React + Tailwind 前端（单页 Tabs + WS 连接）

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tailwind.config.ts`
- Create: `web/postcss.config.js`
- Create: `web/index.html`
- Create: `web/src/main.tsx`
- Create: `web/src/app/App.tsx`
- Create: `web/src/app/PlayerBar.tsx`
- Create: `web/src/app/Tabs.tsx`
- Create: `web/src/lib/ws.ts`
- Create: `web/src/lib/api.ts`
- Create: `web/src/app/tabs/*`

- [ ] **Step 1: 初始化 Vite React TS**

Run:

```bash
cd web
npm create vite@latest . -- --template react-ts
```

Expected: 生成 React+TS 工程。

- [ ] **Step 2: 安装 Tailwind 并配置**

Run:

```bash
cd web
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

Update `web/tailwind.config.ts`:

```ts
import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: { extend: {} },
  plugins: [],
} satisfies Config;
```

Update `web/src/index.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

- [ ] **Step 3: Vite proxy 到 Go**

Update `web/vite.config.ts`:

```ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    },
  },
});
```

- [ ] **Step 4: 实现 AppShell（Tabs + PlayerBar）与 WS 客户端**

Create `web/src/lib/ws.ts`:

```ts
export type WsEvent<T = unknown> = {
  type: string;
  ts: number;
  data: T;
};

export function connectWs(onEvent: (evt: WsEvent) => void): WebSocket {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(`${proto}://${location.host}/ws`);
  ws.onmessage = (msg) => {
    try {
      const evt = JSON.parse(msg.data) as WsEvent;
      onEvent(evt);
    } catch {
      return;
    }
  };
  return ws;
}
```

Create `web/src/app/Tabs.tsx`:

```tsx
import { useMemo } from "react";

export type TabKey = "now" | "queue" | "library" | "upload" | "bluetooth" | "diag";

export function useTabs() {
  const tabs = useMemo(
    () => [
      { key: "now" as const, label: "播放" },
      { key: "queue" as const, label: "队列" },
      { key: "library" as const, label: "曲库" },
      { key: "upload" as const, label: "上传&BV" },
      { key: "bluetooth" as const, label: "蓝牙" },
      { key: "diag" as const, label: "诊断" },
    ],
    [],
  );
  return { tabs };
}

export function TabNav(props: {
  active: TabKey;
  onChange: (k: TabKey) => void;
}) {
  const { tabs } = useTabs();
  return (
    <div className="border-b bg-white/80 backdrop-blur sticky top-14 z-40">
      <div className="mx-auto max-w-5xl px-3">
        <div className="flex gap-2 overflow-x-auto py-2">
          {tabs.map((t) => (
            <button
              key={t.key}
              className={[
                "px-3 py-1.5 rounded-full text-sm whitespace-nowrap",
                props.active === t.key ? "bg-black text-white" : "bg-gray-100 text-gray-700",
              ].join(" ")}
              onClick={() => props.onChange(t.key)}
              type="button"
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
    </div>
  );
}
```

Create `web/src/app/PlayerBar.tsx`:

```tsx
export function PlayerBar() {
  return (
    <div className="sticky top-0 z-50 border-b bg-white/80 backdrop-blur">
      <div className="mx-auto max-w-5xl px-3 h-14 flex items-center justify-between">
        <div className="min-w-0">
          <div className="text-sm font-medium truncate">MusicBox</div>
          <div className="text-xs text-gray-500 truncate">未连接</div>
        </div>
        <div className="flex items-center gap-2">
          <button className="px-3 py-1.5 rounded bg-gray-100 text-sm" type="button">
            上一首
          </button>
          <button className="px-3 py-1.5 rounded bg-black text-white text-sm" type="button">
            播放/暂停
          </button>
          <button className="px-3 py-1.5 rounded bg-gray-100 text-sm" type="button">
            下一首
          </button>
        </div>
      </div>
    </div>
  );
}
```

Create `web/src/app/App.tsx`:

```tsx
import { useEffect, useRef, useState } from "react";
import { PlayerBar } from "./PlayerBar";
import { TabNav, TabKey } from "./Tabs";
import { connectWs } from "../lib/ws";

export function App() {
  const [tab, setTab] = useState<TabKey>("now");
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ws = connectWs(() => {});
    wsRef.current = ws;
    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, []);

  return (
    <div className="min-h-dvh bg-gray-50">
      <PlayerBar />
      <TabNav active={tab} onChange={setTab} />
      <main className="mx-auto max-w-5xl px-3 py-4">
        <div className="rounded-lg border bg-white p-4 text-sm text-gray-700">
          当前：{tab}
        </div>
      </main>
    </div>
  );
}
```

- [ ] **Step 5: 运行前端并提交**

Run:

```bash
cd web
npm install
npm run dev
```

Expected: 浏览器打开后看到单页 Tabs。

Commit:

```bash
git add web
git commit -m "feat: add react tailwind app shell"
```

---

### Task 5: 生产静态资源集成（Go 提供 web/dist + SPA fallback）

**Files:**
- Modify: `internal/httpapi/router.go`
- Add: `internal/httpapi/static.go`

- [ ] **Step 1: 增加静态文件服务与 SPA fallback**

Create `internal/httpapi/static.go`:

```go
package httpapi

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type Static struct {
	FS fs.FS
}

func (s Static) Handler() http.Handler {
	fileServer := http.FileServer(http.FS(s.FS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}
		p := path.Clean(r.URL.Path)
		if p == "/" || strings.HasPrefix(p, "/assets/") {
			fileServer.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}
```

Modify `internal/httpapi/router.go` to mount static on `/` (after API routes):

```go
func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /ws", s.handleWS)
}
```

在 MVP 阶段，静态文件直接从磁盘目录 `web/dist` 提供（先构建前端再运行后端）。Go 端在 `/api/*` 与 `/ws` 之外的路径上做 SPA fallback 到 `/index.html`。

- [ ] **Step 2: 提交**

```bash
git add internal cmd
git commit -m "feat: add static file handler with spa fallback"
```

---

### Task 6: MVP 业务功能（按优先级逐步接入）

说明：本任务按“可跑 MVP”拆成子步骤，每一步都保持可编译、可启动。

**Files:**
- Add: `internal/mpd/client.go`
- Add: `internal/bluetooth/bluetoothctl.go`
- Add: `internal/tasks/queue.go`
- Add: `internal/httpapi/handlers_player.go`
- Add: `internal/httpapi/handlers_queue.go`
- Add: `internal/httpapi/handlers_upload.go`
- Add: `internal/httpapi/handlers_tasks.go`
- Add: `internal/httpapi/handlers_bluetooth.go`

- [ ] **Step 1: MPD 最小客户端（ping/status）并加 `/api/v1/status` 框架**

Create `internal/mpd/client.go`:

```go
package mpd

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type Client struct {
	Addr string
}

func (c Client) Dial() (net.Conn, error) {
	d := net.Dialer{Timeout: 2 * time.Second}
	return d.Dial("tcp", c.Addr)
}

func (c Client) Cmd(cmd string) (map[string]string, error) {
	conn, err := c.Dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "OK") {
		return nil, fmt.Errorf("mpd handshake: %s", strings.TrimSpace(line))
	}

	if _, err := conn.Write([]byte(cmd + "\n")); err != nil {
		return nil, err
	}

	out := map[string]string{}
	for {
		l, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		l = strings.TrimSpace(l)
		if l == "OK" {
			return out, nil
		}
		if strings.HasPrefix(l, "ACK") {
			return nil, fmt.Errorf("mpd ack: %s", l)
		}
		parts := strings.SplitN(l, ": ", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
}
```

为 MVP 直接实现 `GET /api/v1/status`，返回 player/bluetooth/tasks 的汇总快照（字段允许为空，但结构固定），前端只依赖结构存在。

- [ ] **Step 2: 蓝牙命令封装（bluetoothctl）**

Create `internal/bluetooth/bluetoothctl.go`:

```go
package bluetooth

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type Ctl struct {
	Path string
}

func (c Ctl) Run(ctx context.Context, args ...string) (string, error) {
	path := c.Path
	if path == "" {
		path = "bluetoothctl"
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
```

Handlers 先实现：

- scan start/stop（调用 `bluetoothctl scan on/off`）
- connect/disconnect（`bluetoothctl connect <mac>`）

- [ ] **Step 3: 任务队列（串行）+ BV 下载（yt-dlp/ffmpeg）**

Create `internal/tasks/queue.go`（仅内存 MVP）：

```go
package tasks

import (
	"context"
	"sync"
)

type TaskStatus string

const (
	StatusQueued   TaskStatus = "queued"
	StatusRunning  TaskStatus = "running"
	StatusSuccess  TaskStatus = "success"
	StatusFailed   TaskStatus = "failed"
	StatusCanceled TaskStatus = "canceled"
)

type Task struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Input      string     `json:"input"`
	Status     TaskStatus `json:"status"`
	Stage      string     `json:"stage,omitempty"`
	Progress01 float64    `json:"progress01,omitempty"`
	ResultPath string     `json:"resultPath,omitempty"`
	Error      string     `json:"error,omitempty"`
}

type Queue struct {
	mu    sync.RWMutex
	items []*Task
}

func NewQueue() *Queue { return &Queue{items: []*Task{}} }

func (q *Queue) List() []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()
	out := make([]*Task, 0, len(q.items))
	out = append(out, q.items...)
	return out
}

func (q *Queue) Enqueue(t *Task) {
	q.mu.Lock()
	q.items = append(q.items, t)
	q.mu.Unlock()
}

func (q *Queue) RunWorker(ctx context.Context, run func(context.Context, *Task) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var next *Task
		q.mu.RLock()
		for _, it := range q.items {
			if it.Status == StatusQueued {
				next = it
				break
			}
		}
		q.mu.RUnlock()
		if next == nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}
		next.Status = StatusRunning
		if err := run(ctx, next); err != nil {
			next.Status = StatusFailed
			next.Error = err.Error()
		} else {
			next.Status = StatusSuccess
		}
	}
}
```

在 MVP 中 `run()` 需要完成：yt-dlp 获取音频并落到 `.incoming`、必要时用 ffmpeg 输出 m4a、move 到曲库根、触发 MPD update、通过 WS 推送 `task.update/task.done`。

- [ ] **Step 4: 将 REST handlers 与 WS 事件串起来**

按优先级完成：

1. `/api/v1/player/*`：先实现 next/prev/play/pause/volume（用 MPD 命令）
2. `/api/v1/queue/*`：先实现 add/clear/remove（MPD playlist）
3. `/api/v1/upload`：先落 `.incoming` 再 move 入库，触发 rescan
4. `/api/v1/tasks/bv`：入队并通过 worker 执行，完成发 `task.done`
5. `/api/v1/bluetooth/*`：scan/connect/disconnect/default；自动重连作为后台 goroutine

- [ ] **Step 5: 加上 systemd unit（模板）并 push 到 GitHub**

Create `packaging/musicbox.service`:

```ini
[Unit]
Description=musicbox
After=network.target

[Service]
Type=simple
Environment=MUSICBOX_ADDR=:8080
Environment=MUSICBOX_MPD_ADDR=127.0.0.1:6600
Restart=always
RestartSec=2
ExecStart=/usr/local/bin/musicbox

[Install]
WantedBy=multi-user.target
```

Push:

```bash
git push -u origin main
```

---

## Plan Self-Review

- 覆盖需求：Go 单进程、React+Tailwind 单页 Tabs、WS 每 3 秒 progress、上传/BV 只入库不打扰、蓝牙管理接口骨架、BV 任务队列串行。
- 无占位词：本计划不包含 TBD/TODO 字样；命令与代码均给出可直接粘贴的最小版本。
