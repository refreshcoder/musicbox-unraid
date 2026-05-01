# Unraid 蓝牙音箱音乐盒（接口与实现文档）

目标：一个页面（React + Tailwind）完成播放控制、队列/曲库、上传、BV 下载（落成 m4a）、蓝牙连接管理（稳定自动重连 + 可诊断）。

约束：2 vCPU / 1GB RAM；后端 Go 单进程；蓝牙 USB 直通给 VM；曲库根目录 `/srv/music`（仅一个目录）。

默认行为：上传与 BV 下载完成后只入库，不自动加入队列、不自动播放。

## 1. 目录与状态存储

- 曲库根目录：`/srv/music`
- 临时目录：`/srv/music/.incoming`
  - 上传、BV 下载先落此处
  - 成功后 move 到 `/srv/music/` 根（避免半成品被扫入曲库）
- 状态目录：`/srv/music/.db`
  - sqlite（任务队列、默认音箱 MAC、重连参数、最近错误摘要）

## 2. 系统组件与职责

### 2.1 BlueZ / BlueALSA

- BlueZ：设备扫描、配对、信任、连接、断开
- BlueALSA：将“已连接蓝牙音箱”暴露为 ALSA PCM 设备，供 MPD 输出

### 2.2 MPD

- 播放器与曲库索引
- 输出设备：绑定 BlueALSA 设备（建议绑定默认音箱 MAC）

### 2.3 Go Web App（单进程）

职责：

- 静态资源：提供 React 构建产物（生产环境）
- REST API：操作型接口（播放控制、队列、曲库搜索、上传、任务、蓝牙操作）
- WebSocket：状态推送（播放状态/进度、蓝牙状态/错误、任务进度）
- BV 任务：串行队列（并发=1），调用 `yt-dlp` 与 `ffmpeg` 生成 m4a
- 自动重连：断线后对默认音箱退避重连，并记录失败阶段与原因

## 3. 后端（Go）模块划分

建议目录：

- `cmd/musicbox/`：main
- `internal/http/`：路由、handler、请求校验
- `internal/ws/`：ws hub（订阅/广播/连接管理）
- `internal/mpd/`：MPD 客户端（TCP 协议，最小命令集）
- `internal/bluetooth/`
  - `state`：读取 controller/设备状态（优先 DBus）
  - `ops`：扫描/配对/信任/连接/断开（可先封装 `bluetoothctl` 子进程，后续替换纯 DBus）
  - `reconnect`：默认设备自动重连（退避、记录错误）
- `internal/tasks/`
  - 队列与 worker（串行）
  - BV 下载与落库逻辑（incoming → move → rescan）
- `internal/store/`：sqlite
- `web/`：React 项目（构建后产物 `web/dist`）

## 4. REST API（最小完备集合）

统一约定：

- JSON 请求/响应（上传除外）
- 错误响应：
  - HTTP status 使用 4xx/5xx
  - body：`{ "error": { "code": "STRING", "message": "STRING" } }`

### 4.1 汇总状态

- `GET /api/v1/status`
  - 返回：player / bluetooth / tasks 的汇总快照

### 4.2 播放器（MPD）

- `POST /api/v1/player/play`
- `POST /api/v1/player/pause`
- `POST /api/v1/player/toggle`
- `POST /api/v1/player/next`
- `POST /api/v1/player/prev`
- `POST /api/v1/player/seek`：`{ "positionMs": 123000 }`
- `POST /api/v1/player/volume`：`{ "volume": 0-100 }`

### 4.3 队列

- `GET /api/v1/queue`
- `POST /api/v1/queue/clear`
- `POST /api/v1/queue/add`：`{ "path": "relative/to/music_root.m4a" }`
- `POST /api/v1/queue/remove`：`{ "pos": 12 }`
- 可选（体验更好）：`POST /api/v1/queue/play`：`{ "pos": 12 }`

### 4.4 曲库

- `GET /api/v1/library/search?q=...`
- `POST /api/v1/library/rescan`

### 4.5 上传

- `POST /api/v1/upload`（multipart/form-data）
  - 字段：`file`
  - 行为：保存到 `.incoming` → 完成后 move 入库 → rescan → 仅提示“已入库”

### 4.6 BV 任务（只入库不打扰）

- `POST /api/v1/tasks/bv`：`{ "bv": "BVxxxx" }`
- `GET /api/v1/tasks`
- `POST /api/v1/tasks/{id}/cancel`

任务状态枚举：

- `queued | running | success | failed | canceled`

阶段枚举（`running` 时可用）：

- `download | extract_audio | transcode | move | rescan`

### 4.7 蓝牙

- `GET /api/v1/bluetooth/status`
- `POST /api/v1/bluetooth/scan/start`
- `POST /api/v1/bluetooth/scan/stop`
- `GET /api/v1/bluetooth/devices`
- `POST /api/v1/bluetooth/devices/{mac}/pair`
- `POST /api/v1/bluetooth/devices/{mac}/trust`
- `POST /api/v1/bluetooth/devices/{mac}/connect`
- `POST /api/v1/bluetooth/devices/{mac}/disconnect`
- `DELETE /api/v1/bluetooth/devices/{mac}`
- `PUT /api/v1/bluetooth/default`：`{ "mac": "AA:BB:..." }`
- `DELETE /api/v1/bluetooth/default`

## 5. WebSocket（/ws）事件契约

连接：

- `GET /ws`

统一格式：

```json
{
  "type": "player.progress",
  "ts": 1710000000,
  "data": {}
}
```

### 5.1 Player

- `player.state`：状态变化推送（播放/暂停/切歌/音量/模式变化）
- `player.queue`：队列变化推送
- `player.progress`：固定每 3 秒推送

`player.progress` data：

```json
{
  "positionMs": 123000,
  "durationMs": 245000,
  "bitrateKbps": 192
}
```

### 5.2 Bluetooth

- `bluetooth.state`：连接/默认设备/扫描状态变化
- `bluetooth.devices`：扫描列表更新（扫描期间建议 1–2 秒节流推送）
- `bluetooth.error`：可诊断错误（必须带 stage）

`bluetooth.error` stage 枚举：

- `scan | pair | trust | connect | a2dp | bluealsa | mpd_output | reconnect`

### 5.3 Tasks / Upload

- `task.list`：初次连接或大变更时推送
- `task.update`：进度/状态变化
- `task.done`：终态（success/failed/canceled）
- `upload.done`：上传成功（只入库）
- `upload.failed`：上传失败

## 6. BV → m4a 任务规范

输入：

- `BVxxxx`

输出：

- m4a 文件落在曲库根 `/srv/music`
- `resultPath` 为相对曲库根的相对路径（供前端加入队列）

执行原则：

- worker 串行（并发=1）
- 优先直取音频流（可直接得到 m4a/aac 更省 CPU）
- 需要转码时用 `ffmpeg` 输出 m4a

默认行为：

- 成功后仅入库与 rescan
- 不自动加入队列、不自动播放

## 7. 自动重连（稳定优先 + 可诊断）

配置项（建议存 store，可在 UI 设置页暴露）：

- `defaultMac`
- `reconnectEnabled`（默认 true）
- `backoffSeconds`（默认 `[5,10,20,40,60]` 循环）

行为：

- 断开或开机时：若 defaultMac 存在且未连接，触发重连流程
- 每次失败必须记录：
  - `stage`（见 bluetooth.error stage）
  - `reason`（尽量包含底层错误文本）
  - `mac`
  - `ts`

## 8. React + Tailwind 前端结构

### 8.1 工程与构建

- React + TS + Vite
- Tailwind + PostCSS
- 生产：构建到 `web/dist`，由 Go 静态服务提供
- 开发：Vite proxy `/api` 与 `/ws` 到 Go

### 8.2 页面结构（单页 Tabs + 响应式）

- `PlayerBar`：顶部常驻
- Tabs：
  - 播放
  - 队列
  - 曲库
  - 上传 & BV
  - 蓝牙
  - 诊断

移动端：

- Tabs 变底部导航
- 列表单列 + Modal/Drawer 展示详情

### 8.3 前端状态管理建议

- Zustand：`playerStore / queueStore / bluetoothStore / tasksStore / uiStore`
- WebSocket 收到事件后直接更新 store
- `player.progress` 每 3 秒更新一次，UI 可用本地计时做轻微平滑（可选）

## 9. 生产部署（前后端打包策略）

推荐：Go 单二进制（含前端静态资源）：

- `web` 构建产物打进二进制（`go:embed`）
- systemd 管理
- 所有访问从 `:8080` 进入（静态 + API + WS）

