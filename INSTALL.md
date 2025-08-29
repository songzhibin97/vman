# vman 安装指南

本文档提供了 vman（通用命令行工具版本管理器）的详细安装指导，涵盖多种安装方式和平台配置。

## 📋 目录

- [📋 系统要求](#-系统要求)
- [🚀 安装方式](#-安装方式)
- [🔧 安装后配置](#-安装后配置)
- [🔄 更新和卸载](#-更新和卸载)
- [🚨 故障排除](#-故障排除)
- [📚 下一步](#-下一步)

## 📋 系统要求

### 推荐配置

对于最佳体验，建议使用以下配置：

| 组件 | 推荐 | 最低要求 |
|------|------|----------|
| 操作系统 | 最新稳定版本 | 见下表支持列表 |
| 架构 | arm64 (Apple Silicon/新AMD64) | amd64 |
| 内存 | 2GB+ | 512MB |
| 磁盘空间 | 1GB+ | 100MB |
| 网络带宽 | 10Mbps+ | 1Mbps |

### 依赖项检查

在安装 vman 之前，请确保系统具有以下基础工具：

```bash
# 检查系统依赖
which curl || which wget  # HTTP 客户端
which tar                 # 解压工具（Linux/macOS）
which unzip              # 解压工具（Windows）
which git                # Git 版本控制（可选，用于从源码安装）
```

### 支持的平台

- **Linux**: Ubuntu 18.04+, CentOS 7+, Debian 9+, Arch Linux
- **macOS**: macOS 10.15+ (Catalina 及更高版本)
- **Windows**: Windows 10+, Windows Server 2019+

## 🚀 安装方式

### 方式一：自动安装脚本（推荐）

这是最简单快捷的安装方式，适合大多数用户。

#### Linux/macOS

```bash
# 使用 curl 下载并安装
curl -fsSL https://get.vman.dev | bash

# 或使用 wget
wget -qO- https://get.vman.dev | bash

# 自定义安装目录
curl -fsSL https://get.vman.dev | VMAN_INSTALL_DIR=/usr/local/bin bash
```

#### Windows (PowerShell)

```powershell
# 管理员权限下运行
Invoke-WebRequest -Uri https://get.vman.dev/install.ps1 -UseBasicParsing | Invoke-Expression

# 或者
iwr -useb https://get.vman.dev/install.ps1 | iex
```

### 方式二：包管理器安装

#### Homebrew (macOS/Linux)

```bash
# 添加 tap
brew tap songzhibin97/vman

# 安装 vman
brew install vman

# 更新到最新版本
brew upgrade vman
```

#### Scoop (Windows)

```powershell
# 添加 bucket
scoop bucket add vman https://github.com/songzhibin97/scoop-vman.git

# 安装 vman
scoop install vman

# 更新到最新版本
scoop update vman
```

#### APT (Ubuntu/Debian)

```bash
# 添加 GPG 密钥
curl -fsSL https://apt.vman.dev/key.gpg | sudo apt-key add -

# 添加仓库
echo "deb [trusted=yes] https://apt.vman.dev/ /" | sudo tee /etc/apt/sources.list.d/vman.list

# 更新包列表并安装
sudo apt update
sudo apt install vman
```

#### YUM/DNF (CentOS/RHEL/Fedora)

```bash
# 添加仓库
sudo tee /etc/yum.repos.d/vman.repo <<EOF
[vman]
name=vman Repository
baseurl=https://yum.vman.dev/
enabled=1
gpgcheck=1
gpgkey=https://yum.vman.dev/key.gpg
EOF

# 安装 vman
sudo yum install vman  # CentOS/RHEL
sudo dnf install vman  # Fedora
```

#### Snap (Ubuntu)

```bash
# 安装 snap 包
sudo snap install vman --classic

# 更新到最新版本
sudo snap refresh vman
```

#### Flatpak (通用 Linux)

```bash
# 安装 flatpak 包
flatpak install flathub dev.vman.VMan

# 更新到最新版本
flatpak update dev.vman.VMan
```

### 方式五：Docker 安装

适合需要容器化环境或临时使用的情况。

#### 使用官方镜像

```bash
# 拉取镜像
docker pull songzhibin97/vman:latest

# 创建别名
echo 'alias vman="docker run --rm -v \$PWD:/workspace -v \$HOME/.vman:/root/.vman songzhibin97/vman"' >> ~/.bashrc
source ~/.bashrc

# 使用 vman
vman --version
```

#### 构建自定义镜像

```bash
# 克隆仓库
git clone https://github.com/songzhibin97/vman.git
cd vman

# 构建镜像
docker build -t vman:local .

# 使用本地镜像
docker run --rm vman:local --version
```

### 方式三：手动安装

#### 下载预编译二进制文件

1. 访问 [Releases 页面](https://github.com/songzhibin97/vman/releases)
2. 下载适合你平台的二进制文件：

```bash
# Linux amd64
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz

# Linux arm64
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-arm64.tar.gz

# macOS amd64 (Intel)
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-darwin-amd64.tar.gz

# macOS arm64 (Apple Silicon)
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-darwin-arm64.tar.gz

# Windows amd64
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-windows-amd64.zip
```

3. 解压并安装：

```bash
# Linux/macOS
tar -xzf vman-*.tar.gz
sudo mv vman /usr/local/bin/
chmod +x /usr/local/bin/vman

# Windows (PowerShell)
Expand-Archive -Path vman-windows-amd64.zip -DestinationPath C:\Program Files\vman\
# 将 C:\Program Files\vman\ 添加到 PATH 环境变量
```

### 方式四：从源码编译

适合开发者或需要自定义构建的用户。

#### 环境准备

```bash
# 安装 Go 1.21+
# Linux (使用发行版包管理器或官方安装包)
# macOS: brew install go
# Windows: 从 https://golang.org/dl/ 下载安装

# 验证 Go 安装
go version
```

#### 编译安装

```bash
# 方法 1: 使用 go install
go install github.com/songzhibin97/vman/cmd/vman@latest

# 方法 2: 克隆并编译
git clone https://github.com/songzhibin97/vman.git
cd vman

# 构建
make build

# 安装
sudo make install

# 或手动复制
sudo cp build/vman /usr/local/bin/
```

#### 交叉编译

```bash
# 为多个平台构建
make build-all

# 构建产物在 dist/ 目录
ls dist/
```

## 🔧 安装后配置

### 1. 验证安装

```bash
# 检查版本
vman --version

# 显示帮助信息
vman --help

# 运行诊断
vman doctor
```

### 2. Shell 集成设置

为了启用自动补全和命令代理功能，需要配置 shell 集成：

#### Bash

```bash
# 添加到 ~/.bashrc
echo 'eval "$(vman init bash)"' >> ~/.bashrc

# 立即生效
source ~/.bashrc
```

#### Zsh

```bash
# 添加到 ~/.zshrc
echo 'eval "$(vman init zsh)"' >> ~/.zshrc

# 立即生效
source ~/.zshrc
```

#### Fish

```bash
# 添加到 ~/.config/fish/config.fish
mkdir -p ~/.config/fish
echo 'vman init fish | source' >> ~/.config/fish/config.fish

# 立即生效
source ~/.config/fish/config.fish
```

#### PowerShell

```powershell
# 检查配置文件路径
$PROFILE

# 添加到配置文件
Add-Content $PROFILE 'Invoke-Expression (& vman init powershell)'

# 重新加载配置
. $PROFILE
```

### 3. 初始化 vman

```bash
# 初始化配置
vman init

# 检查配置
vman config list

# 设置默认配置
vman config set download.timeout 300s
vman config set logging.level info
```

## 🔄 更新和卸载

### 更新 vman

```bash
# 包管理器安装的版本
brew upgrade vman          # Homebrew
scoop update vman          # Scoop
sudo apt update && sudo apt upgrade vman  # APT

# 手动更新（重新下载安装）
curl -fsSL https://get.vman.dev | bash

# 从源码更新
cd vman && git pull && make build && sudo make install
```

### 卸载 vman

```bash
# 删除 vman 二进制文件
sudo rm -f /usr/local/bin/vman

# 删除配置和数据（可选）
rm -rf ~/.vman

# 从 shell 配置中移除集成代码
# 编辑 ~/.bashrc, ~/.zshrc 等文件，删除 vman init 相关行

# 包管理器卸载
brew uninstall vman        # Homebrew
scoop uninstall vman       # Scoop
sudo apt remove vman       # APT
```

## 🚨 故障排除

### 常见问题

#### 权限问题

```bash
# 如果遇到权限问题，确保有执行权限
chmod +x /usr/local/bin/vman

# 或安装到用户目录
mkdir -p ~/bin
cp vman ~/bin/
export PATH="$HOME/bin:$PATH"
```

#### PATH 问题

```bash
# 检查 vman 是否在 PATH 中
which vman

# 手动添加到 PATH（临时）
export PATH="/usr/local/bin:$PATH"

# 永久添加到 PATH
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
```

#### 网络问题

```bash
# 使用代理下载
export https_proxy=http://proxy.example.com:8080
curl -fsSL https://get.vman.dev | bash

# 或手动下载后安装
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz
```

### 获取帮助

如果遇到安装问题：

1. 查看 [FAQ 文档](FAQ.md)
2. 搜索 [GitHub Issues](https://github.com/songzhibin97/vman/issues)
3. 创建新的 [Issue](https://github.com/songzhibin97/vman/issues/new)
4. 查看 [故障排除指南](TROUBLESHOOTING.md)

## 📚 下一步

安装完成后，建议阅读：

- [用户教程](TUTORIAL.md) - 学习如何使用 vman
- [配置指南](CONFIGURATION.md) - 详细配置说明
- [最佳实践](docs/best-practices.md) - 使用技巧和建议

---

**注意**: 请确保从官方渠道下载 vman，避免使用来源不明的第三方安装包。