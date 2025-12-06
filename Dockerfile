# 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /build

# 安装编译依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o powercontrol ./cmd/powercontrol

# 运行阶段
FROM alpine:latest

# 信息
LABEL name="PowerControl-Go" \
      maintainer="viklion" \
      github="https://github.com/viklion/PowerControl" \
      version="4.0.0"

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sshpass \
    samba-client \
    && rm -rf /var/cache/apk/*

# 创建工作目录
WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /build/powercontrol /app/

# 创建数据目录
RUN mkdir -p /app/data /app/default

# 设置时区
ENV TZ=Asia/Shanghai

# 版本号
ENV VERSION=4.0.0

# 默认环境变量
ENV WEB_PORT=7678
ENV WEB_KEY=admin
ENV DATA_DIR=/app/data

# 暴露端口
EXPOSE 7678

# 启动命令
CMD ["/app/powercontrol"]
