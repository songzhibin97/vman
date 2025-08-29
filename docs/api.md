# vman API 文档

本文档描述了 vman 的内部 API 设计和接口规范，供开发者进行二次开发、插件开发或集成使用。

## 📋 目录

- [🏗️ 架构概述](#️-架构概述)
- [📦 核心接口](#-核心接口)
- [🔧 配置管理 API](#-配置管理-api)
- [📥 下载管理 API](#-下载管理-api)
- [🔄 版本管理 API](#-版本管理-api)
- [🔗 代理管理 API](#-代理管理-api)
- [💾 存储管理 API](#-存储管理-api)
- [🔌 插件系统 API](#-插件系统-api)
- [🌐 REST API](#-rest-api)

## 🏗️ 架构概述

vman 采用模块化架构，核心组件通过接口进行解耦：

```go
// 核心管理器接口
type VmanCore interface {
    ConfigManager() config.Manager
    VersionManager() version.Manager
    DownloadManager() download.Manager
    ProxyManager() proxy.Manager
    StorageManager() storage.Manager
}
```

## 📦 核心接口

### 配置管理接口

```go
package config

// Manager 配置管理器接口
type Manager interface {
    // LoadGlobal 加载全局配置
    LoadGlobal() (*GlobalConfig, error)
    
    // LoadProject 加载项目配置
    LoadProject(path string) (*ProjectConfig, error)
    
    // SaveGlobal 保存全局配置
    SaveGlobal(config *GlobalConfig) error
    
    // SaveProject 保存项目配置
    SaveProject(path string, config *ProjectConfig) error
    
    // GetEffectiveConfig 获取有效配置
    GetEffectiveConfig(toolName, projectPath string) (*EffectiveConfig, error)
    
    // Validate 验证配置
    Validate() error
}

// GlobalConfig 全局配置结构
type GlobalConfig struct {
    Version        string                 `yaml:"version"`
    Settings       Settings               `yaml:"settings"`
    GlobalVersions map[string]string      `yaml:"global_versions"`
    Tools          map[string]ToolInfo    `yaml:"tools"`
}

// ProjectConfig 项目配置结构  
type ProjectConfig struct {
    Version     string            `yaml:"version"`
    Tools       map[string]string `yaml:"tools"`
    Settings    Settings          `yaml:"settings"`
    Environment map[string]string `yaml:"environment"`
}
```

### 版本管理接口

```go
package version

// Manager 版本管理器接口
type Manager interface {
    // InstallVersion 安装版本
    InstallVersion(tool, version string) error
    
    // UninstallVersion 卸载版本
    UninstallVersion(tool, version string) error
    
    // ListVersions 列出已安装版本
    ListVersions(tool string) ([]VersionInfo, error)
    
    // GetCurrentVersion 获取当前版本
    GetCurrentVersion(tool, projectPath string) (string, error)
    
    // SetGlobalVersion 设置全局版本
    SetGlobalVersion(tool, version string) error
    
    // SetLocalVersion 设置本地版本
    SetLocalVersion(tool, version, projectPath string) error
}

// VersionInfo 版本信息
type VersionInfo struct {
    Tool        string    `json:"tool"`
    Version     string    `json:"version"`
    Path        string    `json:"path"`
    InstallTime time.Time `json:"install_time"`
    Size        int64     `json:"size"`
}
```

### 下载管理接口

```go
package download

// Manager 下载管理器接口
type Manager interface {
    // Download 下载工具版本
    Download(tool, version string) error
    
    // GetAvailableVersions 获取可用版本列表
    GetAvailableVersions(tool string) ([]string, error)
    
    // GetLatestVersion 获取最新版本
    GetLatestVersion(tool string) (string, error)
    
    // AddSource 添加下载源
    AddSource(tool string, source Source) error
    
    // RemoveSource 移除下载源
    RemoveSource(tool string) error
}

// Strategy 下载策略接口
type Strategy interface {
    // GetDownloadURL 获取下载URL
    GetDownloadURL(version string) (string, error)
    
    // GetVersions 获取所有版本
    GetVersions() ([]string, error)
    
    // ValidateVersion 验证版本格式
    ValidateVersion(version string) error
}
```

## 🔧 配置管理 API

### 配置操作

```go
// 使用示例
func ExampleConfigUsage() {
    // 创建配置管理器
    configMgr := config.NewManager()
    
    // 加载全局配置
    globalConfig, err := configMgr.LoadGlobal()
    if err != nil {
        log.Fatal(err)
    }
    
    // 设置全局版本
    globalConfig.GlobalVersions["kubectl"] = "1.28.0"
    
    // 保存配置
    err = configMgr.SaveGlobal(globalConfig)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 配置验证

```go
// Validator 配置验证器
type Validator interface {
    // ValidateGlobal 验证全局配置
    ValidateGlobal(config *GlobalConfig) []ValidationError
    
    // ValidateProject 验证项目配置
    ValidateProject(config *ProjectConfig) []ValidationError
    
    // ValidateTool 验证工具配置
    ValidateTool(tool ToolConfig) []ValidationError
}

// ValidationError 验证错误
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
}
```

## 📥 下载管理 API

### 下载策略

```go
// GitHubStrategy GitHub Release 下载策略
type GitHubStrategy struct {
    Repository    string `toml:"repository"`
    AssetPattern  string `toml:"asset_pattern"`
    PreRelease    bool   `toml:"pre_release"`
}

func (g *GitHubStrategy) GetDownloadURL(version string) (string, error) {
    // 实现 GitHub Release 下载逻辑
}

// DirectStrategy 直接下载策略
type DirectStrategy struct {
    URLTemplate string `toml:"url_template"`
    ChecksumURL string `toml:"checksum_url"`
}

func (d *DirectStrategy) GetDownloadURL(version string) (string, error) {
    // 实现直接下载逻辑
}
```

### 下载任务管理

```go
// Task 下载任务
type Task struct {
    ID          string    `json:"id"`
    Tool        string    `json:"tool"`
    Version     string    `json:"version"`
    URL         string    `json:"url"`
    Progress    float64   `json:"progress"`
    Status      Status    `json:"status"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Error       string    `json:"error,omitempty"`
}

// TaskManager 任务管理器
type TaskManager interface {
    // Submit 提交下载任务
    Submit(tool, version string) (*Task, error)
    
    // Cancel 取消任务
    Cancel(taskID string) error
    
    // GetStatus 获取任务状态
    GetStatus(taskID string) (*Task, error)
    
    // ListTasks 列出所有任务
    ListTasks() ([]*Task, error)
}
```

## 🔄 版本管理 API

### 版本解析

```go
// Resolver 版本解析器
type Resolver interface {
    // ResolveVersion 解析版本
    ResolveVersion(tool, version, projectPath string) (string, error)
    
    // GetSource 获取版本来源
    GetSource(tool, projectPath string) (Source, error)
}

// Source 版本来源
type Source int

const (
    SourceGlobal Source = iota
    SourceLocal
    SourceProject
    SourceEnvironment
)
```

### 版本比较

```go
// Comparator 版本比较器
type Comparator interface {
    // Compare 比较版本
    Compare(v1, v2 string) (int, error)
    
    // IsValid 验证版本格式
    IsValid(version string) bool
    
    // Normalize 标准化版本
    Normalize(version string) (string, error)
}
```

## 🔗 代理管理 API

### 命令代理

```go
package proxy

// Manager 代理管理器
type Manager interface {
    // InterceptCommand 拦截命令
    InterceptCommand(cmd string, args []string) error
    
    // ExecuteCommand 执行命令
    ExecuteCommand(toolPath string, args []string) error
    
    // GenerateShim 生成垫片
    GenerateShim(tool, version string) error
    
    // RemoveShim 移除垫片
    RemoveShim(tool string) error
    
    // UpdateShims 更新所有垫片
    UpdateShims() error
}

// Router 命令路由器
type Router interface {
    // Route 路由命令
    Route(cmd string, projectPath string) (toolPath string, err error)
    
    // Register 注册工具
    Register(tool, path string) error
    
    // Unregister 注销工具
    Unregister(tool string) error
}
```

### 环境管理

```go
// EnvironmentManager 环境管理器
type EnvironmentManager interface {
    // SetupEnvironment 设置环境
    SetupEnvironment(tool, version string) (map[string]string, error)
    
    // CleanupEnvironment 清理环境
    CleanupEnvironment(tool string) error
    
    // GetEnvironment 获取环境变量
    GetEnvironment(tool, projectPath string) (map[string]string, error)
}
```

## 💾 存储管理 API

### 文件系统操作

```go
package storage

// Manager 存储管理器
type Manager interface {
    // GetToolPath 获取工具路径
    GetToolPath(tool, version string) string
    
    // EnsureDirectories 确保目录存在
    EnsureDirectories() error
    
    // CleanupOrphaned 清理孤立文件
    CleanupOrphaned() error
    
    // GetDiskUsage 获取磁盘使用情况
    GetDiskUsage() (*DiskUsage, error)
}

// DiskUsage 磁盘使用情况
type DiskUsage struct {
    Total     int64            `json:"total"`
    Used      int64            `json:"used"`
    Available int64            `json:"available"`
    ByTool    map[string]int64 `json:"by_tool"`
}
```

### 缓存管理

```go
// CacheManager 缓存管理器
type CacheManager interface {
    // Get 获取缓存
    Get(key string) (interface{}, bool)
    
    // Set 设置缓存
    Set(key string, value interface{}, ttl time.Duration)
    
    // Delete 删除缓存
    Delete(key string)
    
    // Clear 清空缓存
    Clear()
    
    // Stats 获取缓存统计
    Stats() CacheStats
}

// CacheStats 缓存统计
type CacheStats struct {
    Hits     int64 `json:"hits"`
    Misses   int64 `json:"misses"`
    Size     int64 `json:"size"`
    Entries  int   `json:"entries"`
}
```

## 🔌 插件系统 API

### 插件接口

```go
package plugin

// Plugin 插件接口
type Plugin interface {
    // Name 插件名称
    Name() string
    
    // Version 插件版本
    Version() string
    
    // Initialize 初始化插件
    Initialize(ctx Context) error
    
    // Execute 执行插件
    Execute(ctx Context, args []string) error
    
    // Cleanup 清理插件
    Cleanup() error
}

// Context 插件上下文
type Context interface {
    // GetVmanCore 获取 vman 核心接口
    GetVmanCore() VmanCore
    
    // GetLogger 获取日志器
    GetLogger() Logger
    
    // GetConfig 获取配置
    GetConfig() map[string]interface{}
}
```

### 插件管理器

```go
// Manager 插件管理器
type Manager interface {
    // LoadPlugin 加载插件
    LoadPlugin(path string) (Plugin, error)
    
    // UnloadPlugin 卸载插件
    UnloadPlugin(name string) error
    
    // ListPlugins 列出插件
    ListPlugins() []PluginInfo
    
    // ExecutePlugin 执行插件
    ExecutePlugin(name string, args []string) error
}

// PluginInfo 插件信息
type PluginInfo struct {
    Name        string    `json:"name"`
    Version     string    `json:"version"`
    Description string    `json:"description"`
    Enabled     bool      `json:"enabled"`
    LoadTime    time.Time `json:"load_time"`
}
```

## 🌐 REST API

vman 提供 REST API 用于远程管理和集成：

### 启动 API 服务器

```bash
# 启动 API 服务器
vman server --port 8080 --host 0.0.0.0

# 使用 TLS
vman server --port 8443 --tls-cert cert.pem --tls-key key.pem
```

### API 端点

#### 工具管理

```http
# 获取所有工具
GET /api/v1/tools

# 获取工具信息
GET /api/v1/tools/{tool}

# 添加工具
POST /api/v1/tools
Content-Type: application/json

{
  "name": "kubectl",
  "source": {
    "type": "github",
    "repository": "kubernetes/kubernetes"
  }
}
```

#### 版本管理

```http
# 获取工具版本列表
GET /api/v1/tools/{tool}/versions

# 安装版本
POST /api/v1/tools/{tool}/versions/{version}/install

# 卸载版本
DELETE /api/v1/tools/{tool}/versions/{version}

# 设置全局版本
PUT /api/v1/tools/{tool}/global
Content-Type: application/json

{
  "version": "1.28.0"
}
```

#### 配置管理

```http
# 获取全局配置
GET /api/v1/config/global

# 更新全局配置
PUT /api/v1/config/global
Content-Type: application/json

{
  "settings": {
    "download": {
      "timeout": "300s"
    }
  }
}
```

### 错误处理

API 使用标准 HTTP 状态码和统一的错误格式：

```json
{
  "error": {
    "code": "TOOL_NOT_FOUND",
    "message": "Tool 'invalid-tool' not found",
    "details": {
      "tool": "invalid-tool"
    }
  }
}
```

---

## 📚 相关文档

- [用户指南](user-guide.md)
- [架构设计](architecture.md)
- [贡献指南](CONTRIBUTING.md)
- [插件开发指南](plugin-development.md)

如需更多信息，请查看源码中的接口定义和示例。