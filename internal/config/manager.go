package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/songzhibin97/vman/pkg/types"
)

// Manager 配置管理器接口
type Manager interface {
	// LoadGlobal 加载全局配置
	LoadGlobal() (*types.GlobalConfig, error)

	// LoadProject 加载项目配置
	LoadProject(path string) (*types.ProjectConfig, error)

	// LoadToolConfig 加载工具配置
	LoadToolConfig(toolName string) (*types.ToolMetadata, error)

	// SaveGlobal 保存全局配置
	SaveGlobal(config *types.GlobalConfig) error

	// SaveProject 保存项目配置
	SaveProject(path string, config *types.ProjectConfig) error

	// GetEffectiveVersion 获取有效版本（合并全局和项目配置）
	GetEffectiveVersion(toolName, projectPath string) (string, error)

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

	// GetEffectiveConfig 获取有效配置（合并后）
	GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error)

	// CleanupOrphanedConfig 清理孤立的配置条目
	CleanupOrphanedConfig() error
}

// DefaultManager 默认配置管理器实现
type DefaultManager struct {
	fs        afero.Fs
	logger    *logrus.Logger
	paths     *types.ConfigPaths
	globalCfg *types.GlobalConfig
	viper     *viper.Viper
}

// NewManager 创建新的配置管理器
func NewManager(homeDir string) (Manager, error) {
	paths := types.DefaultConfigPaths(homeDir)
	logger := logrus.New()

	manager := &DefaultManager{
		fs:     afero.NewOsFs(),
		logger: logger,
		paths:  paths,
		viper:  viper.New(),
	}

	return manager, nil
}

// Initialize 初始化配置目录和文件
func (m *DefaultManager) Initialize() error {
	m.logger.Debug("Initializing configuration directories")

	// 创建所有必要的目录
	dirs := []string{
		m.paths.ConfigDir,
		m.paths.ToolsDir,
		m.paths.BinDir,
		m.paths.ShimsDir,
		m.paths.VersionsDir,
		m.paths.LogsDir,
		m.paths.CacheDir,
		m.paths.TempDir,
	}

	for _, dir := range dirs {
		if err := m.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		m.logger.Debugf("Created directory: %s", dir)
	}

	// 创建默认全局配置文件（如果不存在）
	if _, err := m.fs.Stat(m.paths.GlobalConfigFile); os.IsNotExist(err) {
		defaultConfig := types.GetDefaultGlobalConfig()
		if err := m.SaveGlobal(defaultConfig); err != nil {
			return fmt.Errorf("failed to create default global config: %w", err)
		}
		m.logger.Info("Created default global configuration")
	}

	return nil
}

// LoadGlobal 加载全局配置
func (m *DefaultManager) LoadGlobal() (*types.GlobalConfig, error) {
	m.logger.Debug("Loading global configuration")

	if m.globalCfg != nil {
		return m.globalCfg, nil
	}

	// 检查文件是否存在
	if _, err := m.fs.Stat(m.paths.GlobalConfigFile); os.IsNotExist(err) {
		m.logger.Warn("Global config file not found, using defaults")
		m.globalCfg = types.GetDefaultGlobalConfig()
		return m.globalCfg, nil
	}

	// 读取文件
	data, err := afero.ReadFile(m.fs, m.paths.GlobalConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file: %w", err)
	}

	// 解析YAML
	var config types.GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config file: %w", err)
	}

	// 应用默认值
	m.applyGlobalDefaults(&config)

	m.globalCfg = &config
	m.logger.Debug("Global configuration loaded successfully")
	return m.globalCfg, nil
}

// LoadProject 加载项目配置
func (m *DefaultManager) LoadProject(projectPath string) (*types.ProjectConfig, error) {
	m.logger.Debugf("Loading project configuration from: %s", projectPath)

	configPath := m.GetProjectConfigPath(projectPath)

	// 检查文件是否存在
	if _, err := m.fs.Stat(configPath); os.IsNotExist(err) {
		m.logger.Debug("Project config file not found, using defaults")
		return types.GetDefaultProjectConfig(), nil
	}

	// 读取文件
	data, err := afero.ReadFile(m.fs, configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read project config file: %w", err)
	}

	// 解析YAML
	var config types.ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse project config file: %w", err)
	}

	// 应用默认值
	m.applyProjectDefaults(&config)

	m.logger.Debug("Project configuration loaded successfully")
	return &config, nil
}

// LoadToolConfig 加载工具配置
func (m *DefaultManager) LoadToolConfig(toolName string) (*types.ToolMetadata, error) {
	m.logger.Debugf("Loading tool configuration for: %s", toolName)

	toolConfigPath := filepath.Join(m.paths.ToolsDir, toolName+".toml")

	// 检查文件是否存在
	if _, err := m.fs.Stat(toolConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("tool configuration not found for %s", toolName)
	}

	// 读取文件
	data, err := afero.ReadFile(m.fs, toolConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool config file: %w", err)
	}

	// 解析TOML
	var metadata types.ToolMetadata
	if err := toml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse tool config file: %w", err)
	}

	m.logger.Debug("Tool configuration loaded successfully")
	return &metadata, nil
}

// SaveGlobal 保存全局配置
func (m *DefaultManager) SaveGlobal(config *types.GlobalConfig) error {
	m.logger.Debug("Saving global configuration")

	// 序列化为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	// 写入文件
	if err := afero.WriteFile(m.fs, m.paths.GlobalConfigFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write global config file: %w", err)
	}

	// 更新缓存
	m.globalCfg = config

	m.logger.Debug("Global configuration saved successfully")
	return nil
}

// SaveProject 保存项目配置
func (m *DefaultManager) SaveProject(projectPath string, config *types.ProjectConfig) error {
	m.logger.Debugf("Saving project configuration to: %s", projectPath)

	configPath := m.GetProjectConfigPath(projectPath)

	// 确保目录存在
	if err := m.fs.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create project config directory: %w", err)
	}

	// 序列化为YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	// 写入文件
	if err := afero.WriteFile(m.fs, configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config file: %w", err)
	}

	m.logger.Debug("Project configuration saved successfully")
	return nil
}

// GetEffectiveVersion 获取有效版本（合并全局和项目配置）
func (m *DefaultManager) GetEffectiveVersion(toolName, projectPath string) (string, error) {
	m.logger.Debugf("Getting effective version for tool: %s, project: %s", toolName, projectPath)

	// 加载项目配置
	projectConfig, err := m.LoadProject(projectPath)
	if err != nil {
		return "", fmt.Errorf("failed to load project config: %w", err)
	}

	// 在项目配置中查找
	if version, exists := projectConfig.Tools[toolName]; exists && version != "" {
		// 验证版本是否真实存在
		if m.IsToolInstalled(toolName, version) {
			m.logger.Debugf("Found version %s for %s in project config", version, toolName)
			return version, nil
		}
		m.logger.Warnf("Tool %s version %s configured but not installed, ignoring", toolName, version)
	}

	// 加载全局配置
	globalConfig, err := m.LoadGlobal()
	if err != nil {
		return "", fmt.Errorf("failed to load global config: %w", err)
	}

	// 在全局配置中查找
	if version, exists := globalConfig.GlobalVersions[toolName]; exists && version != "" {
		// 验证版本是否真实存在
		if m.IsToolInstalled(toolName, version) {
			m.logger.Debugf("Found version %s for %s in global config", version, toolName)
			return version, nil
		}
		m.logger.Warnf("Tool %s version %s configured in global_versions but not installed, ignoring", toolName, version)
	}

	// 检查工具信息中的当前版本
	if toolInfo, exists := globalConfig.Tools[toolName]; exists && toolInfo.CurrentVersion != "" {
		// 验证版本是否真实存在
		if m.IsToolInstalled(toolName, toolInfo.CurrentVersion) {
			m.logger.Debugf("Found current version %s for %s in tool info", toolInfo.CurrentVersion, toolName)
			return toolInfo.CurrentVersion, nil
		}
		m.logger.Warnf("Tool %s current version %s configured but not installed, ignoring", toolName, toolInfo.CurrentVersion)
	}

	return "", fmt.Errorf("no version configured for tool %s", toolName)
}

// GetConfigDir 获取配置目录
func (m *DefaultManager) GetConfigDir() string {
	return m.paths.ConfigDir
}

// GetProjectConfigPath 获取项目配置文件路径
func (m *DefaultManager) GetProjectConfigPath(projectPath string) string {
	return filepath.Join(projectPath, ".vman.yaml")
}

// Validate 验证配置
func (m *DefaultManager) Validate() error {
	m.logger.Debug("Validating configuration")

	// 验证全局配置
	globalConfig, err := m.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config for validation: %w", err)
	}

	if err := m.validateGlobalConfig(globalConfig); err != nil {
		return fmt.Errorf("global config validation failed: %w", err)
	}

	m.logger.Debug("Configuration validation passed")
	return nil
}

// ListTools 列出所有已注册的工具
func (m *DefaultManager) ListTools() ([]string, error) {
	m.logger.Debug("Listing available tools")

	// 读取tools目录
	files, err := afero.ReadDir(m.fs, m.paths.ToolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}

	var tools []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".toml") {
			toolName := strings.TrimSuffix(file.Name(), ".toml")
			tools = append(tools, toolName)
		}
	}

	m.logger.Debugf("Found %d tools", len(tools))
	return tools, nil
}

// IsToolInstalled 检查工具是否已安装
func (m *DefaultManager) IsToolInstalled(toolName, version string) bool {
	m.logger.Debugf("Checking if tool %s version %s is installed", toolName, version)

	// 检查版本目录是否存在
	versionDir := filepath.Join(m.paths.VersionsDir, toolName, version)
	if _, err := m.fs.Stat(versionDir); os.IsNotExist(err) {
		return false
	}

	// 检查二进制文件是否存在（在bin子目录中）
	binaryPath := filepath.Join(versionDir, "bin", toolName)
	if _, err := m.fs.Stat(binaryPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetToolVersion 设置工具版本
func (m *DefaultManager) SetToolVersion(toolName, version string, global bool, projectPath string) error {
	m.logger.Debugf("Setting tool %s version to %s (global: %v, project: %s)", toolName, version, global, projectPath)

	// 验证版本是否真实安装（除非是system版本）
	if version != "system" && !m.IsToolInstalled(toolName, version) {
		return fmt.Errorf("cannot set version %s@%s: version not installed", toolName, version)
	}

	if global {
		// 设置全局版本
		globalConfig, err := m.LoadGlobal()
		if err != nil {
			return fmt.Errorf("failed to load global config: %w", err)
		}

		if globalConfig.GlobalVersions == nil {
			globalConfig.GlobalVersions = make(map[string]string)
		}
		globalConfig.GlobalVersions[toolName] = version

		// 更新工具信息
		if globalConfig.Tools == nil {
			globalConfig.Tools = make(map[string]types.ToolInfo)
		}
		toolInfo := globalConfig.Tools[toolName]
		toolInfo.CurrentVersion = version

		// 只为真实安装的版本添加到已安装版本列表
		if version != "system" {
			found := false
			for _, v := range toolInfo.InstalledVersions {
				if v == version {
					found = true
					break
				}
			}
			if !found {
				toolInfo.InstalledVersions = append(toolInfo.InstalledVersions, version)
			}
		}
		globalConfig.Tools[toolName] = toolInfo

		return m.SaveGlobal(globalConfig)
	} else {
		// 设置项目版本
		projectConfig, err := m.LoadProject(projectPath)
		if err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}

		if projectConfig.Tools == nil {
			projectConfig.Tools = make(map[string]string)
		}
		projectConfig.Tools[toolName] = version

		return m.SaveProject(projectPath, projectConfig)
	}
}

// RemoveToolVersion 移除工具版本
func (m *DefaultManager) RemoveToolVersion(toolName, version string) error {
	m.logger.Debugf("Removing tool %s version %s", toolName, version)

	// 加载全局配置
	globalConfig, err := m.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// 从已安装版本列表中移除
	if toolInfo, exists := globalConfig.Tools[toolName]; exists {
		var newVersions []string
		for _, v := range toolInfo.InstalledVersions {
			if v != version {
				newVersions = append(newVersions, v)
			}
		}
		toolInfo.InstalledVersions = newVersions

		// 如果移除的是当前版本，更新当前版本
		if toolInfo.CurrentVersion == version {
			if len(newVersions) > 0 {
				toolInfo.CurrentVersion = newVersions[len(newVersions)-1] // 使用最后一个版本
			} else {
				toolInfo.CurrentVersion = ""
			}
		}
		globalConfig.Tools[toolName] = toolInfo
	}

	// 从全局版本中移除（如果匹配）
	if globalConfig.GlobalVersions[toolName] == version {
		delete(globalConfig.GlobalVersions, toolName)
	}

	return m.SaveGlobal(globalConfig)
}

// CleanupOrphanedConfig 清理孤立的配置条目
func (m *DefaultManager) CleanupOrphanedConfig() error {
	m.logger.Debug("Cleaning up orphaned configuration entries")

	globalConfig, err := m.LoadGlobal()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	changed := false

	// 清理GlobalVersions中的孤立条目
	for toolName, version := range globalConfig.GlobalVersions {
		if !m.IsToolInstalled(toolName, version) {
			m.logger.Warnf("Removing orphaned global version: %s@%s", toolName, version)
			delete(globalConfig.GlobalVersions, toolName)
			changed = true
		}
	}

	// 清理Tools中的孤立条目
	for toolName, toolInfo := range globalConfig.Tools {
		// 过滤掉未安装的版本
		var validVersions []string
		for _, version := range toolInfo.InstalledVersions {
			if m.IsToolInstalled(toolName, version) {
				validVersions = append(validVersions, version)
			} else {
				m.logger.Warnf("Removing orphaned installed version: %s@%s", toolName, version)
				changed = true
			}
		}

		// 检查当前版本是否有效
		if toolInfo.CurrentVersion != "" && !m.IsToolInstalled(toolName, toolInfo.CurrentVersion) {
			m.logger.Warnf("Removing orphaned current version: %s@%s", toolName, toolInfo.CurrentVersion)
			toolInfo.CurrentVersion = ""
			changed = true
		}

		// 更新工具信息
		if len(validVersions) == 0 && toolInfo.CurrentVersion == "" {
			// 如果没有有效版本，删除整个工具条目
			m.logger.Warnf("Removing orphaned tool entry: %s", toolName)
			delete(globalConfig.Tools, toolName)
			changed = true
		} else {
			toolInfo.InstalledVersions = validVersions
			globalConfig.Tools[toolName] = toolInfo
		}
	}

	if changed {
		m.logger.Info("Saving cleaned up configuration")
		return m.SaveGlobal(globalConfig)
	}

	m.logger.Debug("No orphaned configuration entries found")
	return nil
}

// GetEffectiveConfig 获取有效配置（合并后）
func (m *DefaultManager) GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error) {
	m.logger.Debugf("Getting effective configuration for project: %s", projectPath)

	// 加载全局配置
	globalConfig, err := m.LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// 加载项目配置
	projectConfig, err := m.LoadProject(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load project config: %w", err)
	}

	// 合并版本配置
	resolvedVersions := make(map[string]string)
	configSource := make(map[string]string)

	// 先使用全局版本
	for toolName, version := range globalConfig.GlobalVersions {
		resolvedVersions[toolName] = version
		configSource[toolName] = "global"
	}

	// 项目配置覆盖全局配置
	for toolName, version := range projectConfig.Tools {
		resolvedVersions[toolName] = version
		configSource[toolName] = projectPath
	}

	return &types.EffectiveConfig{
		Global:           globalConfig,
		Project:          projectConfig,
		ResolvedVersions: resolvedVersions,
		ConfigSource:     configSource,
	}, nil
}

// 私有方法

// applyGlobalDefaults 应用全局配置默认值
func (m *DefaultManager) applyGlobalDefaults(config *types.GlobalConfig) {
	if config.Version == "" {
		config.Version = "1.0"
	}

	// 应用下载设置默认值
	if config.Settings.Download.Timeout == 0 {
		config.Settings.Download.Timeout = 300 * time.Second
	}
	if config.Settings.Download.Retries == 0 {
		config.Settings.Download.Retries = 3
	}
	if config.Settings.Download.ConcurrentDownloads == 0 {
		config.Settings.Download.ConcurrentDownloads = 2
	}

	// 应用日志设置默认值
	if config.Settings.Logging.Level == "" {
		config.Settings.Logging.Level = "info"
	}
	if config.Settings.Logging.File == "" {
		config.Settings.Logging.File = "~/.vman/logs/vman.log"
	}

	// 初始化映射（如果为nil）
	if config.GlobalVersions == nil {
		config.GlobalVersions = make(map[string]string)
	}
	if config.Tools == nil {
		config.Tools = make(map[string]types.ToolInfo)
	}
}

// applyProjectDefaults 应用项目配置默认值
func (m *DefaultManager) applyProjectDefaults(config *types.ProjectConfig) {
	if config.Version == "" {
		config.Version = "1.0"
	}
	if config.Tools == nil {
		config.Tools = make(map[string]string)
	}
}

// validateGlobalConfig 验证全局配置
func (m *DefaultManager) validateGlobalConfig(config *types.GlobalConfig) error {
	// 验证版本
	if config.Version == "" {
		return &types.ConfigValidationError{
			Field:   "version",
			Message: "version is required",
			Value:   config.Version,
		}
	}

	// 验证下载设置
	if config.Settings.Download.Retries < 0 {
		return &types.ConfigValidationError{
			Field:   "settings.download.retries",
			Message: "retries must be >= 0",
			Value:   config.Settings.Download.Retries,
		}
	}

	if config.Settings.Download.ConcurrentDownloads < 1 {
		return &types.ConfigValidationError{
			Field:   "settings.download.concurrent_downloads",
			Message: "concurrent_downloads must be >= 1",
			Value:   config.Settings.Download.ConcurrentDownloads,
		}
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[config.Settings.Logging.Level] {
		return &types.ConfigValidationError{
			Field:   "settings.logging.level",
			Message: "invalid log level, must be one of: debug, info, warn, error",
			Value:   config.Settings.Logging.Level,
		}
	}

	return nil
}
