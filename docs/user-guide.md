# vman 用户使用指南

欢迎使用 vman - 通用命令行工具版本管理器！本指南将详细介绍如何使用 vman 管理你的命令行工具版本。

## 📋 目录

- [🚀 快速上手](#-快速上手)
- [📦 工具管理](#-工具管理)
- [🔄 版本管理](#-版本管理)
- [⚙️ 配置管理](#️-配置管理)
- [🔍 查询和检索](#-查询和检索)
- [🛠️ 高级功能](#️-高级功能)
- [🔧 protoc一键解决方案](#-protoc一键解决方案)
- [🎯 实际应用场景](#-实际应用场景)
- [💡 最佳实践](#-最佳实践)
- [🚨 故障排除](#-故障排除)

## 🚀 快速上手

### 初始化 vman

安装完成后，首先需要初始化 vman 环境：

```bash
# 初始化配置目录和基础配置
vman init

# 启用 shell 集成（根据你的 shell 选择）
echo 'eval "$(vman init bash)"' >> ~/.bashrc    # Bash
echo 'eval "$(vman init zsh)"' >> ~/.zshrc      # Zsh
echo 'vman init fish | source' >> ~/.config/fish/config.fish  # Fish

# 重新加载 shell 配置
source ~/.bashrc  # 或重启终端
```

### 验证安装

```bash
# 检查 vman 版本
vman --version

# 运行诊断检查
vman doctor

# 查看帮助信息
vman --help
```

## 📦 工具管理

### 搜索和添加工具

```bash
# 搜索可用工具
vman search kubectl
vman search "kube"     # 模糊搜索
vman search --all      # 显示所有可用工具

# 添加工具源
vman add kubectl
vman add terraform
vman add helm

# 查看已添加的工具
vman list-tools
```

### 工具源配置

vman 支持多种工具源类型：

#### GitHub Release 工具
```bash
# 大多数开源工具都使用 GitHub Release
vman add kubectl      # Kubernetes CLI
vman add terraform    # HashiCorp Terraform
vman add helm         # Kubernetes Helm
vman add sqlc         # SQL 编译器
```

#### 直接下载工具
```bash
# 一些工具提供直接下载链接
vman add nodejs       # Node.js
vman add golang       # Go 语言
```

#### 自定义工具源
你也可以添加自定义的工具源配置：

```bash
# 创建自定义工具配置
vman add-custom my-tool --config ./my-tool.toml
```

## 🔄 版本管理

### 查看可用版本

```bash
# 查看工具的所有可用版本
vman list-all kubectl

# 查看最新的几个版本
vman list-all kubectl --limit 10

# 查看特定版本范围
vman list-all kubectl --range ">=1.28.0,<1.30.0"
```

### 安装版本

```bash
# 安装特定版本
vman install kubectl 1.28.0
vman install terraform 1.6.0

# 安装最新版本
vman install kubectl latest
vman install kubectl stable

# 批量安装多个版本
vman install kubectl 1.28.0 1.29.0 1.30.0

# 安装时指定别名
vman install kubectl 1.28.0 --alias lts
```

### 查看已安装版本

```bash
# 查看所有已安装版本
vman list

# 查看特定工具的已安装版本
vman list kubectl

# 显示详细信息
vman list kubectl --verbose

# 显示安装路径
vman list kubectl --paths
```

### 设置和切换版本

#### 全局版本设置

```bash
# 设置全局默认版本
vman global kubectl 1.28.0
vman global terraform 1.6.0

# 查看当前全局版本
vman global kubectl
vman global --all
```

#### 项目版本设置

```bash
# 在项目目录中设置本地版本
cd my-k8s-project
vman local kubectl 1.29.0
vman local terraform 1.7.0

# 查看当前项目版本
vman local kubectl
vman local --all

# 移除项目版本设置（回退到全局版本）
vman local kubectl --unset
```

#### 临时版本使用

```bash
# 临时使用特定版本执行命令
vman exec kubectl@1.27.0 version
vman exec terraform@1.5.0 --version

# 在临时环境中运行
vman shell kubectl@1.27.0
# 现在在这个 shell 中，kubectl 将是 1.27.0 版本
kubectl version
exit  # 退出临时环境
```

### 版本优先级

vman 按以下优先级解析版本：

1. **临时版本**: `vman exec tool@version`
2. **项目版本**: `.vmanrc` 或 `.vman-version` 文件
3. **全局版本**: `~/.vman/config.yaml`
4. **默认版本**: 工具的最新稳定版本

### 查看当前版本

```bash
# 查看当前使用的版本
vman current kubectl
vman current --all

# 查看版本来源
vman current kubectl --verbose

# 查看所有工具的当前版本
vman status
```

## ⚙️ 配置管理

### 全局配置

全局配置文件位于 `~/.vman/config.yaml`：

```bash
# 编辑全局配置
vman config edit

# 查看配置
vman config show

# 设置配置项
vman config set download.timeout 600s
vman config set logging.level debug

# 获取配置项
vman config get download.timeout
vman config get logging.level
```

### 项目配置

#### 使用 .vmanrc 文件

在项目根目录创建 `.vmanrc` 文件：

```yaml
# .vmanrc
version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.7.0"
  helm: "3.12.0"

settings:
  auto_install: true
  strict_mode: false

environment:
  KUBECONFIG: "./kubeconfig"
  TF_VAR_environment: "development"
```

#### 使用简化的 .vman-version 文件

```bash
# .vman-version
kubectl 1.29.0
terraform 1.7.0
helm 3.12.0
```

### 环境变量

vman 支持通过环境变量覆盖配置：

```bash
# 设置下载超时
export VMAN_DOWNLOAD_TIMEOUT=300s

# 设置日志级别
export VMAN_LOG_LEVEL=debug

# 设置配置目录
export VMAN_CONFIG_DIR=~/.config/vman
```

## 🔍 查询和检索

### 工具信息查询

```bash
# 显示工具详细信息
vman info kubectl

# 显示工具安装路径
vman which kubectl
vman which kubectl@1.28.0

# 显示工具的实际可执行文件路径
vman whereis kubectl
```

### 版本信息查询

```bash
# 查看版本详细信息
vman show kubectl 1.28.0

# 查看版本的安装时间
vman list kubectl --with-time

# 查看版本的大小
vman list kubectl --with-size
```

### 依赖关系查询

```bash
# 查看工具依赖
vman deps kubectl

# 查看被依赖情况
vman rdeps kubectl
```

## 🛠️ 高级功能

### 工具管理

#### 更新工具源

```bash
# 更新所有工具源信息
vman update

# 更新特定工具源
vman update kubectl

# 强制更新（忽略缓存）
vman update --force
```

#### 移除工具和版本

```bash
# 卸载特定版本
vman uninstall kubectl 1.27.0

# 卸载工具的所有版本
vman uninstall kubectl --all

# 移除工具源（不删除已安装版本）
vman remove kubectl

# 彻底删除工具（包括所有版本和配置）
vman purge kubectl
```

### 系统维护

#### 清理功能

```bash
# 清理未使用的版本
vman cleanup

# 清理下载缓存
vman cleanup --cache

# 清理所有临时文件
vman cleanup --all

# 查看可清理的内容（不实际清理）
vman cleanup --dry-run
```

#### 重建链接

```bash
# 重建所有工具的符号链接
vman reshim

# 重建特定工具的符号链接
vman reshim kubectl

# 强制重建（即使链接存在）
vman reshim --force
```

### 备份和恢复

#### 导出配置

```bash
# 导出当前配置
vman export > vman-config.yaml

# 导出特定工具配置
vman export kubectl > kubectl-config.yaml

# 导出安装列表
vman export --list > installed-tools.txt
```

#### 导入配置

```bash
# 导入配置
vman import vman-config.yaml

# 批量安装
vman import --install installed-tools.txt
```

### 插件系统

#### 管理插件

```bash
# 列出可用插件
vman plugin list

# 安装插件
vman plugin install completion-extra

# 启用/禁用插件
vman plugin enable completion-extra
vman plugin disable completion-extra

# 移除插件
vman plugin remove completion-extra
```

#### 自定义插件

你可以创建自定义插件来扩展 vman 功能：

```bash
# 创建插件目录
mkdir -p ~/.vman/plugins/my-plugin

# 创建插件脚本
cat > ~/.vman/plugins/my-plugin/plugin.sh << 'EOF'
#!/bin/bash
# 自定义插件逻辑
EOF

# 注册插件
vman plugin register my-plugin
```

## 🔧 protoc一键解决方案

vman 提供了专门的 protoc 一键解决方案，帮助解决 Protocol Buffer 编译过程中的版本冲突和环境配置问题。这个功能特别适合需要使用特定版本 protoc 插件的项目，如 Go gRPC 项目。

### 问题背景

在使用 protoc 编译 Protocol Buffer 文件时，经常会遇到以下问题：

1. **版本冲突**: 系统安装的 protoc 插件版本与项目需求不匹配
2. **shim 冲突**: vman 的 shim 机制可能与 protoc 的插件发现机制冲突
3. **环境配置复杂**: 需要手动设置 PATH、启用代理、备份 shim 等多个步骤
4. **重复操作**: 每次编译都需要重复相同的环境配置步骤

### 一键解决方案

vman protoc 提供了四个核心命令来简化整个流程：

#### 1. 一键环境设置

```bash
# 一键设置 protoc 环境，包括启用代理、备份 shim、设置环境变量
vman protoc setup
```

这个命令会自动完成：
- 启用 vman 代理
- 智能备份 protoc shim 文件（避免冲突）
- 设置插件路径环境变量

#### 2. 在 protoc 模式下执行命令

```bash
# 在配置好的 protoc 环境中执行任意命令
vman protoc exec make api
vman protoc exec protoc --version
vman protoc exec make build
```

#### 3. 一键执行 make api

```bash
# 最简化的使用方式 - 在当前目录执行 make api
vman protoc make-api

# 指定项目目录
vman protoc make-api --dir /path/to/project
vman protoc make-api -d ~/go/src/myproject
```

#### 4. 查看环境状态

```bash
# 检查当前 protoc 环境状态
vman protoc status
```

### 使用场景

#### 场景1：Go gRPC 项目开发

```bash
# 安装需要的特定版本工具
vman install protoc-gen-go 1.31.0
vman install protoc-gen-go-grpc 1.3.0  
vman install protoc-gen-go-http 2.7.0

# 设置为使用这些版本
vman use protoc-gen-go 1.31.0
vman use protoc-gen-go-grpc 1.3.0
vman use protoc-gen-go-http 2.7.0

# 现在直接使用一键命令
cd /path/to/your/grpc-project
vman protoc make-api
```

#### 场景2：多项目管理

```bash
# 项目A - 使用较新版本
cd ~/projects/project-a
vman local protoc-gen-go 1.31.0
vman local protoc-gen-go-grpc 1.3.0
vman protoc make-api

# 项目B - 使用旧版本（向后兼容）
cd ~/projects/project-b  
vman local protoc-gen-go 1.28.0
vman local protoc-gen-go-grpc 1.2.0
vman protoc make-api
```

#### 场景3：CI/CD 集成

```bash
#!/bin/bash
# CI/CD 脚本示例
set -e

# 安装所需版本
vman install protoc-gen-go 1.31.0
vman install protoc-gen-go-grpc 1.3.0
vman install protoc-gen-go-http 2.7.0

# 一键编译
vman protoc make-api

# 继续其他构建步骤
go build ./...
go test ./...
```

### 高级配置

#### 项目配置文件

你可以在项目中创建 `.vmanrc` 文件来固定使用的 protoc 插件版本：

```yaml
# .vmanrc
version: "1.0"
tools:
  protoc-gen-go: "1.31.0"
  protoc-gen-go-grpc: "1.3.0"
  protoc-gen-go-http: "2.7.0"
  protoc: "21.12"

settings:
  auto_install: true
  
# protoc 特定配置
protoc:
  backup_shims: true
  auto_setup: true

environment:
  GOPATH: "/go"
  GO111MODULE: "on"
```

使用项目配置：

```bash
# 自动安装配置文件中指定的版本
vman install --from .vmanrc

# 一键执行（会自动使用配置的版本）
vman protoc make-api
```

### 故障排除

#### 常见问题

**1. protoc 插件找不到**

```bash
# 检查插件状态
vman protoc status

# 重新设置环境
vman protoc setup

# 检查是否正确安装了所需版本
vman list protoc-gen-go
```

**2. 版本冲突**

```bash
# 查看当前使用的版本
vman current protoc-gen-go
vman current protoc-gen-go-grpc

# 切换到正确版本
vman use protoc-gen-go 1.31.0
vman use protoc-gen-go-grpc 1.3.0

# 重新执行
vman protoc make-api
```

**3. 权限问题**

```bash
# 检查目录权限
ls -la ~/.vman/shims/

# 修复权限
chmod +x ~/.vman/shims/*
```

#### 调试模式

```bash
# 启用详细日志
export VMAN_LOG_LEVEL=debug
vman protoc setup
vman protoc make-api

# 查看详细执行过程
vman protoc exec --verbose make api
```

### 与传统方式对比

#### 传统方式（8个步骤）

```bash
# 1. 启用代理
vman proxy setup

# 2. 设置环境变量
export PATH="~/.vman/versions/protoc-gen-go/1.31.0:$PATH"
export PATH="~/.vman/versions/protoc-gen-go-grpc/1.3.0:$PATH" 

# 3. source 配置
source ~/.bashrc

# 4. 备份 protoc shim
mv ~/.vman/shims/protoc ~/.vman/shims/protoc.backup

# 5. 验证版本
protoc-gen-go --version
protoc-gen-go-grpc --version

# 6. 执行编译
make api

# 7. 恢复 shim
mv ~/.vman/shims/protoc.backup ~/.vman/shims/protoc

# 8. 清理环境
unset PATH
```

#### vman protoc 方式（1个命令）

```bash
# 一键搞定！
vman protoc make-api
```

### 最佳实践

1. **版本固定**: 在项目的 `.vmanrc` 文件中明确指定 protoc 插件版本
2. **团队协作**: 团队成员使用相同的配置文件，确保一致的编译环境
3. **CI/CD 集成**: 在构建脚本中使用 `vman protoc make-api` 确保一致性
4. **定期更新**: 定期检查和更新 protoc 插件版本，但要做好测试
5. **备份恢复**: vman 会自动处理 shim 的备份和恢复，无需手动管理

```

## 🎯 实际应用场景

### 场景一：Kubernetes 开发

```bash
# 设置 Kubernetes 开发环境
vman add kubectl
vman add helm
vman add kustomize

# 安装不同版本用于不同集群
vman install kubectl 1.28.0  # 生产集群
vman install kubectl 1.29.0  # 测试集群
vman install kubectl 1.30.0  # 开发集群

# 为不同项目设置不同版本
cd production-project
vman local kubectl 1.28.0

cd testing-project  
vman local kubectl 1.29.0

cd development-project
vman local kubectl 1.30.0
```

### 场景二：多云基础设施

```bash
# 添加云平台工具
vman add terraform
vman add aws-cli
vman add azure-cli
vman add gcloud

# 安装多个版本
vman install terraform 1.6.0 1.7.0
vman install aws-cli 2.13.0 2.14.0

# 项目配置
cat > .vmanrc << EOF
version: "1.0"
tools:
  terraform: "1.6.0"
  aws-cli: "2.13.0"
environment:
  AWS_PROFILE: "production"
  TF_VAR_environment: "prod"
EOF
```

### 场景三：数据库开发

```bash
# 添加数据库工具
vman add sqlc
vman add migrate
vman add pgcli

# 设置开发环境
vman install sqlc 1.20.0
vman local sqlc 1.20.0

# 在项目中使用
sqlc generate
sqlc compile
```

### 场景四：团队协作

#### 项目配置标准化

```yaml
# 团队项目的 .vmanrc
version: "1.0"
tools:
  kubectl: "1.28.0"
  terraform: "1.6.0"
  helm: "3.12.0"
  sqlc: "1.20.0"

settings:
  auto_install: true
  strict_mode: true

environment:
  KUBECONFIG: "./k8s/config"
  TF_VAR_project: "my-project"
```

#### CI/CD 集成

```bash
# 在 CI/CD 脚本中
#!/bin/bash
set -e

# 安装 vman
curl -fsSL https://get.vman.dev | bash

# 安装项目依赖的工具版本
vman install --from .vmanrc

# 运行构建任务
kubectl apply -f k8s/
terraform plan
```

## 💡 最佳实践

### 1. 版本管理策略

- **使用精确版本号**: 避免使用 `latest` 标签，使用具体版本号如 `1.28.0`
- **项目级版本**: 每个项目都应该有 `.vmanrc` 文件指定工具版本
- **定期更新**: 定期检查和更新工具版本，但要充分测试

### 2. 配置管理

```yaml
# 推荐的 .vmanrc 结构
version: "1.0"

# 明确指定工具版本
tools:
  kubectl: "1.28.0"
  terraform: "1.6.0"

# 项目特定设置
settings:
  auto_install: true
  strict_mode: false

# 环境变量
environment:
  KUBECONFIG: "./kubeconfig"
  
# 文档说明
metadata:
  description: "Development environment for microservices project"
  maintainer: "team@company.com"
  updated: "2024-01-15"
```

### 3. 团队协作

- **统一工具版本**: 团队成员使用相同的工具版本
- **版本控制**: 将 `.vmanrc` 加入版本控制
- **文档说明**: 在 README 中说明如何使用 vman

### 4. 性能优化

```bash
# 启用并发下载
vman config set download.concurrent_downloads 3

# 增加缓存时间
vman config set cache.ttl "48h"

# 启用压缩
vman config set download.compression true
```

### 5. 安全最佳实践

```bash
# 启用文件校验
vman config set download.verify_checksum true

# 使用 HTTPS
vman config set download.force_https true

# 定期更新工具源
vman update
```

## 🚨 故障排除

### 常见问题

#### 1. 命令找不到

```bash
# 检查 PATH 设置
echo $PATH | grep vman

# 重建符号链接
vman reshim

# 重新初始化 shell 集成
vman init bash >> ~/.bashrc
source ~/.bashrc
```

#### 2. 下载失败

```bash
# 检查网络连接
curl -I https://github.com

# 清理缓存重试
vman cleanup --cache
vman install kubectl 1.28.0

# 使用代理
export https_proxy=http://proxy.company.com:8080
vman install kubectl 1.28.0
```

#### 3. 版本冲突

```bash
# 检查版本来源
vman current kubectl --verbose

# 清理配置
rm .vmanrc
vman local kubectl --unset

# 重新设置
vman global kubectl 1.28.0
```

#### 4. 权限问题

```bash
# 检查目录权限
ls -la ~/.vman

# 修复权限
chmod -R 755 ~/.vman
chmod -R 644 ~/.vman/config.yaml
```

### 诊断工具

```bash
# 运行完整诊断
vman doctor

# 检查特定工具
vman doctor kubectl

# 验证配置
vman config validate

# 查看日志
tail -f ~/.vman/logs/vman.log
```

### 获取帮助

```bash
# 查看帮助信息
vman --help
vman install --help

# 查看版本信息
vman --version

# 报告问题
vman bug-report
```

---

## 📚 相关文档

- [安装指南](INSTALL.md)
- [配置参考](configuration.md)
- [架构设计](architecture.md)
- [贡献指南](CONTRIBUTING.md)
- [故障排除](troubleshooting.md)

如果本指南没有解决你的问题，请查看 [GitHub Issues](https://github.com/songzhibin97/vman/issues) 或创建新的问题报告。