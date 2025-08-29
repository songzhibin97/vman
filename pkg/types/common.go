package types

import (
	"runtime"
	"time"
)

// GlobalConfig 全局配置结构
type GlobalConfig struct {
	Version        string              `yaml:"version"`
	Settings       Settings            `yaml:"settings"`
	GlobalVersions map[string]string   `yaml:"global_versions"`
	Tools          map[string]ToolInfo `yaml:"tools"`
}

// ProjectConfig 项目配置结构
type ProjectConfig struct {
	Version string            `yaml:"version"`
	Tools   map[string]string `yaml:"tools"`
}

// Settings 全局设置
type Settings struct {
	Download DownloadSettings `yaml:"download"`
	Proxy    ProxySettings    `yaml:"proxy"`
	Logging  LoggingSettings  `yaml:"logging"`
}

// DownloadSettings 下载设置
type DownloadSettings struct {
	Timeout             time.Duration `yaml:"timeout"`
	Retries             int           `yaml:"retries"`
	ConcurrentDownloads int           `yaml:"concurrent_downloads"`
}

// ProxySettings 代理设置
type ProxySettings struct {
	Enabled     bool `yaml:"enabled"`
	ShimsInPath bool `yaml:"shims_in_path"`
}

// LoggingSettings 日志设置
type LoggingSettings struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// ToolInfo 工具信息
type ToolInfo struct {
	CurrentVersion    string   `yaml:"current_version"`
	InstalledVersions []string `yaml:"installed_versions"`
}

// ToolMetadata 工具元数据
type ToolMetadata struct {
	Name           string         `toml:"name"`
	Description    string         `toml:"description"`
	Homepage       string         `toml:"homepage"`
	Repository     string         `toml:"repository"`
	DownloadConfig DownloadConfig `toml:"download"`
	VersionConfig  VersionConfig  `toml:"versions"`
	PostInstall    []string       `toml:"post_install,omitempty"`
}

// DownloadConfig 下载配置
type DownloadConfig struct {
	Type          string            `toml:"type"`
	Repository    string            `toml:"repository,omitempty"`
	AssetPattern  string            `toml:"asset_pattern,omitempty"`
	URLTemplate   string            `toml:"url_template,omitempty"`
	ExtractBinary string            `toml:"extract_binary,omitempty"`
	Headers       map[string]string `toml:"headers,omitempty"`
}

// VersionConfig 版本配置
type VersionConfig struct {
	Aliases     map[string]string  `toml:"aliases,omitempty"`
	Constraints VersionConstraints `toml:"constraints,omitempty"`
}

// VersionConstraints 版本约束
type VersionConstraints struct {
	MinVersion string `toml:"min_version,omitempty"`
	MaxVersion string `toml:"max_version,omitempty"`
}

// InstallStatus 安装状态
type InstallStatus struct {
	Installed   bool      `json:"installed"`
	InstalledAt time.Time `json:"installed_at,omitempty"`
	InstallPath string    `json:"install_path,omitempty"`
	BinaryPath  string    `json:"binary_path,omitempty"`
	Version     string    `json:"version"`
	Size        int64     `json:"size,omitempty"`
	Checksum    string    `json:"checksum,omitempty"`
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	Tools       map[string]*ToolMetadata `json:"tools"`
	LastUpdated time.Time                `json:"last_updated"`
	Version     string                   `json:"version"`
}

// ConfigError 配置错误
type ConfigError struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Field   string      `json:"field,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ConfigError) Error() string {
	return e.Message
}

// DownloadInfo 下载信息
type DownloadInfo struct {
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers,omitempty"`
	Checksum string            `json:"checksum,omitempty"`
	Size     int64             `json:"size,omitempty"`
	Filename string            `json:"filename"`
	Mirrors  []string          `json:"mirrors,omitempty"` // 镜像URL列表
	Method   string            `json:"method,omitempty"`  // HTTP方法，默认GET
}

// VersionInfo 版本详细信息
type VersionInfo struct {
	Version      string                  `json:"version"`
	ReleaseDate  string                  `json:"release_date,omitempty"`
	ChangeLog    string                  `json:"changelog,omitempty"`
	Downloads    map[string]DownloadInfo `json:"downloads"` // platform -> DownloadInfo
	IsPrerelease bool                    `json:"is_prerelease,omitempty"`
	IsStable     bool                    `json:"is_stable,omitempty"`
	Tags         []string                `json:"tags,omitempty"`
}

// PlatformInfo 平台信息
type PlatformInfo struct {
	OS   string `json:"os"`   // darwin, linux, windows
	Arch string `json:"arch"` // amd64, arm64, 386
}

// GetCurrentPlatform 获取当前平台信息
func GetCurrentPlatform() *PlatformInfo {
	return &PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// GetPlatformKey 获取平台键名
func (p *PlatformInfo) GetPlatformKey() string {
	return p.OS + "_" + p.Arch
}

// ProgressInfo 下载进度信息
type ProgressInfo struct {
	Total      int64   `json:"total"`      // 总字节数
	Downloaded int64   `json:"downloaded"` // 已下载字节数
	Percentage float64 `json:"percentage"` // 下载百分比
	Speed      int64   `json:"speed"`      // 下载速度 (字节/秒)
	ETA        int64   `json:"eta"`        // 预计剩余时间（秒）
	Status     string  `json:"status"`     // 状态信息
}

// DownloadInfo 下载信息

// InstallRequest 安装请求
type InstallRequest struct {
	ToolName    string `json:"tool_name"`
	Version     string `json:"version"`
	Force       bool   `json:"force,omitempty"`
	Global      bool   `json:"global,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// UninstallRequest 卸载请求
type UninstallRequest struct {
	ToolName    string `json:"tool_name"`
	Version     string `json:"version,omitempty"`
	All         bool   `json:"all,omitempty"`
	Global      bool   `json:"global,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// VersionMetadata 版本元数据
type VersionMetadata struct {
	Version     string    `json:"version"`
	ToolName    string    `json:"tool_name"`
	InstallPath string    `json:"install_path"`
	BinaryPath  string    `json:"binary_path"`
	InstalledAt time.Time `json:"installed_at"`
	InstallType string    `json:"install_type"` // "manual", "download", "build"
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum,omitempty"`
	Source      string    `json:"source,omitempty"` // 安装来源描述
}

// VersionRegistry 版本注册表
type VersionRegistry struct {
	Versions    map[string]*VersionMetadata `json:"versions"` // version -> metadata
	LastUpdated time.Time                   `json:"last_updated"`
}

// ToolVersionRegistry 工具版本注册表
type ToolVersionRegistry struct {
	ToolName       string                      `json:"tool_name"`
	CurrentVersion string                      `json:"current_version,omitempty"`
	Versions       map[string]*VersionMetadata `json:"versions"`
	LastUpdated    time.Time                   `json:"last_updated"`
}

// VersionSwitchContext 版本切换上下文
type VersionSwitchContext struct {
	ToolName    string    `json:"tool_name"`
	FromVersion string    `json:"from_version,omitempty"`
	ToVersion   string    `json:"to_version"`
	Scope       string    `json:"scope"` // "global" or "project"
	ProjectPath string    `json:"project_path,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
