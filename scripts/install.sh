#!/usr/bin/env bash
set -euo pipefail

MUSICBOX_REPO="${MUSICBOX_REPO:-https://github.com/refreshcoder/musicbox-unraid.git}"
MUSICBOX_REF="${MUSICBOX_REF:-main}"
MUSICBOX_DIR="${MUSICBOX_DIR:-/opt/musicbox}"

MUSICBOX_ADDR="${MUSICBOX_ADDR:-:8080}"
MUSICBOX_MPD_ADDR="${MUSICBOX_MPD_ADDR:-127.0.0.1:6600}"
MUSICBOX_MUSIC_DIR="${MUSICBOX_MUSIC_DIR:-/srv/music}"
MUSICBOX_STATIC_DIR="${MUSICBOX_STATIC_DIR:-/opt/musicbox/web/dist}"

GO_VERSION="${GO_VERSION:-1.22.13}"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"
GO_DIR="/usr/local/go"

require_root() {
  if [ "${EUID:-$(id -u)}" -ne 0 ]; then
    echo "Please run as root (use sudo)." >&2
    exit 1
  fi
}

have_cmd() {
  command -v "$1" >/dev/null 2>&1
}

go_version_ok() {
  if ! have_cmd go; then
    return 1
  fi
  local v
  v="$(go version | awk '{print $3}' | sed 's/^go//')"
  local major minor
  major="$(echo "$v" | cut -d. -f1)"
  minor="$(echo "$v" | cut -d. -f2)"
  if [ -z "$major" ] || [ -z "$minor" ]; then
    return 1
  fi
  if [ "$major" -gt 1 ]; then
    return 0
  fi
  if [ "$major" -lt 1 ]; then
    return 1
  fi
  if [ "$minor" -ge 22 ]; then
    return 0
  fi
  return 1
}

install_apt_deps() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y --no-install-recommends \
    ca-certificates curl git \
    nodejs npm \
    ffmpeg yt-dlp \
    mpd mpc \
    bluez bluealsa \
    build-essential
}

install_go() {
  if go_version_ok; then
    return 0
  fi

  rm -rf "${GO_DIR}"
  curl -fsSL "${GO_URL}" -o "/tmp/${GO_TARBALL}"
  tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"
  rm -f "/tmp/${GO_TARBALL}"

  cat >/etc/profile.d/go.sh <<'EOF'
export PATH="/usr/local/go/bin:$PATH"
EOF

  export PATH="/usr/local/go/bin:${PATH}"
  if ! go_version_ok; then
    echo "Go install failed or version too old." >&2
    exit 1
  fi
}

ensure_dirs() {
  mkdir -p "${MUSICBOX_DIR}"
  mkdir -p "${MUSICBOX_MUSIC_DIR}"
  mkdir -p "${MUSICBOX_MUSIC_DIR}/.incoming"
}

sync_repo() {
  if [ -d "${MUSICBOX_DIR}/.git" ]; then
    git -C "${MUSICBOX_DIR}" fetch --all --prune
    git -C "${MUSICBOX_DIR}" checkout "${MUSICBOX_REF}"
    git -C "${MUSICBOX_DIR}" pull --ff-only
  else
    rm -rf "${MUSICBOX_DIR}"
    git clone --branch "${MUSICBOX_REF}" "${MUSICBOX_REPO}" "${MUSICBOX_DIR}"
  fi
}

build_web() {
  (cd "${MUSICBOX_DIR}/web" && npm ci && npm run build)
}

build_backend() {
  (cd "${MUSICBOX_DIR}" && /usr/local/go/bin/go build -o /usr/local/bin/musicbox ./cmd/musicbox)
}

write_env_file() {
  cat >/etc/musicbox.env <<EOF
MUSICBOX_ADDR=${MUSICBOX_ADDR}
MUSICBOX_MPD_ADDR=${MUSICBOX_MPD_ADDR}
MUSICBOX_MUSIC_DIR=${MUSICBOX_MUSIC_DIR}
MUSICBOX_STATIC_DIR=${MUSICBOX_STATIC_DIR}
EOF
  chmod 0644 /etc/musicbox.env
}

write_service() {
  cat >/etc/systemd/system/musicbox.service <<'EOF'
[Unit]
Description=musicbox
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/musicbox.env
Restart=always
RestartSec=2
ExecStart=/usr/local/bin/musicbox

[Install]
WantedBy=multi-user.target
EOF
}

enable_service() {
  systemctl daemon-reload
  systemctl enable --now musicbox
}

health_check() {
  for _ in 1 2 3 4 5; do
    if curl -fsS "http://127.0.0.1:8080/api/v1/health" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

main() {
  require_root
  install_apt_deps
  install_go
  ensure_dirs
  sync_repo
  build_web
  build_backend
  write_env_file
  write_service
  enable_service

  if health_check; then
    echo "OK"
    echo "UI: http://<vm-ip>:8080/"
    echo "Logs: journalctl -u musicbox -f"
    exit 0
  fi

  echo "Service started but health check failed. Check logs: journalctl -u musicbox -n 200 --no-pager" >&2
  exit 1
}

main "$@"

