# vman Makefile

# 项目信息
PROJECT_NAME := vman
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关变量
GO := go
GOFMT := gofmt
GOLINT := golangci-lint
GOTEST := $(GO) test

# 构建目录
BUILD_DIR := build
DIST_DIR := dist

# 安装相关变量
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
USER_HOME := $(shell echo $$HOME)

# 根据操作系统确定安装目录
ifeq ($(UNAME_S),Darwin)
	# macOS
	INSTALL_DIR := $(USER_HOME)/.local/bin
	CONFIG_DIR := $(USER_HOME)/Library/Application Support/vman
else ifeq ($(UNAME_S),Linux)
	# Linux
	INSTALL_DIR := $(USER_HOME)/.local/bin
	CONFIG_DIR := $(USER_HOME)/.config/vman
else
	# Windows (假设在Git Bash或类似环境中)
	INSTALL_DIR := $(USER_HOME)/bin
	CONFIG_DIR := $(USER_HOME)/AppData/Local/vman
endif

# 目标平台
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# 默认目标
.PHONY: all
all: clean fmt lint test build

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@$(GOFMT) -s -w .

# 代码检查
.PHONY: lint
lint:
	@echo "执行代码检查..."
	@$(GOLINT) run

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...

# 测试覆盖率
.PHONY: coverage
coverage: test
	@echo "生成测试覆盖率报告..."
	@$(GO) tool cover -html=coverage.out -o coverage.html

# 构建本地版本
.PHONY: build
build:
	@echo "构建 $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)" \
		-o $(BUILD_DIR)/$(PROJECT_NAME) ./cmd/vman

# 跨平台构建
.PHONY: build-all
build-all: clean
	@echo "跨平台构建 $(PROJECT_NAME)..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GO) build -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)" \
			-o $(DIST_DIR)/$(PROJECT_NAME)-$${platform%/*}-$${platform#*/}$(if $(findstring windows,$$platform),.exe,) \
			./cmd/vman; \
	done

# 安装到用户目录
.PHONY: install
install: build
	@echo "安装 $(PROJECT_NAME) 到用户目录..."
	@echo "操作系统: $(UNAME_S)"
	@echo "安装目录: $(INSTALL_DIR)"
	@echo "配置目录: $(CONFIG_DIR)"
	@mkdir -p $(INSTALL_DIR)
	@mkdir -p $(CONFIG_DIR)
	@cp $(BUILD_DIR)/$(PROJECT_NAME) $(INSTALL_DIR)/
	@chmod +x $(INSTALL_DIR)/$(PROJECT_NAME)
	@echo "✅ $(PROJECT_NAME) 已成功安装到 $(INSTALL_DIR)"
	@echo "✅ 配置目录已创建: $(CONFIG_DIR)"
	@echo ""
	@echo "请确保 $(INSTALL_DIR) 在您的 PATH 中:"
ifeq ($(UNAME_S),Darwin)
	@echo "  echo 'export PATH=\"$(INSTALL_DIR):\$$PATH\"' >> ~/.zshrc"
	@echo "  source ~/.zshrc"
else ifeq ($(UNAME_S),Linux)
	@echo "  echo 'export PATH=\"$(INSTALL_DIR):\$$PATH\"' >> ~/.bashrc"
	@echo "  source ~/.bashrc"
else
	@echo "  请手动将 $(INSTALL_DIR) 添加到您的 PATH 环境变量中"
endif
	@echo ""
	@echo "初始化 vman:"
	@echo "  vman init"

# 安装到系统目录（需要sudo）
.PHONY: install-system
install-system: build
	@echo "安装 $(PROJECT_NAME) 到系统目录..."
ifeq ($(UNAME_S),Darwin)
	@sudo cp $(BUILD_DIR)/$(PROJECT_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(PROJECT_NAME)
	@echo "✅ $(PROJECT_NAME) 已安装到 /usr/local/bin/"
else ifeq ($(UNAME_S),Linux)
	@sudo cp $(BUILD_DIR)/$(PROJECT_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(PROJECT_NAME)
	@echo "✅ $(PROJECT_NAME) 已安装到 /usr/local/bin/"
else
	@echo "请手动将 $(BUILD_DIR)/$(PROJECT_NAME) 复制到系统 PATH 目录中"
endif

# 卸载
.PHONY: uninstall
uninstall:
	@echo "卸载 $(PROJECT_NAME)..."
	@rm -f $(INSTALL_DIR)/$(PROJECT_NAME)
	@rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "⚠️  请手动删除配置目录（如果不再需要）: $(CONFIG_DIR)"
	@echo "✅ $(PROJECT_NAME) 已卸载"

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html

# 下载依赖
.PHONY: deps
deps:
	@echo "下载依赖..."
	@$(GO) mod download
	@$(GO) mod tidy

# 更新依赖
.PHONY: deps-update
deps-update:
	@echo "更新依赖..."
	@$(GO) get -u ./...
	@$(GO) mod tidy

# 生成文档
.PHONY: docs
docs:
	@echo "生成文档..."
	@$(GO) run ./cmd/vman completion bash > scripts/vman-completion.bash
	@$(GO) run ./cmd/vman completion zsh > scripts/vman-completion.zsh
	@$(GO) run ./cmd/vman completion fish > scripts/vman-completion.fish

# 开发环境设置
.PHONY: dev-setup
dev-setup:
	@echo "设置开发环境..."
	@$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GO) mod download

# 运行开发版本
.PHONY: run
run: build
	@echo "运行 $(PROJECT_NAME)..."
	@./$(BUILD_DIR)/$(PROJECT_NAME)

# 显示帮助
.PHONY: help
help:
	@echo "可用的 make 目标:"
	@echo "  all           - 执行完整的构建流程（清理、格式化、检查、测试、构建）"
	@echo "  build         - 构建本地版本"
	@echo "  build-all     - 跨平台构建"
	@echo "  clean         - 清理构建文件"
	@echo "  coverage      - 生成测试覆盖率报告"
	@echo "  deps          - 下载依赖"
	@echo "  deps-update   - 更新依赖"
	@echo "  dev-setup     - 设置开发环境"
	@echo "  docs          - 生成文档"
	@echo "  fmt           - 格式化代码"
	@echo "  install       - 安装到用户目录 ($(INSTALL_DIR))"
	@echo "  install-system- 安装到系统目录 (需要sudo)"
	@echo "  lint          - 代码检查"
	@echo "  run           - 运行开发版本"
	@echo "  test          - 运行测试"
	@echo "  uninstall     - 卸载 vman"
	@echo "  help          - 显示此帮助信息"
	@echo ""
	@echo "安装信息:"
	@echo "  操作系统: $(UNAME_S)"
	@echo "  用户安装目录: $(INSTALL_DIR)"
	@echo "  配置目录: $(CONFIG_DIR)"