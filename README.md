# PowerControl

> 🚀 轻量级远程设备电源管理系统 - Go 重构版

![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/iLay1678/PowerControl-go/docker-build.yml)
![License](https://img.shields.io/github/license/iLay1678/PowerControl-go)
![Go Version](https://img.shields.io/github/go-mod/go-version/iLay1678/PowerControl-go)
![GitHub Stars](https://img.shields.io/github/stars/iLay1678/PowerControl-go)

**基于 [viklion/PowerControl](https://github.com/viklion/PowerControl) 原 Python 版本重构**
本项目使用 Go 语言重写，大幅提升性能并降低资源占用，同时保持与原版本的完全兼容。

## ✨ 特性亮点


### 核心功能

- ✅ **网络唤醒** (WOL) - 远程唤醒
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
