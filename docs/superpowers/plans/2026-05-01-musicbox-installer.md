# MusicBox VM Installer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供一个可在 Debian 12 VM 上运行的“一键安装脚本”，自动拉取 `refreshcoder/musicbox-unraid` 并安装所有依赖、构建前后端、配置 systemd 服务；同时更新仓库 README，包含 VM 安装要点与一键安装流程。

**Architecture:** `scripts/install.sh` 负责：安装 apt 依赖 → 安装 Go toolchain(>=1.22) → clone/pull 到 `/opt/musicbox` → 前端 `npm ci && npm run build` → 后端 `go build` → 写入 `/etc/systemd/system/musicbox.service` + `/etc/musicbox.env` → enable/start → 健康检查。README 提供可复制粘贴的一键命令与必要前置条件（USB 蓝牙直通、/srv/music 已挂载可写、仅监听 8080）。

**Tech Stack:** Bash, apt, systemd, Go, Node.js/npm.

---

## File Structure

- Create: `scripts/install.sh`
- Create: `packaging/musicbox.service`（作为模板/参考）
- Modify: `README.md`

---

### Task 1: 安装脚本（scripts/install.sh）

**Files:**
- Create: `scripts/install.sh`

- [ ] **Step 1: 设计脚本参数与默认值**

默认值（允许环境变量覆盖）：

- `MUSICBOX_REPO=https://github.com/refreshcoder/musicbox-unraid.git`
- `MUSICBOX_REF=main`
- `MUSICBOX_DIR=/opt/musicbox`
- `MUSICBOX_ADDR=:8080`
- `MUSICBOX_MPD_ADDR=127.0.0.1:6600`
- `MUSICBOX_MUSIC_DIR=/srv/music`
- `MUSICBOX_STATIC_DIR=/opt/musicbox/web/dist`
- `GO_VERSION=1.22.13`（或你确认的具体版本）

- [ ] **Step 2: root/sudo 检测与基础依赖安装**

Run（脚本内部）：

```bash
apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates curl git \
  nodejs npm \
  ffmpeg yt-dlp \
  mpd mpc \
  bluez bluealsa \
  build-essential
```

- [ ] **Step 3: 安装 Go toolchain（>=1.22）**

脚本内部行为：

- 若 `go version` >= 1.22 则跳过
- 否则下载 `https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz`
- 解压到 `/usr/local/go`（覆盖旧版本）
- 写入 `/etc/profile.d/go.sh`（仅 PATH）

- [ ] **Step 4: clone/pull 仓库并构建**

脚本内部行为：

- 若 `/opt/musicbox/.git` 存在：`git fetch --all --prune && git checkout $MUSICBOX_REF && git pull`
- 否则：`git clone --branch $MUSICBOX_REF $MUSICBOX_REPO $MUSICBOX_DIR`
- 前端：`cd $MUSICBOX_DIR/web && npm ci && npm run build`
- 后端：`cd $MUSICBOX_DIR && /usr/local/go/bin/go build -o /usr/local/bin/musicbox ./cmd/musicbox`

- [ ] **Step 5: systemd + env 文件**

写入：

- `/etc/musicbox.env`：包含上述 MUSICBOX_* 环境变量
- `/etc/systemd/system/musicbox.service`：引用 `EnvironmentFile=/etc/musicbox.env`

启动：

```bash
systemctl daemon-reload
systemctl enable --now musicbox
```

- [ ] **Step 6: 健康检查与输出下一步提示**

```bash
curl -fsS http://127.0.0.1:8080/api/v1/health
```

输出提示：

- UI 地址：`http://<vm-ip>:8080/`
- `journalctl -u musicbox -f`
- MPD/蓝牙音频链路属于运行环境配置项（参考 docs 部署文档）

---

### Task 2: packaging/musicbox.service 模板

**Files:**
- Create: `packaging/musicbox.service`

- [ ] **Step 1: 添加与 install.sh 一致的 systemd unit**

Unit 关键点：

- `Restart=always`
- `EnvironmentFile=/etc/musicbox.env`
- `ExecStart=/usr/local/bin/musicbox`

---

### Task 3: 更新 README（VM 安装要点 + 一键安装）

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 增加“Unraid VM 简版安装要点”**

包含：

- Debian 12 最小化安装 + SSH
- USB 蓝牙直通
- `/srv/music` 已挂载可写

- [ ] **Step 2: 增加“一键安装”**

提供可复制粘贴命令：

```bash
curl -fsSL https://raw.githubusercontent.com/refreshcoder/musicbox-unraid/main/scripts/install.sh | sudo bash
```

并给出：

- 自定义环境变量示例（端口、music dir）
- 升级方式（重复执行 install.sh）

---

### Task 4: 验证、提交与推送

- [ ] **Step 1: 静态检查**

Run:

```bash
bash -n scripts/install.sh
```

- [ ] **Step 2: 构建验证**

Run:

```bash
go test ./...
(cd web && npm run build)
```

- [ ] **Step 3: 提交并 push**

Run:

```bash
git add scripts packaging README.md
git commit -m "feat: add vm installer script"
env -u GITHUB_TOKEN git push
```

