# vman - 通用命令行工具版本管理器

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Tests](https://github.com/songzhibin97/vman/workflows/CI/badge.svg)](https://github.com/songzhibin97/vman/actions)
[![Coverage](https://codecov.io/gh/songzhibin97/vman/branch/main/graph/badge.svg)](https://codecov.io/gh/songzhibin97/vman)
[![Go Report Card](https://goreportcard.com/badge/github.com/songzhibin97/vman)](https://goreportcard.com/report/github.com/songzhibin97/vman)
[![Release](https://img.shields.io/github/v/release/songzhibin97/vman)](https://github.com/songzhibin97/vman/releases)
[![Downloads](https://img.shields.io/github/downloads/songzhibin97/vman/total)](https://github.com/songzhibin97/vman/releases)

<div align="center">
  <img src="https://raw.githubusercontent.com/songzhibin97/vman/main/assets/logo.png" alt="vman logo" width="200"/>
  <p><em>让多版本工具管理变得简单</em></p>
</div>

vman 是一个现代化的通用命令行工具版本管理器，专为开发者设计，解决在同一台计算机上管理多个命令行工具版本的复杂性。它提供了简洁的API、强大的功能和无缝的用户体验。

## 📚 目录

- [✨ 核心特性](#-核心特性)
- [🎯 核心概念](#-核心概念)
- [📦 安装指南](#-安装指南)
- [🚀 快速开始](#-快速开始)
- [📚 命令参考](#-命令参考)
- [📁 目录结构](#-目录结构)
- [⚙️ 配置详解](#️-配置详解)
- [🔧 开发](#-开发)
- [🤝 贡献](#-贡献)
- [📜 许可证](#-许可证)
- [🙏 致谢](#-致谢)
- [📞 支持与社区](#-支持与社区)

## ✨ 核心特性

### 🎯 开发者友好
- 🔧 **通用性**: 支持任意二进制工具的版本管理，不局限于特定语言生态
- 👻 **透明使用**: 拦截命令并代理到正确版本，用户无需改变使用习惯
- 📁 **项目感知**: 自动检测项目配置，智能切换工具版本
- 🔄 **灵活切换**: 支持全局和项目级版本切换

### 🚀 强大功能
- 📦 **智能管理**: 自动下载、安装、注册工具版本
- 🌐 **多源支持**: 支持 GitHub Release、直接下载等多种下载策略
- 🔗 **符号链接**: 智能管理工具符号链接，确保命令可用性
- 🛡️ **安全可靠**: 内置校验机制，确保下载文件的完整性
- 🔧 **protoc一键解决方案**: 专门解决Protocol Buffer编译环境配置问题

### ⚡ 高性能设计
- 🚀 **零开销**: 最小化命令执行开销，几乎无感知
- 💾 **智能缓存**: 高效的缓存机制，减少重复下载
- 🔄 **并发下载**: 支持并发下载，提升安装速度
- 📈 **性能监控**: 内置性能分析工具

### 🔒 安全特性
- 🔐 **文件校验**: 内置 SHA256 校验机制，确保下载文件的完整性
- 🔒 **HTTPS 强制**: 所有下载都通过 HTTPS 进行，保障传输安全
- 🎯 **权限最小化**: 遵循最小权限原则，降低安全风险
- 🚫 **沙箱执行**: 隔离工具执行环境

### 🌍 跨平台支持
- 💻 **Windows**: Windows 10/11, Windows Server 2019+
- 🍎 **macOS**: macOS 10.15+ (Intel & Apple Silicon)
- 🐧 **Linux**: Ubuntu 18.04+, CentOS 7+, Debian 9+, Arch Linux

### 📊 系统要求
- 🔧 架构：amd64, arm64
- 💾 磁盘空间：至少 100MB
- 🌐 网络：用于下载工具的互联网连接

## 🛠️ 支持的工具

vman 当前内置支持以下开发工具：

### Protocol Buffers 生态
- **protoc** - Protocol Buffers 编译器
- **protoc-gen-go** - Go Protocol Buffer 插件
- **protoc-gen-go-grpc** - gRPC Go 插件
- **protoc-gen-go-http** - HTTP Go 插件 (Kratos)

### 数据库工具
- **sqlc** - 从 SQL 生成类型安全的 Go 代码

### 基础设施工具
- **terraform** - 基础设施即代码工具
- **kubectl** - Kubernetes 命令行工具（可选，避免冲突）
- **helm** - Kubernetes 包管理器

> **扩展支持**: vman 支持任意二进制工具，您可以通过添加配置文件来支持更多工具。

## 🎯 核心概念

### 全局版本 vs 项目版本

- **全局版本**: 在系统范围内使用的默认版本
- **项目版本**: 在特定项目目录下使用的版本，优先级高于全局版本

### 版本优先级

1. 项目级配置 (`.vmanrc`)
2. 全局配置 (`~/.vman/config.yaml`)
3. 默认配置

## 📦 安装指南

### 快速安装

#### 使用安装脚本（推荐）
```bash
# Linux/macOS
curl -fsSL https://get.vman.dev | bash

# Windows (PowerShell)
iwr -useb https://get.vman.dev/install.ps1 | iex
```

#### 包管理器安装
```bash
# Homebrew (macOS/Linux)
brew install vman

# Scoop (Windows)
scoop install vman

# APT (Ubuntu/Debian)
echo "deb [trusted=yes] https://apt.vman.dev/ /" | sudo tee /etc/apt/sources.list.d/vman.list
sudo apt update && sudo apt install vman
```

### 手动安装

### 手动安装

#### 使用源码构建（推荐）
```bash
# 克隆仓库
git clone https://github.com/songzhibin97/vman.git
cd vman

# 构建并安装到用户目录（推荐）
make install

# 或者安装到系统目录（需要sudo）
make install-system

# 查看所有安装选项
make help
```

#### 二进制安装
1. 从 [Releases](https://github.com/songzhibin97/vman/releases) 下载对应平台的二进制文件
2. 解压并移动到 PATH 目录：
```bash
# Linux/macOS - 用户目录安装（推荐）
tar -xzf vman-*.tar.gz
mkdir -p ~/.local/bin
mv vman ~/.local/bin/

# Linux/macOS - 系统目录安装
tar -xzf vman-*.tar.gz
sudo mv vman /usr/local/bin/

# Windows
# 解压后将 vman.exe 添加到 PATH
```

### 验证安装
```bash
vman --version
vman help
```

### Shell 集成
安装完成后，需要配置 PATH 和 shell 集成：

#### 配置 PATH
```bash
# 将 vman 安装目录添加到 PATH
# macOS/Linux (用户目录安装)
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc   # Bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc    # Zsh

# Windows (手动添加到系统 PATH)
```

#### 启用 shell 集成
```bash
# Bash
echo 'eval "$(vman init bash)"' >> ~/.bashrc

# Zsh  
echo 'eval "$(vman init zsh)"' >> ~/.zshrc

# Fish
echo 'vman init fish | source' >> ~/.config/fish/config.fish

# PowerShell
Add-Content $PROFILE 'Invoke-Expression (& vman init powershell)'
```

#### 重新加载配置
```bash
source ~/.bashrc  # Bash
source ~/.zshrc   # Zsh
# 或者重新启动终端
```

## 🚀 快速开始

> **注意**: vman 默认不管理 kubectl，以避免与您现有的 kubectl 配置冲突。如果您需要管理 kubectl，请手动添加。

### 第一步：初始化环境
```bash
# 初始化 vman 配置
vman init

# 添加 shell 集成（如果还没有添加）
echo 'eval "$(vman init bash)"' >> ~/.bashrc  # Bash
source ~/.bashrc
```

### 第二步：添加工具
```bash
# 添加 Protocol Buffers 相关工具
vman add-source protoc --type github --repo protocolbuffers/protobuf --description "Protocol Buffers compiler"
vman add-source protoc-gen-go --type github --repo protocolbuffers/protobuf-go --description "Protocol buffer compiler for Go"
vman add-source protoc-gen-go-grpc --type github --repo grpc/grpc-go --description "Protocol buffer compiler for gRPC-Go"
vman add-source protoc-gen-go-http --type github --repo go-kratos/kratos --description "Protocol buffer compiler for Go HTTP API"

# 添加 SQL 代码生成器
vman add-source sqlc --type github --repo sqlc-dev/sqlc --description "Generate type-safe code from SQL"

# 添加其他常用工具
vman add terraform
vman add helm
vman add kubectl  # 注意：只在需要时添加，避免与现有kubectl冲突
```

### 第三步：安装版本
```bash
# 注册系统现有工具（推荐方式）
vman register protoc 29.3 /opt/homebrew/bin/protoc
vman register sqlc 1.26.0 /opt/homebrew/bin/sqlc

# 或者从网络下载安装
vman install protoc 28.0
vman install sqlc 1.26.0
vman install protoc-gen-go 1.34.2

# 安装最新版本
vman install protoc latest
vman install sqlc latest

# 批量安装
vman install protoc 28.0 29.3
vman install sqlc 1.25.0 1.26.0
```

### 第四步：管理版本
```bash
# 查看已安装版本
vman list protoc
vman list sqlc

# 设置全局默认版本
vman global protoc 29.3
vman global sqlc 1.26.0

# 设置项目版本（在项目目录中）
cd my-project
vman local protoc 28.0
vman local sqlc 1.25.0

# 查看当前使用版本
vman current protoc
vman current sqlc
```

### 第五步：使用工具
```bash
# 直接使用，vman 自动路由到正确版本
protoc --version
sqlc version

# 在项目目录中会自动使用项目配置的版本
cd my-project
protoc --version    # 使用 28.0
sqlc version       # 使用 1.25.0

cd ~/
protoc --version    # 使用全局版本 29.3
sqlc version       # 使用全局版本 1.26.0
```

### 高级用法
```bash
# 临时使用特定版本
vman exec protoc@28.0 --version
vman exec sqlc@1.25.0 version

# 查看工具路径
vman which protoc
vman which sqlc

# 清理未使用的版本
vman cleanup

# 更新工具源信息
vman update

# 卸载版本
vman uninstall protoc 28.0
vman remove-source sqlc
```

### protoc 一键解决方案
对于 Protocol Buffer 开发，vman 提供了专门的一键解决方案：

```bash
# 安装 protoc 相关工具的特定版本
vman install protoc-gen-go 1.31.0
vman install protoc-gen-go-grpc 1.3.0  
vman install protoc-gen-go-http 2.7.0

# 设置为使用这些版本
vman use protoc-gen-go 1.31.0
vman use protoc-gen-go-grpc 1.3.0
vman use protoc-gen-go-http 2.7.0

# 一键设置 protoc 环境（解决 shim 冲突、设置 PATH 等）
vman protoc setup

# 一键执行 make api（在项目目录中）
vman protoc make-api

# 或者指定项目目录
vman protoc make-api --dir /path/to/project

# 在 protoc 模式下执行任意命令
vman protoc exec make build
vman protoc exec protoc --version

# 查看 protoc 环境状态
vman protoc status
```

这个一键解决方案将原本的 8 个手动步骤简化为 1 个命令，自动处理：
- 启用 vman 代理
- 智能备份 protoc shim（避免冲突）
- 设置插件路径环境变量
- 一键执行 make api

## 📚 命令参考

### 基础命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `vman init` | 初始化 vman 环境 | `vman init` |
| `vman add <tool>` | 添加工具源 | `vman add kubectl` |
| `vman install <tool> <version>` | 安装工具版本 | `vman install kubectl 1.28.0` |
| `vman global <tool> <version>` | 设置全局版本 | `vman global kubectl 1.28.0` |
| `vman local <tool> <version>` | 设置项目版本 | `vman local kubectl 1.29.0` |
| `vman list [tool]` | 显示已安装版本 | `vman list kubectl` |
| `vman current [tool]` | 显示当前版本 | `vman current kubectl` |

### 管理命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `vman list-all <tool>` | 显示可用版本 | `vman list-all kubectl` |
| `vman search <keyword>` | 搜索可用工具 | `vman search kube` |
| `vman uninstall <tool> <version>` | 卸载版本 | `vman uninstall kubectl 1.28.0` |
| `vman remove <tool>` | 移除工具源 | `vman remove kubectl` |
| `vman update` | 更新工具源信息 | `vman update` |
| `vman cleanup` | 清理缓存和旧版本 | `vman cleanup` |

### 实用命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `vman exec <tool>@<version> <args>` | 临时使用特定版本 | `vman exec kubectl@1.27.0 version` |
| `vman which <tool>` | 显示工具路径 | `vman which kubectl` |
| `vman reshim [tool]` | 重新生成符号链接 | `vman reshim kubectl` |
| `vman info <tool>` | 显示工具详细信息 | `vman info kubectl` |
| `vman doctor` | 诊断环境问题 | `vman doctor` |
| `vman version` | 显示 vman 版本 | `vman version` |

## 📁 目录结构

vman 会根据操作系统标准创建以下配置目录：

### 配置目录位置
- **macOS**: `~/Library/Application Support/vman/`
- **Linux**: `~/.config/vman/`  
- **Windows**: `~/AppData/Local/vman/`

### 目录结构
```
# macOS 示例
~/Library/Application Support/vman/  # vman 配置目录
├── config.yaml                    # 全局配置文件
├── tools/                         # 工具配置
│   ├── protoc.toml                # protoc 配置
│   ├── protoc-gen-go.toml         # protoc-gen-go 配置
│   ├── protoc-gen-go-grpc.toml    # protoc-gen-go-grpc 配置
│   ├── protoc-gen-go-http.toml    # protoc-gen-go-http 配置
│   └── sqlc.toml                  # sqlc 配置
├── versions/                      # 工具版本存储
│   ├── protoc/
│   │   └── 29.3/                   # protoc 29.3 版本
│   │       └── bin/
│   │           └── protoc*
│   └── sqlc/
│       └── 1.26.0/                # sqlc 1.26.0 版本
│           └── bin/
│               └── sqlc*
├── shims/                         # 命令垫片（代理命令）
│   ├── protoc*
│   └── sqlc*
├── cache/                         # 缓存目录
│   ├── downloads/                 # 下载缓存
│   └── sources/                   # 元数据缓存
└── logs/                          # 日志文件
    └── vman.log

# 项目目录中的配置文件
my-project/
├── .vmanrc                        # 项目级配置
└── .vman-version                  # 简单版本指定（可选）
```

## ⚙️ 配置详解

### 全局配置 (`~/.vman/config.yaml`)

```yaml
# vman 配置文件版本
version: "1.0"

# 全局设置
settings:
  # 下载设置
  download:
    timeout: 300s                    # 下载超时时间
    retries: 3                       # 重试次数
    concurrent_downloads: 2          # 并发下载数
    verify_checksum: true            # 校验文件完整性
    
  # 代理设置
  proxy:
    enabled: true                    # 启用命令代理
    shims_in_path: true             # 将 shims 目录添加到 PATH
    performance_mode: true           # 性能模式
    
  # 日志设置
  logging:
    level: "info"                   # 日志级别: debug, info, warn, error
    file: "~/.vman/logs/vman.log"    # 日志文件路径
    max_size: "10MB"                # 日志文件最大大小
    max_files: 5                     # 保留日志文件数量
    
  # 缓存设置
  cache:
    enabled: true                    # 启用缓存
    ttl: "24h"                      # 缓存有效期
    max_size: "1GB"                 # 缓存最大大小

# 全局默认版本
global_versions:
  protoc: "29.3"
  sqlc: "1.26.0"
  protoc-gen-go: "1.34.2"
  protoc-gen-go-grpc: "1.5.1"
  protoc-gen-go-http: "2.8.0"

# 环境变量
environment:
  KUBECTL_CACHE_DIR: "~/.vman/cache/kubectl"
  TERRAFORM_LOG_LEVEL: "INFO"
```

### 项目配置 (`.vmanrc`)

```yaml
# 项目配置文件
version: "1.0"

# 项目使用的工具版本
tools:
  protoc: "28.0"              # 指定 protoc 版本
  sqlc: "1.25.0"              # 指定 sqlc 版本
  protoc-gen-go: "1.34.1"     # 指定 protoc-gen-go 版本
  protoc-gen-go-grpc: "1.5.0" # 指定 protoc-gen-go-grpc 版本

# 项目特定设置
settings:
  auto_install: true       # 自动安装缺失的版本
  strict_mode: false       # 严格模式（必须使用指定版本）

# 项目环境变量
environment:
  KUBECONFIG: "./kubeconfig"
  TF_VAR_environment: "staging"
```

### 简化配置 (`.vman-version`)

对于简单场景，可以使用简化的版本文件：

```
# 一行一个工具版本
protoc 28.0
sqlc 1.25.0
protoc-gen-go 1.34.1
protoc-gen-go-grpc 1.5.0
```

### 工具源配置

vman 支持多种下载策略：

#### GitHub Release

```toml
[tool]
name = "sqlc"

[download]
type = "github"
repository = "sqlc-dev/sqlc"
asset_pattern = "sqlc_{version}_{os}_{arch}.tar.gz"
```

#### 直接下载

```toml
[tool]
name = "kubectl"

[download]
type = "direct"
url_template = "https://dl.k8s.io/release/v{version}/bin/{os}/{arch}/kubectl"
```

## 🔧 开发

### 环境要求

- Go 1.21+
- Make

### 开发设置

```bash
# 克隆项目
git clone https://github.com/songzhibin97/vman.git
cd vman

# 设置开发环境
make dev-setup

# 安装依赖
make deps
```

### 构建和测试

```bash
# 代码格式化
make fmt

# 代码检查
make lint

# 运行测试
make test

# 构建
make build

# 跨平台构建
make build-all
```

### 项目结构

```
vman/
├── cmd/vman/           # 主程序入口
├── internal/           # 内部包
│   ├── cli/           # CLI 命令
│   ├── config/        # 配置管理
│   ├── version/       # 版本管理
│   ├── download/      # 下载管理
│   ├── proxy/         # 命令代理
│   └── storage/       # 存储管理
├── pkg/               # 公共包
│   ├── types/         # 类型定义
│   └── utils/         # 工具函数
├── configs/           # 配置文件模板
├── docs/              # 文档
└── test/              # 测试
```

## 🤝 贡献

欢迎贡献代码！请阅读 [贡献指南](CONTRIBUTING.md) 了解详情。

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -am 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。详情请见 [LICENSE](LICENSE) 文件。

## 🙏 致谢

- [asdf](https://asdf-vm.com/) - 启发了这个项目的设计理念
- [Cobra](https://github.com/spf13/cobra) - 优秀的 CLI 框架
- [Viper](https://github.com/spf13/viper) - 强大的配置管理库
- [resty](https://github.com/go-resty/resty) - 简洁的 HTTP 客户端
- [logrus](https://github.com/sirupsen/logrus) - 结构化日志库

## 📞 支持与社区

### 获取帮助

如果你遇到问题或有建议，请：

1. 📖 查看 [完整文档](docs/)
2. 🔍 搜索 [已知问题](https://github.com/songzhibin97/vman/issues)
3. 💬 创建新的 [Issue](https://github.com/songzhibin97/vman/issues/new)
4. 📧 发送邮件到 songzhibin97@gmail.com

### 贡献代码

我们欢迎所有形式的贡献：

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码修复
- 🌍 添加多语言支持

详情请阅读 [贡献指南](CONTRIBUTING.md)。

### 社区交流

- 💬 [Discussions](https://github.com/songzhibin97/vman/discussions) - 讨论和交流
- 📢 [Twitter](https://twitter.com/songzhibin97) - 获取最新动态
- 📺 [YouTube](https://youtube.com/@songzhibin97) - 视频教程

---

⭐ 如果这个项目对你有帮助，请给它一个 Star！