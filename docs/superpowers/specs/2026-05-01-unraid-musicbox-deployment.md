# Unraid 6.12.13 蓝牙音箱音乐盒（VM 部署文档）

目标：在 Unraid 6.12.13 上部署一个轻量 Debian VM，USB 蓝牙小尾巴直通给 VM；VM 内通过 BlueZ + BlueALSA + MPD 输出到蓝牙音箱，并运行一个 Web App（Go 后端 + React 前端）提供一页式控制台。

约束：2 vCPU / 1GB RAM（建议加 swap）/ 单目录曲库 `/srv/music`（挂载 Unraid share）。

## 1. 前置条件

- Unraid 已启用虚拟化：BIOS 开启 VT-x/AMD-V；Unraid 侧启用 VM Manager
- USB 蓝牙小尾巴插入 Unraid 主机
- 规划一个 Unraid share 用于曲库（示例：`MusicBox`）

## 2. Unraid 创建 Debian VM

建议参数（通用、优先稳定）：

- OS：Debian 12（netinst ISO）
- Machine：Q35
- BIOS：OVMF
- CPU：2 vCPU
- RAM：1024 MB
- Disk：10–20 GB（VirtIO）
- Network：VirtIO（桥接到局域网）

网络建议：

- 给 VM 配置固定 IP（DHCP 静态绑定或 VM 内静态地址），便于反代与访问

## 3. USB 蓝牙直通

目标：让蓝牙栈完全运行在 VM 内，避开 Unraid 宿主蓝牙兼容性问题。

操作要点：

- 在 VM 的 USB 设备分配中选择蓝牙小尾巴直通给 VM
- 尽量使用“按设备身份固定绑定”（VID:PID 或设备路径固定），避免重启后设备枚举变化导致直通失效

验证点（VM 内）：

- `lsusb` 能看到蓝牙适配器
- `bluetoothctl list` 能看到 controller

## 4. Debian 12 安装建议

安装时：

- 仅安装最小系统（无桌面环境）
- 开启 SSH（便于远程管理）
- 创建普通用户并加入 sudo 组

安装后建议：

- 关闭不必要服务
- 配置时区与时间同步（systemd-timesyncd / chrony 均可）

## 5. 配置 swap（强烈建议）

目的：BV 下载/转码会出现短时内存峰值，swap 可降低 OOM 风险。

建议：

- swap 1–2GB
- 使用 swapfile（系统盘即可）

## 6. 挂载 Unraid 曲库 share 到 `/srv/music`

曲库根目录要求：只用一个曲库目录 `/srv/music`，但允许创建隐藏子目录做中间态与状态库：

- `/srv/music/.incoming`：上传与 BV 下载临时落点（完成后 move 到曲库根）
- `/srv/music/.db`：状态存储（sqlite 等）

挂载方式：

- 优先 NFS（更轻、更适合 Linux）
- 也可 SMB/CIFS（取决于你现有 share 配置）

验证点：

- VM 重启后自动挂载成功
- `/srv/music` 可读写（权限与 UID/GID 一致）

## 7. 安装与启用蓝牙音频链路（BlueZ + BlueALSA）

目标链路：蓝牙音箱（A2DP sink）⇐ VM（A2DP source）→ BlueALSA（ALSA PCM）。

组件：

- BlueZ：`bluetoothd`
- BlueALSA：`bluealsa` 与 `bluealsa-aplay`（如需要测试）

验证点：

- `bluetoothctl` 能扫描、配对、信任、连接音箱
- 连接后 ALSA 设备列表中能看到 `bluealsa:` 对应设备（具体名称依系统而定）

## 8. 安装与配置 MPD（输出到 BlueALSA）

目标：

- `music_directory` 指向 `/srv/music`
- 输出设备绑定到 BlueALSA（建议绑定默认音箱 MAC，避免多设备时漂移）

验证点：

- `mpc update` 能扫描曲库
- `mpc add ... && mpc play` 后蓝牙音箱出声
- 音量与暂停等基本控制正常

## 9. 部署 Web App（Go 单进程）

服务职责：

- 提供静态前端（React 构建产物）
- 提供 REST API + WebSocket
- 管理蓝牙、MPD、上传、任务队列（BV → m4a）

建议运行方式：

- systemd service（开机自启、崩溃自动重启）

监听端口：

- `:8080`（HTTP + WebSocket）

反代策略（你已有反代）：

- 反代到 VM `http://<vm-ip>:8080`

## 10. 验收清单（逐项）

- USB 直通：VM 内可识别蓝牙适配器
- 蓝牙：可扫描/配对/信任/连接；断线后可自动重连默认音箱；UI 显示最近失败原因
- 音频：MPD 播放文件稳定出声；seek/音量/暂停正常
- 上传：上传成功后只入库，不自动播放；曲库刷新后可在曲库中搜索到
- BV：输入 BV 后生成 m4a，成功入库，不自动播放；失败时 UI 显示错误原因

