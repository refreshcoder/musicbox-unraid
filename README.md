# musicbox-unraid

Unraid VM music box: BlueZ + BlueALSA + MPD + Go(Web) + React(Tailwind).

## Unraid VM（简版要点）

- 在 Unraid 创建 Debian 12 最小化 VM（开启 SSH）
- USB 蓝牙适配器直通给 VM（VM 内 `lsusb`/`bluetoothctl list` 可见）
- `/srv/music` 已存在且可写（建议挂载 Unraid share 到该目录）

详细步骤见部署文档。

## Debian VM 一键安装（从 GitHub 拉取并安装）

在 Debian 12 VM 内执行：

```bash
curl -fsSL https://raw.githubusercontent.com/refreshcoder/musicbox-unraid/main/scripts/install.sh | sudo bash
```

可选自定义（示例：改端口、改曲库路径）：

```bash
curl -fsSL https://raw.githubusercontent.com/refreshcoder/musicbox-unraid/main/scripts/install.sh | sudo \
  MUSICBOX_ADDR=:18080 \
  MUSICBOX_MUSIC_DIR=/srv/music \
  bash
```

安装完成后：

- Web UI：`http://<vm-ip>:8080/`
- 查看日志：`journalctl -u musicbox -f`
- 修改配置：编辑 `/etc/musicbox.env` 后执行 `systemctl restart musicbox`
- 升级：重复执行一遍 install.sh（会 `git pull` 并重新构建）

## Docs

- Deployment: docs/superpowers/specs/2026-05-01-unraid-musicbox-deployment.md
- Implementation: docs/superpowers/specs/2026-05-01-unraid-musicbox-implementation.md
