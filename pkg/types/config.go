package types

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// LoadGlobal 加载全局配置
	LoadGlobal() (*GlobalConfig, error)

	// LoadProject 加载项目配置
	LoadProject(path string) (*ProjectConfig, error)

	// LoadToolConfig 加载工具配置
	LoadToolConfig(toolName string) (*ToolMetadata, error)

	// SaveGlobal 保存全局配置
	SaveGlobal(config *GlobalConfig) error

	// SaveProject 保存项目配置
	SaveProject(path string, config *ProjectConfig) error

	// GetEffectiveVersion 获取有效版本（合并全局和项目配置）
	GetEffectiveVersion(toolName string, projectPath string) (string, error)

	// GetConfigDir 获取配置目录
	GetConfigDir() string

	// GetProjectConfigPath 获取项目配置文件路径
	GetProjectConfigPath(projectPath string) string

	// Initialize 初始化配置
	Initialize() error

	// Validate 验证配置
	Validate() error

	// ListTools 列出所有已注册的工具
	ListTools() ([]string, error)

	// IsToolInstalled 检查工具是否已安装
	IsToolInstalled(toolName, version string) bool

	// SetToolVersion 设置工具版本
	SetToolVersion(toolName, version string, global bool, projectPath string) error

	// RemoveToolVersion 移除工具版本
	RemoveToolVersion(toolName, version string) error
}

// ConfigValidationError 配置验证错误
type ConfigValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConfigValidationError) Error() string {
	return e.Message
}

// ConfigPaths 配置路径结构
type ConfigPaths struct {
	// ConfigDir 配置根目录 (~/.vman)
	ConfigDir string

	// GlobalConfigFile 全局配置文件 (~/.vman/config.yaml)
	GlobalConfigFile string

	// ToolsDir 工具定义目录 (~/.vman/tools)
	ToolsDir string

	// BinDir 工具二进制目录 (~/.vman/bin)
	BinDir string

	// ShimsDir 工具shims目录 (~/.vman/shims)
	ShimsDir string

	// VersionsDir 版本存储目录 (~/.vman/versions)
	VersionsDir string

	// LogsDir 日志目录 (~/.vman/logs)
	LogsDir string

	// CacheDir 缓存目录 (~/.vman/cache)
	CacheDir string

	// TempDir 临时目录 (~/.vman/tmp)
	TempDir string
}

// DefaultConfigPaths 创建默认配置路径
func DefaultConfigPaths(homeDir string) *ConfigPaths {
	var configDir string
	
	// 根据操作系统确定配置目录
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/vman
		configDir = filepath.Join(homeDir, "Library", "Application Support", "vman")
	case "windows":
		// Windows: %APPDATA%/vman or ~/AppData/Local/vman
		if appData := os.Getenv("APPDATA"); appData != "" {
			configDir = filepath.Join(appData, "vman")
		} else {
			configDir = filepath.Join(homeDir, "AppData", "Local", "vman")
		}
	default:
		// Linux and other Unix-like: ~/.config/vman
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			configDir = filepath.Join(xdgConfig, "vman")
		} else {
			configDir = filepath.Join(homeDir, ".config", "vman")
		}
	}
	
	return &ConfigPaths{
		ConfigDir:        configDir,
		GlobalConfigFile: filepath.Join(configDir, "config.yaml"),
		ToolsDir:         filepath.Join(configDir, "tools"),
		BinDir:           filepath.Join(configDir, "bin"),
		ShimsDir:         filepath.Join(configDir, "shims"),
		VersionsDir:      filepath.Join(configDir, "versions"),
		LogsDir:          filepath.Join(configDir, "logs"),
		CacheDir:         filepath.Join(configDir, "cache"),
		TempDir:          filepath.Join(configDir, "tmp"),
	}
}

// GlobalConfigDefaults 全局配置默认值
type GlobalConfigDefaults struct {
	Version  string
	Settings SettingsDefaults
}

// SettingsDefaults 设置默认值
type SettingsDefaults struct {
	Download DownloadSettingsDefaults
	Proxy    ProxySettingsDefaults
	Logging  LoggingSettingsDefaults
}

// DownloadSettingsDefaults 下载设置默认值
type DownloadSettingsDefaults struct {
	Timeout             time.Duration
	Retries             int
	ConcurrentDownloads int
}

// ProxySettingsDefaults 代理设置默认值
type ProxySettingsDefaults struct {
	Enabled     bool
	ShimsInPath bool
}

// LoggingSettingsDefaults 日志设置默认值
type LoggingSettingsDefaults struct {
	Level string
	File  string
}

// GetDefaultGlobalConfig 获取默认全局配置
func GetDefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Version: "1.0",
		Settings: Settings{
			Download: DownloadSettings{
				Timeout:             300 * time.Second,
				Retries:             3,
				ConcurrentDownloads: 2,
			},
			Proxy: ProxySettings{
				Enabled:     true,
				ShimsInPath: true,
			},
			Logging: LoggingSettings{
				Level: "info",
				File:  "~/.vman/logs/vman.log",
			},
		},
		GlobalVersions: make(map[string]string),
		Tools:          make(map[string]ToolInfo),
	}
}

// GetDefaultProjectConfig 获取默认项目配置
func GetDefaultProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		Version: "1.0",
		Tools:   make(map[string]string),
	}
}

// ConfigMergeStrategy 配置合并策略
type ConfigMergeStrategy int

const (
	// OverrideStrategy 覆盖策略：项目配置覆盖全局配置
	OverrideStrategy ConfigMergeStrategy = iota
	// MergeStrategy 合并策略：将项目配置与全局配置合并
	MergeStrategy
	// IgnoreStrategy 忽略策略：只使用全局配置
	IgnoreStrategy
)

// EffectiveConfig 有效配置（合并后的配置）
type EffectiveConfig struct {
	// GlobalConfig 全局配置
	Global *GlobalConfig

	// ProjectConfig 项目配置
	Project *ProjectConfig

	// ResolvedVersions 解析后的版本映射
	ResolvedVersions map[string]string

	// ConfigSource 配置来源映射
	ConfigSource map[string]string // "global" or project path
}

// VersionResolution 版本解析结果
type VersionResolution struct {
	// ToolName 工具名称
	ToolName string

	// RequestedVersion 请求的版本
	RequestedVersion string

	// ResolvedVersion 解析后的版本
	ResolvedVersion string

	// Source 版本来源 ("global", "project", "alias", "constraint")
	Source string

	// IsInstalled 是否已安装
	IsInstalled bool

	// InstallPath 安装路径
	InstallPath string
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	// Type 变更类型
	Type ConfigChangeType

	// ConfigType 配置类型
	ConfigType string // "global", "project", "tool"

	// Key 变更的配置键
	Key string

	// OldValue 旧值
	OldValue interface{}

	// NewValue 新值
	NewValue interface{}

	// Timestamp 变更时间
	Timestamp time.Time
}

// ConfigChangeType 配置变更类型
type ConfigChangeType int

const (
	// ConfigAdded 配置添加
	ConfigAdded ConfigChangeType = iota
	// ConfigModified 配置修改
	ConfigModified
	// ConfigDeleted 配置删除
	ConfigDeleted
)

// String 返回配置变更类型的字符串表示
func (t ConfigChangeType) String() string {
	switch t {
	case ConfigAdded:
		return "added"
	case ConfigModified:
		return "modified"
	case ConfigDeleted:
		return "deleted"
	default:
		return "unknown"
	}
}
