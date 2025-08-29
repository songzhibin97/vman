# vman 项目架构设计文档

vman是一个通用命令行工具版本管理器，类似于asdf，旨在解决开发者在同一台计算机上管理和使用任意二进制程序（如kubectl、terraform、sqlc等）的多个版本时遇到的困难。

## 📋 目录

- [🎯 项目概述](#-项目概述)
- [🏗️ 整体架构](#️-整体架构)
- [🔧 技术栈选型](#-技术栈选型)
- [📦 核心模块设计](#-核心模块设计)
- [💾 数据存储设计](#-数据存储设计)
- [⚡ 性能设计](#-性能设计)
- [🔒 安全设计](#-安全设计)
- [🌐 扩展性设计](#-扩展性设计)
- [🔄 部署架构](#-部署架构)
- [📈 监控和运维](#-监控和运维)

## 🎯 项目概述

### 核心目标

- **通用性**：支持任意二进制工具的版本管理，不局限于特定语言生态
- **版本注册**：允许用户注册已下载的工具版本
- **软件注册**：配置软件下载源，支持自动下载
- **版本切换**：支持全局和项目级版本切换
- **透明使用**：拦截命令并代理到正确版本，用户无需改变使用习惯

### 设计原则

#### 简单易用
- 🎯 **用户体验优先**：API 设计简洁直观，命令符合直觉
- 🚀 **快速上手**：零配置开始，智能默认设置
- 📚 **文档完善**：提供完整的用户指南和示例

#### 可靠性
- 🔒 **数据安全**：配置和数据的完整性保障
- 🔄 **原子操作**：关键操作的事务性保证
- 🛡️ **容错设计**：优雅处理异常情况

#### 性能
- ⚡ **低延迟**：命令执行开销最小化
- 💾 **智能缓存**：多层缓存策略提升性能
- 🔄 **并发处理**：支持并发下载和安装

#### 扩展性
- 🔌 **模块化**：清晰的模块边界和接口设计
- 🔌 **插件支持**：预留插件接口用于功能扩展
- 🌐 **跨平台**：原生支持多个操作系统

## 2. 整体架构

### 2.1 架构模式

采用分层架构模式，从上到下分为：
- **CLI层**：用户交互界面
- **业务逻辑层**：核心功能实现
- **数据访问层**：配置和文件管理

### 2.2 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI 层                               │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐│
│  │ install │ │   use   │ │  list   │ │register │ │  ...    ││
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘│
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                      业务逻辑层                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐            │
│  │ 配置管理模块  │ │ 版本管理模块  │ │ 下载管理模块  │            │
│  └─────────────┘ └─────────────┘ └─────────────┘            │
│  ┌─────────────┐ ┌─────────────┐                            │
│  │ 命令代理模块  │ │ 存储管理模块  │                            │
│  └─────────────┘ └─────────────┘                            │
└─────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────────────────────────────────────────────┐
│                      数据访问层                             │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐            │
│  │  文件系统    │ │   配置文件   │ │    缓存     │            │
│  └─────────────┘ └─────────────┘ └─────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

## 3. 技术栈选型

### 3.1 编程语言：Go

**选择理由：**
- 编译为单一二进制文件，便于分发和安装
- 优秀的跨平台支持（Windows、macOS、Linux）
- 丰富的标准库，文件操作和网络请求支持良好
- 静态编译，无运行时依赖
- 优秀的并发支持，适合下载和IO密集型操作

### 3.2 配置文件格式

- **YAML**：用户友好的配置文件（全局配置、项目配置）
- **TOML**：严格的配置文件（工具元数据定义）

### 3.3 核心依赖

| 依赖库 | 版本 | 用途 |
|--------|------|------|
| cobra | v1.8+ | CLI框架 |
| viper | v1.17+ | 配置管理 |
| resty | v2.10+ | HTTP客户端 |
| afero | v1.10+ | 文件系统抽象 |
| logrus | v1.9+ | 结构化日志 |
| testify | v1.8+ | 测试框架 |
| semver | v3.2+ | 语义化版本处理 |

## 4. 核心模块设计

### 4.1 配置管理模块 (internal/config)

负责全局配置、项目配置的加载、保存和合并。

**核心接口：**
```go
type ConfigManager interface {
    LoadGlobal() (*GlobalConfig, error)
    LoadProject(path string) (*ProjectConfig, error)
    SaveGlobal(config *GlobalConfig) error
    SaveProject(path string, config *ProjectConfig) error
    GetEffectiveConfig(toolName string) (*ToolConfig, error)
}
```

**主要功能：**
- 全局配置文件管理
- 项目级配置文件管理
- 配置合并策略（项目配置覆盖全局配置）
- 配置验证和迁移

### 4.2 版本管理模块 (internal/version)

负责工具版本的注册、切换、查询等核心功能。

**核心接口：**
```go
type VersionManager interface {
    RegisterVersion(tool, version, path string) error
    ListVersions(tool string) ([]string, error)
    GetVersionPath(tool, version string) (string, error)
    RemoveVersion(tool, version string) error
    SetGlobalVersion(tool, version string) error
    SetLocalVersion(tool, version string) error
}
```

**主要功能：**
- 版本注册和注销
- 全局版本设置
- 项目级版本设置
- 版本查询和列表

### 4.3 下载管理模块 (internal/download)

负责工具的自动下载和安装。

**核心接口：**
```go
type DownloadManager interface {
    Download(tool, version string) error
    GetDownloadStrategy(tool string) (DownloadStrategy, error)
    AddSource(tool string, source DownloadSource) error
}

type DownloadStrategy interface {
    GetDownloadURL(version string) (string, error)
    ExtractArchive(archivePath, targetPath string) error
    GetLatestVersion() (string, error)
}
```

**下载策略：**
- GitHub Release策略
- 直接URL下载策略
- 自定义脚本策略

### 4.4 命令代理模块 (internal/proxy)

负责命令拦截和代理执行。

**核心接口：**
```go
type CommandProxy interface {
    InterceptCommand(cmd string, args []string) error
    ExecuteCommand(toolPath string, args []string) error
    GenerateShim(tool, version string) error
    RemoveShim(tool string) error
}
```

**主要功能：**
- 生成命令垫片（shim）
- 命令拦截和路由
- 参数透传
- 环境变量处理

### 4.5 存储管理模块 (internal/storage)

负责文件系统操作和目录管理。

**核心接口：**
```go
type StorageManager interface {
    GetToolsDir() string
    GetConfigDir() string
    GetShimsDir() string
    EnsureDirectories() error
    CleanupOrphaned() error
}
```

## 5. 数据存储设计

### 5.1 目录结构

```
~/.vman/                 # 用户主目录下的vman配置
├── config.yaml         # 全局配置文件
├── tools/              # 工具版本存储
│   ├── kubectl/
│   │   ├── 1.28.0/
│   │   └── 1.29.0/
│   └── terraform/
│       ├── 1.6.0/
│       └── 1.7.0/
├── shims/              # 命令垫片
├── sources/            # 下载源配置
└── cache/              # 缓存目录
```

### 5.2 配置文件格式

**全局配置 (config.yaml)：**
```yaml
version: "1.0"
settings:
  download:
    timeout: 300s
    retries: 3
  proxy:
    enabled: true
    shims_in_path: true
global_versions:
  kubectl: "1.28.0"
  terraform: "1.6.0"
```

**项目配置 (.vmanrc)：**
```yaml
version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.7.0"
```

**工具源定义 (sources/kubectl.toml)：**
```toml
[tool]
name = "kubectl"
description = "Kubernetes command-line tool"

[download]
type = "github"
repository = "kubernetes/kubernetes"
asset_pattern = "kubectl-{os}-{arch}"
```

## 6. 关键设计决策

### 6.1 配置优先级

1. 项目级配置 (.vmanrc)
2. 全局配置 (~/.vman/config.yaml)
3. 默认配置

### 6.2 版本解析策略

- 支持精确版本号（1.28.0）
- 支持语义化版本范围（~1.28, ^1.28）
- 支持别名（latest, stable）

### 6.3 命令代理机制

1. 生成轻量级shell脚本作为shim
2. shim脚本调用vman主程序
3. vman根据当前配置路由到正确版本

### 6.4 跨平台兼容性

- 使用Go的path/filepath包处理路径
- 动态检测操作系统和架构
- 适配不同平台的可执行文件格式

## 7. 性能考虑

### 7.1 启动性能

- 最小化依赖加载
- 延迟初始化非必要组件
- 缓存配置文件解析结果

### 7.2 下载性能

- 支持并发下载
- 断点续传支持
- 下载缓存机制

### 7.3 命令执行性能

- 最小化shim开销
- 避免重复配置解析
- 优化路径查找算法

## 8. 安全考虑

### 8.1 下载安全

- 验证下载文件完整性（SHA256）
- HTTPS强制要求
- 下载源白名单机制

### 8.2 执行安全

- 沙箱化工具执行
- 路径注入防护
- 权限最小化原则

## 9. 扩展性设计

### 9.1 插件架构

预留插件接口，支持：
- 自定义下载策略
- 自定义版本解析器
- 自定义后处理脚本

### 9.2 API接口

为未来的GUI或Web界面预留REST API接口。

## 10. 开发阶段规划

### 阶段1：核心功能
- 基础项目结构
- 配置管理
- 版本管理核心功能

### 阶段2：增强功能
- 下载管理
- 命令代理
- CLI界面完善

### 阶段3：优化完善
- 性能优化
- 测试覆盖
- 文档完善

---

本文档将随着项目开发进展持续更新和完善。