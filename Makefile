.PHONY: build run test clean docker

# 变量定义
VERSION ?= 4.0.0
BINARY_NAME = powercontrol
DOCKER_IMAGE = viklion/powercontrol-go

# 构建
build:
	@echo "正在编译..."
	@go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/powercontrol
	@echo "编译完成: $(BINARY_NAME)"

# 运行
run: build
	@echo "启动程序..."
	@./$(BINARY_NAME)

# 测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 清理
clean:
	@echo "清理编译文件..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/

# 构建 Docker 镜像
docker:
	@echo "构建 Docker 镜像..."
	@docker build -f Dockerfile.go -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "构建完成: $(DOCKER_IMAGE):$(VERSION)"

# 运行 Docker 容器
docker-run:
	@echo "启动 Docker 容器..."
	@docker run -d \
		-v $(PWD)/data:/app/data \
		-e WEB_PORT=7678 \
		-e WEB_KEY=admin \
		--network host \
		--name powercontrol \
		$(DOCKER_IMAGE):latest

# 停止 Docker 容器
docker-stop:
	@docker stop powercontrol || true
	@docker rm powercontrol || true

# 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy

# 代码格式化
fmt:
	@echo "格式化代码..."
	@go fmt ./...

# 代码检查
lint:
	@echo "代码检查..."
	@golangci-lint run ./...

# 帮助
help:
	@echo "PowerControl 构建命令:"
	@echo "  make build       - 编译程序"
	@echo "  make run         - 运行程序"
	@echo "  make test        - 运行测试"
	@echo "  make clean       - 清理编译文件"
	@echo "  make docker      - 构建 Docker 镜像"
	@echo "  make docker-run  - 运行 Docker 容器"
	@echo "  make docker-stop - 停止 Docker 容器"
	@echo "  make deps        - 下载依赖"
	@echo "  make fmt         - 格式化代码"
	@echo "  make lint        - 代码检查"
