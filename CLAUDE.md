# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

PowerControl 是一个基于 Go 语言的远程设备电源管理系统，支持网络唤醒(WOL)、远程关机、状态监控和米家集成。这是从原 Python 版本重构而来的 v4.0 版本，使用 Go 1.21 开发。

**核心技术栈：** Go 1.21 + Gin Web Framework + Zap Logger + YAML 配置

## 常用命令

### 开发与测试

```bash
# 本地运行
go mod download
go build -o powercontrol ./cmd/powercontrol
./powercontrol

# 运行并指定配置
WEB_PORT=7678 WEB_KEY=admin DATA_DIR=./data ./powercontrol

# 代码检查
go vet ./...
go fmt ./...

# 构建 Docker 镜像
docker build -t powercontrol:local .

# 运行容器（必须使用 host 网络）
docker run -d \
  -v ./data:/app/data \
  -e WEB_PORT=7678 \
  -e WEB_KEY=admin \
  --network host \
  --name powercontrol \
  powercontrol:local

# 查看容器日志
docker logs -f powercontrol
```

### 环境变量

- `WEB_PORT`: Web 服务端口（默认 7678）
- `WEB_KEY`: API 访问密钥（默认 admin，生产环境必须修改）
- `DATA_DIR`: 数据目录路径（默认 /app/data）
- `TZ`: 时区（默认 Asia/Shanghai）

## 核心架构

### 模块结构

```
cmd/powercontrol/     # 主入口
  main.go            # 程序入口，初始化日志、配置、服务管理器和Web服务

internal/
  config/           # 配置管理
    config.go       # YAML配置加载、保存、环境变量覆盖

  service/          # 设备服务层
    manager.go      # 服务管理器，管理所有设备服务的生命周期
    device.go       # 设备服务，每个设备独立服务(Ping检测、Bemfa连接)

  core/             # 核心功能
    network.go      # WOL魔术包发送、Ping检测、广播地址计算
    shutdown.go     # SSH/SMB关机实现

  web/              # Web服务
    server.go       # Gin服务器、路由配置、中间件
    handlers.go     # API处理函数

  bemfa/            # 巴法云集成
    client.go       # TCP协议客户端、心跳、自动重连

  logger/           # 日志系统
    logger.go       # Zap日志封装、文件轮转

  messenger/        # 消息推送
    messenger.go    # Webhook推送

web/                # 前端静态文件
  index.html        # 单页面管理界面
```

### 启动流程

1. **main.go** 加载配置和初始化日志
2. **service.Manager** 启动所有启用的设备服务
3. **web.Server** 启动 Gin HTTP 服务器
4. 每个 **DeviceService** 在独立 goroutine 中运行：
   - Ping 检测定时任务（可配置间隔）
   - Bemfa TCP 连接（如果启用）
   - 设备状态管理和同步

### 关键设计模式

**服务管理器模式：** `service.Manager` 集中管理所有设备服务的启动、停止、重启，使用 `sync.RWMutex` 保证并发安全

**独立设备服务：** 每个设备有独立的 `DeviceService` 实例，运行在独立的上下文中，使用 `context.Context` 控制生命周期

**配置分离：**
- `main.yaml` - 全局配置（Web、日志、消息）
- `device_<id>.yaml` - 每个设备独立配置文件

**环境变量优先：** 环境变量会覆盖配置文件中的 `WEB_PORT` 和 `WEB_KEY`

### 网络要求

**WOL 必须使用 host/ipvlan/macvlan 网络模式**，不支持 bridge 模式，因为网络唤醒需要发送 UDP 广播包。

### WOL 广播地址计算

使用子网掩码计算定向广播地址：`BroadcastIP = DeviceIP | (~Netmask)`

示例：
- 设备 IP: 192.168.1.100
- 子网掩码: 255.255.255.0
- 广播地址: 192.168.1.255

如果子网掩码为空或 `255.255.255.255`，则使用全局广播 `255.255.255.255`

### Bemfa 云平台集成

- **协议：** 私有 TCP 协议（非标准 MQTT），连接 `bemfa.com:8344`
- **认证：** UID（私钥）+ Topic
- **命令：** 接收 `on`（开机）和 `off`（关机）指令
- **心跳：** 每 60 秒发送 `ping\r\n`
- **重连策略：** 每日前 5 次重连延迟 5 秒，第 6 次起延迟 60 秒
- **状态同步：** 设备状态变化时自动上报到 Bemfa

### API 端点设计

所有 API 都需要 `?key=YOUR_KEY` 参数进行鉴权：

**设备控制：**
- `/wol/<device_id>?key=xxx` - 网络唤醒
- `/shutdown/<device_id>?key=xxx` - 远程关机
- `/ping/<device_id>?key=xxx` - 状态检测

**设备管理：**
- `/device/get/all-brief?key=xxx` - 获取所有设备简要信息
- `/device/get/<id>?key=xxx` - 获取设备详细配置
- `/device/new?key=xxx` - 创建设备（POST JSON）
- `/device/update/<id>?key=xxx` - 更新设备（POST JSON）
- `/device/delete/<id>?key=xxx` - 删除设备（DELETE）

**批量操作：**
- `/device/batch/wol?key=xxx` - 批量唤醒（POST `{"device_ids": [...]}`)
- `/device/batch/shutdown?key=xxx` - 批量关机
- `/device/wol/all?key=xxx` - 全部唤醒（POST）
- `/device/shutdown/all?key=xxx` - 全部关机（POST）

**服务控制：**
- `/device/start/<id>?key=xxx` - 启动设备服务
- `/device/stop/<id>?key=xxx` - 停止设备服务
- `/device/restart/<id>?key=xxx` - 重启设备服务

支持使用设备别名（alias）代替 device_id。

### 日志系统

- **Logger：** Uber Zap 高性能日志库
- **级别：** DEBUG、INFO、WARN、ERROR
- **轮转：** 使用 lumberjack，每天轮转，保留 N 天
- **多记录器：** main + manager + web + 每个设备独立日志记录器

### 配置文件结构

**主配置 (`/app/data/main.yaml`)：**
```yaml
web:
  port: 7678
  key: "admin"
log:
  level: "INFO"
  keep_days: 7
message:
  enabled: false
  webhook:
    enabled: false
    url: ""
    method: "POST"
    headers: {}
```

**设备配置 (`/app/data/device_<id>.yaml`)：**
```yaml
id: "device_12345678"
name: "设备名称"
alias: "device-alias"
ip: "192.168.1.100"
enabled: true

wol:
  enabled: true
  mac: "00-11-22-33-44-55"
  netmask: "255.255.255.255"  # 全局广播
  port: 9
  interface: "default"

shutdown:
  enabled: true
  method: "ssh"  # ssh, smb, custom
  account: "username"
  password: "password"
  command: ""
  time: 60
  timeout: 2

ping:
  enabled: true
  interval: 60

bemfa:
  enabled: false
  uid: ""      # 巴法私钥
  topic: ""

message:
  enabled: true
```

## 常见开发任务

### 添加新的关机方法

1. 在 `internal/core/shutdown.go` 的 `Shutdown()` 函数添加新方法分支
2. 实现具体关机逻辑函数（参考 `shutdownViaSSH`、`shutdownViaSMB`）
3. 在 `internal/config/config.go` 的 `ShutdownConfig.Method` 注释中添加方法名
4. 在前端 `web/index.html` 的关机方式下拉框添加选项

### 修改配置文件结构

1. 修改 `internal/config/config.go` 中对应的结构体
2. 更新 `config.copyDefaultConfig()` 创建的默认配置
3. 考虑向后兼容性，确保旧配置文件仍能正常加载
4. 更新前端表单和 JavaScript（如果涉及设备配置）

### 添加新的 API 端点

1. 在 `internal/web/server.go` 的 `setupRoutes()` 添加路由
2. 在 `internal/web/handlers.go` 实现处理函数
3. 使用统一的响应格式：`{"success": bool, "message": string, "data": ...}`
4. 记得在路由组中使用 `authMiddleware()` 进行鉴权

### 调试设备服务

查看特定设备的日志：
```bash
# 容器内日志文件位于 /app/data/logs/
docker exec powercontrol tail -f /app/data/logs/device_<设备名>.log
```

## 注意事项

- **线程安全：** `service.Manager` 的所有公开方法都使用 `sync.RWMutex` 保护
- **优雅停止：** 使用 `context.Context` 传递停止信号，所有服务都支持优雅关闭
- **配置热重载：** 更新设备配置后需调用 `/device/restart/<id>` 重启设备服务
- **WOL 魔术包格式：** 6 字节 0xFF + 16 次重复的 MAC 地址（共 102 字节）
- **SSH 关机依赖：** 需要容器内安装 `sshpass` 工具
- **SMB 关机依赖：** 需要容器内安装 `samba-client` (`net` 命令)
- **Bemfa 重连：** 网络异常时自动重连，每日前 5 次快速重连，之后降速
- **代码注释：** 保持与现有代码库语言一致（中文注释）

## 项目特殊约定

- 设备 ID 格式：`device_` + 随机字符串
- 别名用于 API 调用，可包含空格（API 调用时空格转换为 `-`）
- 配置文件命名：`main.yaml`（唯一）+ `device_<id>.yaml`（多个）
- Web 界面端口和密钥优先使用环境变量，不依赖配置文件
- Docker 镜像采用两阶段构建：builder（编译）+ alpine（运行）
- 编译时使用 `CGO_ENABLED=0` 实现静态链接，方便跨平台部署
