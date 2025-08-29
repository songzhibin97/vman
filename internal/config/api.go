package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

// API 配置管理API接口
type API interface {
	// 初始化相关
	Init(ctx context.Context) error
	Reset(ctx context.Context) error
	Backup(ctx context.Context, backupPath string) error
	Restore(ctx context.Context, backupPath string) error

	// 全局配置管理
	GetGlobalConfig(ctx context.Context) (*types.GlobalConfig, error)
	UpdateGlobalConfig(ctx context.Context, config *types.GlobalConfig) error
	SetGlobalSetting(ctx context.Context, key string, value interface{}) error
	GetGlobalSetting(ctx context.Context, key string) (interface{}, error)

	// 项目配置管理
	GetProjectConfig(ctx context.Context, projectPath string) (*types.ProjectConfig, error)
	UpdateProjectConfig(ctx context.Context, projectPath string, config *types.ProjectConfig) error
	CreateProjectConfig(ctx context.Context, projectPath string) error
	DeleteProjectConfig(ctx context.Context, projectPath string) error

	// 工具管理
	ListTools(ctx context.Context) ([]string, error)
	GetToolConfig(ctx context.Context, toolName string) (*types.ToolMetadata, error)
	RegisterTool(ctx context.Context, metadata *types.ToolMetadata) error
	UnregisterTool(ctx context.Context, toolName string) error

	// 版本管理
	GetEffectiveVersion(ctx context.Context, toolName, projectPath string) (string, error)
	SetToolVersion(ctx context.Context, toolName, version string, global bool, projectPath string) error
	RemoveToolVersion(ctx context.Context, toolName, version string) error
	ListInstalledVersions(ctx context.Context, toolName string) ([]string, error)

	// 配置查询
	GetEffectiveConfig(ctx context.Context, projectPath string) (*types.EffectiveConfig, error)
	ValidateConfig(ctx context.Context) error
	GetConfigPaths(ctx context.Context) (*types.ConfigPaths, error)

	// 事件和监听
	Watch(ctx context.Context, callback func(*types.ConfigChangeEvent)) error
	StopWatch(ctx context.Context) error
}

// DefaultAPI 默认配置管理API实现
type DefaultAPI struct {
	manager   Manager
	merger    Merger
	validator Validator
	logger    *logrus.Logger
	fs        afero.Fs
	paths     *types.ConfigPaths
	watchers  map[string]func(*types.ConfigChangeEvent)
}

// NewAPI 创建新的配置管理API
func NewAPI(homeDir string) (API, error) {
	manager, err := NewManager(homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	merger := NewMerger()
	validator := NewValidator()
	paths := types.DefaultConfigPaths(homeDir)

	return &DefaultAPI{
		manager:   manager,
		merger:    merger,
		validator: validator,
		logger:    logrus.New(),
		fs:        afero.NewOsFs(),
		paths:     paths,
		watchers:  make(map[string]func(*types.ConfigChangeEvent)),
	}, nil
}

// Init 初始化配置
func (api *DefaultAPI) Init(ctx context.Context) error {
	api.logger.Info("Initializing vman configuration")

	if err := api.manager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	api.logger.Info("Configuration initialized successfully")
	return nil
}

// Reset 重置配置
func (api *DefaultAPI) Reset(ctx context.Context) error {
	api.logger.Warn("Resetting vman configuration")

	// 备份现有配置
	backupPath := filepath.Join(api.paths.ConfigDir, fmt.Sprintf("backup_%d", time.Now().Unix()))
	if err := api.Backup(ctx, backupPath); err != nil {
		api.logger.Warnf("Failed to backup configuration before reset: %v", err)
	}

	// 删除配置目录
	if err := api.fs.RemoveAll(api.paths.ConfigDir); err != nil {
		return fmt.Errorf("failed to remove config directory: %w", err)
	}

	// 重新初始化
	if err := api.Init(ctx); err != nil {
		return fmt.Errorf("failed to reinitialize after reset: %w", err)
	}

	api.logger.Info("Configuration reset completed")
	return nil
}

// Backup 备份配置
func (api *DefaultAPI) Backup(ctx context.Context, backupPath string) error {
	api.logger.Infof("Backing up configuration to: %s", backupPath)

	// 创建备份目录
	if err := api.fs.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 复制配置文件
	files := []string{
		api.paths.GlobalConfigFile,
	}

	for _, file := range files {
		if _, err := api.fs.Stat(file); err == nil {
			destPath := filepath.Join(backupPath, filepath.Base(file))
			if err := api.copyFile(file, destPath); err != nil {
				return fmt.Errorf("failed to backup file %s: %w", file, err)
			}
		}
	}

	// 复制工具配置目录
	if _, err := api.fs.Stat(api.paths.ToolsDir); err == nil {
		destToolsDir := filepath.Join(backupPath, "tools")
		if err := api.copyDir(api.paths.ToolsDir, destToolsDir); err != nil {
			return fmt.Errorf("failed to backup tools directory: %w", err)
		}
	}

	api.logger.Info("Configuration backup completed")
	return nil
}

// Restore 恢复配置
func (api *DefaultAPI) Restore(ctx context.Context, backupPath string) error {
	api.logger.Infof("Restoring configuration from: %s", backupPath)

	// 检查备份目录是否存在
	if _, err := api.fs.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory does not exist: %s", backupPath)
	}

	// 恢复全局配置文件
	globalConfigBackup := filepath.Join(backupPath, "config.yaml")
	if _, err := api.fs.Stat(globalConfigBackup); err == nil {
		if err := api.copyFile(globalConfigBackup, api.paths.GlobalConfigFile); err != nil {
			return fmt.Errorf("failed to restore global config: %w", err)
		}
	}

	// 恢复工具配置目录
	toolsBackupDir := filepath.Join(backupPath, "tools")
	if _, err := api.fs.Stat(toolsBackupDir); err == nil {
		// 先删除现有的工具配置目录
		if err := api.fs.RemoveAll(api.paths.ToolsDir); err != nil {
			return fmt.Errorf("failed to remove existing tools directory: %w", err)
		}

		if err := api.copyDir(toolsBackupDir, api.paths.ToolsDir); err != nil {
			return fmt.Errorf("failed to restore tools directory: %w", err)
		}
	}

	api.logger.Info("Configuration restore completed")
	return nil
}

// GetGlobalConfig 获取全局配置
func (api *DefaultAPI) GetGlobalConfig(ctx context.Context) (*types.GlobalConfig, error) {
	return api.manager.LoadGlobal()
}

// UpdateGlobalConfig 更新全局配置
func (api *DefaultAPI) UpdateGlobalConfig(ctx context.Context, config *types.GlobalConfig) error {
	// 验证配置
	if err := api.validator.ValidateGlobalConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 保存配置
	if err := api.manager.SaveGlobal(config); err != nil {
		return fmt.Errorf("failed to save global config: %w", err)
	}

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigModified,
		ConfigType: "global",
		Key:        "global_config",
		NewValue:   config,
		Timestamp:  time.Now(),
	})

	return nil
}

// SetGlobalSetting 设置全局设置
func (api *DefaultAPI) SetGlobalSetting(ctx context.Context, key string, value interface{}) error {
	config, err := api.GetGlobalConfig(ctx)
	if err != nil {
		return err
	}

	oldValue := api.getSettingValue(config, key)

	// 根据key设置对应的值
	if err := api.setSettingValue(config, key, value); err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}

	// 保存配置
	if err := api.UpdateGlobalConfig(ctx, config); err != nil {
		return err
	}

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigModified,
		ConfigType: "global",
		Key:        key,
		OldValue:   oldValue,
		NewValue:   value,
		Timestamp:  time.Now(),
	})

	return nil
}

// GetGlobalSetting 获取全局设置
func (api *DefaultAPI) GetGlobalSetting(ctx context.Context, key string) (interface{}, error) {
	config, err := api.GetGlobalConfig(ctx)
	if err != nil {
		return nil, err
	}

	return api.getSettingValue(config, key), nil
}

// GetProjectConfig 获取项目配置
func (api *DefaultAPI) GetProjectConfig(ctx context.Context, projectPath string) (*types.ProjectConfig, error) {
	return api.manager.LoadProject(projectPath)
}

// UpdateProjectConfig 更新项目配置
func (api *DefaultAPI) UpdateProjectConfig(ctx context.Context, projectPath string, config *types.ProjectConfig) error {
	// 验证配置
	if err := api.validator.ValidateProjectConfig(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 保存配置
	if err := api.manager.SaveProject(projectPath, config); err != nil {
		return fmt.Errorf("failed to save project config: %w", err)
	}

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigModified,
		ConfigType: "project",
		Key:        projectPath,
		NewValue:   config,
		Timestamp:  time.Now(),
	})

	return nil
}

// CreateProjectConfig 创建项目配置
func (api *DefaultAPI) CreateProjectConfig(ctx context.Context, projectPath string) error {
	config := types.GetDefaultProjectConfig()
	return api.UpdateProjectConfig(ctx, projectPath, config)
}

// DeleteProjectConfig 删除项目配置
func (api *DefaultAPI) DeleteProjectConfig(ctx context.Context, projectPath string) error {
	configPath := api.manager.GetProjectConfigPath(projectPath)

	if err := api.fs.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete project config: %w", err)
	}

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigDeleted,
		ConfigType: "project",
		Key:        projectPath,
		Timestamp:  time.Now(),
	})

	return nil
}

// ListTools 列出所有工具
func (api *DefaultAPI) ListTools(ctx context.Context) ([]string, error) {
	return api.manager.ListTools()
}

// GetToolConfig 获取工具配置
func (api *DefaultAPI) GetToolConfig(ctx context.Context, toolName string) (*types.ToolMetadata, error) {
	return api.manager.LoadToolConfig(toolName)
}

// RegisterTool 注册工具
func (api *DefaultAPI) RegisterTool(ctx context.Context, metadata *types.ToolMetadata) error {
	// 验证工具元数据
	if err := api.validator.ValidateToolMetadata(metadata); err != nil {
		return fmt.Errorf("tool metadata validation failed: %w", err)
	}

	// 这里应该将metadata序列化为TOML并保存
	// 为了简化，我们先记录日志
	api.logger.Infof("Registering tool: %s", metadata.Name)

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigAdded,
		ConfigType: "tool",
		Key:        metadata.Name,
		NewValue:   metadata,
		Timestamp:  time.Now(),
	})

	return nil
}

// UnregisterTool 取消注册工具
func (api *DefaultAPI) UnregisterTool(ctx context.Context, toolName string) error {
	toolConfigPath := filepath.Join(api.paths.ToolsDir, toolName+".toml")

	if err := api.fs.Remove(toolConfigPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove tool config: %w", err)
	}

	// 触发配置变更事件
	api.notifyConfigChange(&types.ConfigChangeEvent{
		Type:       types.ConfigDeleted,
		ConfigType: "tool",
		Key:        toolName,
		Timestamp:  time.Now(),
	})

	return nil
}

// GetEffectiveVersion 获取有效版本
func (api *DefaultAPI) GetEffectiveVersion(ctx context.Context, toolName, projectPath string) (string, error) {
	return api.manager.GetEffectiveVersion(toolName, projectPath)
}

// SetToolVersion 设置工具版本
func (api *DefaultAPI) SetToolVersion(ctx context.Context, toolName, version string, global bool, projectPath string) error {
	return api.manager.SetToolVersion(toolName, version, global, projectPath)
}

// RemoveToolVersion 移除工具版本
func (api *DefaultAPI) RemoveToolVersion(ctx context.Context, toolName, version string) error {
	return api.manager.RemoveToolVersion(toolName, version)
}

// ListInstalledVersions 列出已安装的版本
func (api *DefaultAPI) ListInstalledVersions(ctx context.Context, toolName string) ([]string, error) {
	config, err := api.GetGlobalConfig(ctx)
	if err != nil {
		return nil, err
	}

	if toolInfo, exists := config.Tools[toolName]; exists {
		return toolInfo.InstalledVersions, nil
	}

	return []string{}, nil
}

// GetEffectiveConfig 获取有效配置
func (api *DefaultAPI) GetEffectiveConfig(ctx context.Context, projectPath string) (*types.EffectiveConfig, error) {
	return api.manager.GetEffectiveConfig(projectPath)
}

// ValidateConfig 验证配置
func (api *DefaultAPI) ValidateConfig(ctx context.Context) error {
	return api.manager.Validate()
}

// GetConfigPaths 获取配置路径
func (api *DefaultAPI) GetConfigPaths(ctx context.Context) (*types.ConfigPaths, error) {
	return api.paths, nil
}

// Watch 监听配置变更
func (api *DefaultAPI) Watch(ctx context.Context, callback func(*types.ConfigChangeEvent)) error {
	watcherID := fmt.Sprintf("watcher_%d", time.Now().UnixNano())
	api.watchers[watcherID] = callback
	api.logger.Debugf("Added config watcher: %s", watcherID)
	return nil
}

// StopWatch 停止监听配置变更
func (api *DefaultAPI) StopWatch(ctx context.Context) error {
	api.watchers = make(map[string]func(*types.ConfigChangeEvent))
	api.logger.Debug("Stopped all config watchers")
	return nil
}

// 私有辅助方法

// copyFile 复制文件
func (api *DefaultAPI) copyFile(src, dst string) error {
	data, err := afero.ReadFile(api.fs, src)
	if err != nil {
		return err
	}
	return afero.WriteFile(api.fs, dst, data, 0644)
}

// copyDir 复制目录
func (api *DefaultAPI) copyDir(src, dst string) error {
	return afero.Walk(api.fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return api.fs.MkdirAll(dstPath, info.Mode())
		}

		return api.copyFile(path, dstPath)
	})
}

// notifyConfigChange 通知配置变更
func (api *DefaultAPI) notifyConfigChange(event *types.ConfigChangeEvent) {
	for _, callback := range api.watchers {
		go callback(event)
	}
}

// getSettingValue 获取设置值
func (api *DefaultAPI) getSettingValue(config *types.GlobalConfig, key string) interface{} {
	// 根据key返回对应的设置值
	switch key {
	case "download.timeout":
		return config.Settings.Download.Timeout
	case "download.retries":
		return config.Settings.Download.Retries
	case "download.concurrent_downloads":
		return config.Settings.Download.ConcurrentDownloads
	case "proxy.enabled":
		return config.Settings.Proxy.Enabled
	case "proxy.shims_in_path":
		return config.Settings.Proxy.ShimsInPath
	case "logging.level":
		return config.Settings.Logging.Level
	case "logging.file":
		return config.Settings.Logging.File
	default:
		return nil
	}
}

// setSettingValue 设置设置值
func (api *DefaultAPI) setSettingValue(config *types.GlobalConfig, key string, value interface{}) error {
	// 根据key设置对应的值
	switch key {
	case "download.timeout":
		if timeout, ok := value.(time.Duration); ok {
			config.Settings.Download.Timeout = timeout
		} else {
			return fmt.Errorf("invalid type for download.timeout, expected time.Duration")
		}
	case "download.retries":
		if retries, ok := value.(int); ok {
			config.Settings.Download.Retries = retries
		} else {
			return fmt.Errorf("invalid type for download.retries, expected int")
		}
	case "download.concurrent_downloads":
		if concurrent, ok := value.(int); ok {
			config.Settings.Download.ConcurrentDownloads = concurrent
		} else {
			return fmt.Errorf("invalid type for download.concurrent_downloads, expected int")
		}
	case "proxy.enabled":
		if enabled, ok := value.(bool); ok {
			config.Settings.Proxy.Enabled = enabled
		} else {
			return fmt.Errorf("invalid type for proxy.enabled, expected bool")
		}
	case "proxy.shims_in_path":
		if shims, ok := value.(bool); ok {
			config.Settings.Proxy.ShimsInPath = shims
		} else {
			return fmt.Errorf("invalid type for proxy.shims_in_path, expected bool")
		}
	case "logging.level":
		if level, ok := value.(string); ok {
			config.Settings.Logging.Level = level
		} else {
			return fmt.Errorf("invalid type for logging.level, expected string")
		}
	case "logging.file":
		if file, ok := value.(string); ok {
			config.Settings.Logging.File = file
		} else {
			return fmt.Errorf("invalid type for logging.file, expected string")
		}
	default:
		return fmt.Errorf("unknown setting key: %s", key)
	}
	return nil
}
