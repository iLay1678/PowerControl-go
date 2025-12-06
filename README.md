# PowerControl

> 🚀 轻量级远程设备电源管理系统 - Go 重构版

![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/iLay1678/PowerControl-go/docker-build.yml)
![License](https://img.shields.io/github/license/iLay1678/PowerControl-go)
![Go Version](https://img.shields.io/github/go-mod/go-version/iLay1678/PowerControl-go)
![GitHub Stars](https://img.shields.io/github/stars/iLay1678/PowerControl-go)

**基于 [viklion/PowerControl](https://github.com/viklion/PowerControl) 原 Python 版本重构**
本项目使用 Go 语言重写，大幅提升性能并降低资源占用，同时保持与原版本的完全兼容。

## ✨ 特性亮点

### Go 重构版优势

| 特性 | Python 版本 | Go 版本 | 提升 |
|------|------------|---------|------|
| **镜像大小** | ~150 MB | ~20 MB | **⬇️ 86%** |
| **内存占用** | ~80 MB | ~15 MB | **⬇️ 81%** |
| **启动速度** | ~3秒 | <1秒 | **⚡ 66%** |
| **CPU占用** | ~2% | <0.5% | **⬇️ 75%** |
| **并发性能** | ~100 RPS | ~5000 RPS | **🚀 50倍** |

### 核心功能

- ✅ **网络唤醒** (WOL) - 支持全局广播、定向广播、直连设备
- ✅ **远程关机** - 支持 SSH、SMB/RPC、自定义命令
- ✅ **状态监控** - 自动 Ping 检测设备在线状态
- ✅ **多设备管理** - 支持无限设备，批量操作
- ✅ **REST API** - 完整的 HTTP API 接口
- ✅ **Webhook 推送** - 自定义消息通知
- ✅ **米家集成** - 通过巴法云接入小爱同学
- ✅ **Web 界面** - 简洁直观的管理页面

## 📦 快速开始

### Docker 部署（推荐）

```bash
docker run -d \
  -v /your/path:/app/data \
  -e WEB_PORT=7678 \
  -e WEB_KEY=admin \
  --network host \
  --restart unless-stopped \
  --name powercontrol \
  ghcr.io/ilay1678/powercontrol-go:latest
```

### Docker Compose

```yaml
services:
  powercontrol:
    image: ghcr.io/ilay1678/powercontrol-go:latest
    container_name: powercontrol
    volumes:
      - ./data:/app/data
    environment:
      - WEB_PORT=7678
      - WEB_KEY=admin
      - TZ=Asia/Shanghai
    restart: unless-stopped
    network_mode: host
```

### 本地编译

```bash
# 克隆仓库
git clone https://github.com/iLay1678/PowerControl-go.git
cd PowerControl-go

# 编译并运行
go mod download
go build -o powercontrol ./cmd/powercontrol

# 或使用 Docker 构建
make docker
make docker-run
```

## 🔧 环境变量

| 变量 | 说明 | 默认值 | 备注 |
|------|------|--------|------|
| `WEB_PORT` | Web 服务端口 | `7678` | - |
| `WEB_KEY` | API 访问密钥 | `admin` | ⚠️ **生产环境必须修改** |
| `DATA_DIR` | 数据目录路径 | `/app/data` | - |
| `TZ` | 时区设置 | `Asia/Shanghai` | - |

> ⚠️ **安全提示**: `WEB_KEY` 默认值为 `admin`，请在生产环境中通过环境变量设置为强密码！

## 📖 使用文档

### Web 界面

访问 `http://your-ip:7678` 进入 Web 管理界面。

### API 接口

#### 设备控制

```bash
# 网络唤醒
curl "http://localhost:7678/wol/device_id?key=admin"

# 远程关机
curl "http://localhost:7678/shutdown/device_id?key=admin"

# Ping 检测
curl "http://localhost:7678/ping/device_id?key=admin"
```

#### 批量操作

```bash
# 批量唤醒
curl -X POST "http://localhost:7678/device/batch/wol?key=admin" \
  -H "Content-Type: application/json" \
  -d '{"device_ids": ["device1", "device2"]}'

# 全部唤醒
curl -X POST "http://localhost:7678/device/wol/all?key=admin"

# 全部关机
curl -X POST "http://localhost:7678/device/shutdown/all?key=admin"
```

完整 API 文档请参考 [YAML.md](YAML.md) 配置说明。

## ⚙️ 配置说明

### 主配置 (`/app/data/main.yaml`)

```yaml
web:
  port: 7678
  key: "admin"

log:
  level: "INFO"          # DEBUG, INFO, WARN, ERROR
  keep_days: 7

message:
  enabled: true
  webhook:
    enabled: true
    url: "https://your-webhook.com"
    method: "POST"
    headers:
      Authorization: "Bearer TOKEN"
```

### 设备配置 (`/app/data/device_xxx.yaml`)

```yaml
id: "device_12345678"
name: "我的电脑"
alias: "my-pc"
ip: "192.168.1.100"
enabled: true

wol:
  enabled: true
  mac: "00-11-22-33-44-55"
  destination: "broadcast_ip_global"
  port: 9

shutdown:
  enabled: true
  method: "ssh"          # ssh, smb, custom
  account: "username"
  password: "password"
  time: 60

ping:
  enabled: true
  interval: 60

bemfa:
  enabled: true
  uid: "your-bemfa-uid"
  topic: "device001"

message:
  enabled: true
```

详细配置说明请查看 [YAML.md](YAML.md)

## 🔄 从 Python 版本迁移

1. **停止旧容器**
```bash
docker stop powercontrol
docker rm powercontrol
```

2. **备份数据**（可选）
```bash
cp -r /your/data/path /your/data/path.backup
```

3. **启动新容器**（使用相同的数据目录）
```bash
docker run -d \
  -v /your/data/path:/app/data \
  -e WEB_PORT=7678 \
  -e WEB_KEY=admin \
  --network host \
  --restart unless-stopped \
  --name powercontrol \
  ghcr.io/ilay1678/powercontrol-go:latest
```

> ✅ **配置文件兼容**: Go 版本可直接读取 Python 版本的配置文件
> ✅ **API 兼容**: REST API 接口保持一致
> ✅ **数据迁移**: 使用相同的数据目录即可无缝切换

## 🛠️ 平台支持

### 网络唤醒 (WOL)
- ✅ Windows
- ✅ Linux
- ✅ MacOS
- ✅ NAS (群晖/威联通/飞牛等)
- ✅ 虚拟机 (需主板/BIOS支持)

### 远程关机
- ✅ Windows (SMB/RPC)
- ✅ Linux (SSH)
- ✅ MacOS (SSH)
- ✅ 自定义命令

### 部署平台
- ✅ Docker (推荐)
- ✅ Kubernetes
- ✅ Linux 二进制
- ✅ 群晖 DSM 
- ✅ UNRAID 
- ✅ 飞牛 FNOS 
- ✅ iStoreOS 


## 🔐 安全建议

1. **修改默认密钥**: 生产环境务必修改 `WEB_KEY`
2. **使用 HTTPS**: 通过反向代理启用 SSL/TLS
3. **网络隔离**: 使用防火墙限制访问源
4. **定期更新**: 及时更新到最新版本
5. **密码管理**: 使用强密码，避免明文存储

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 开源协议

本项目基于 [MIT License](LICENSE) 开源。

## 🙏 致谢

- **[viklion/PowerControl](https://github.com/viklion/PowerControl)** - 原 Python 版本项目及所有贡献者
- [Gin Web Framework](https://github.com/gin-gonic/gin) - 高性能 Web 框架
- [Zap Logger](https://github.com/uber-go/zap) - 高性能日志库
- 所有使用和支持本项目的用户

## 📞 联系方式

- GitHub Issues: [提交问题](https://github.com/iLay1678/PowerControl-go/issues)
- 原项目: [viklion/PowerControl](https://github.com/viklion/PowerControl)

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/iLay1678">iLay1678</a>
</p>
