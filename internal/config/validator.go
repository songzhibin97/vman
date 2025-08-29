package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
)

// Validator 配置验证器接口
type Validator interface {
	// ValidateGlobalConfig 验证全局配置
	ValidateGlobalConfig(config *types.GlobalConfig) error

	// ValidateProjectConfig 验证项目配置
	ValidateProjectConfig(config *types.ProjectConfig) error

	// ValidateToolMetadata 验证工具元数据
	ValidateToolMetadata(metadata *types.ToolMetadata) error

	// ValidateVersion 验证版本字符串
	ValidateVersion(version string) error

	// ValidateToolName 验证工具名称
	ValidateToolName(name string) error

	// ValidatePath 验证路径
	ValidatePath(path string) error
}

// DefaultValidator 默认配置验证器实现
type DefaultValidator struct {
	logger *logrus.Logger
}

// NewValidator 创建新的配置验证器
func NewValidator() Validator {
	return &DefaultValidator{
		logger: logrus.New(),
	}
}

// ValidateGlobalConfig 验证全局配置
func (v *DefaultValidator) ValidateGlobalConfig(config *types.GlobalConfig) error {
	v.logger.Debug("Validating global configuration")

	if config == nil {
		return &types.ConfigValidationError{
			Field:   "config",
			Message: "global config cannot be nil",
			Value:   nil,
		}
	}

	// 验证版本
	if err := v.validateConfigVersion(config.Version); err != nil {
		return err
	}

	// 验证设置
	if err := v.validateSettings(&config.Settings); err != nil {
		return err
	}

	// 验证全局版本映射
	if err := v.validateGlobalVersions(config.GlobalVersions); err != nil {
		return err
	}

	// 验证工具信息
	if err := v.validateToolsInfo(config.Tools); err != nil {
		return err
	}

	v.logger.Debug("Global configuration validation passed")
	return nil
}

// ValidateProjectConfig 验证项目配置
func (v *DefaultValidator) ValidateProjectConfig(config *types.ProjectConfig) error {
	v.logger.Debug("Validating project configuration")

	if config == nil {
		return &types.ConfigValidationError{
			Field:   "config",
			Message: "project config cannot be nil",
			Value:   nil,
		}
	}

	// 验证版本
	if err := v.validateConfigVersion(config.Version); err != nil {
		return err
	}

	// 验证工具版本映射
	if err := v.validateToolVersions(config.Tools); err != nil {
		return err
	}

	v.logger.Debug("Project configuration validation passed")
	return nil
}

// ValidateToolMetadata 验证工具元数据
func (v *DefaultValidator) ValidateToolMetadata(metadata *types.ToolMetadata) error {
	if metadata == nil {
		return &types.ConfigValidationError{
			Field:   "metadata",
			Message: "tool metadata cannot be nil",
			Value:   nil,
		}
	}

	v.logger.Debugf("Validating tool metadata for: %s", metadata.Name)

	// 验证工具名称
	if err := v.ValidateToolName(metadata.Name); err != nil {
		return err
	}

	// 验证描述
	if strings.TrimSpace(metadata.Description) == "" {
		return &types.ConfigValidationError{
			Field:   "description",
			Message: "tool description is required",
			Value:   metadata.Description,
		}
	}

	// 验证主页URL
	if err := v.validateURL(metadata.Homepage, "homepage"); err != nil {
		return err
	}

	// 验证仓库URL
	if err := v.validateURL(metadata.Repository, "repository"); err != nil {
		return err
	}

	// 验证下载配置
	if err := v.validateDownloadConfig(&metadata.DownloadConfig); err != nil {
		return err
	}

	// 验证版本配置
	if err := v.validateVersionConfig(&metadata.VersionConfig); err != nil {
		return err
	}

	v.logger.Debug("Tool metadata validation passed")
	return nil
}

// ValidateVersion 验证版本字符串
func (v *DefaultValidator) ValidateVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return &types.ConfigValidationError{
			Field:   "version",
			Message: "version cannot be empty",
			Value:   version,
		}
	}

	// 检查是否为有效的语义版本
	if _, err := semver.NewVersion(version); err != nil {
		// 如果不是严格的语义版本，检查是否为常见的版本格式
		if !v.isValidVersionFormat(version) {
			return &types.ConfigValidationError{
				Field:   "version",
				Message: fmt.Sprintf("invalid version format: %s", version),
				Value:   version,
			}
		}
	}

	return nil
}

// ValidateToolName 验证工具名称
func (v *DefaultValidator) ValidateToolName(name string) error {
	if strings.TrimSpace(name) == "" {
		return &types.ConfigValidationError{
			Field:   "name",
			Message: "tool name cannot be empty",
			Value:   name,
		}
	}

	// 工具名称只能包含字母、数字、连字符和下划线
	validNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validNamePattern.MatchString(name) {
		return &types.ConfigValidationError{
			Field:   "name",
			Message: "tool name can only contain letters, numbers, hyphens, and underscores",
			Value:   name,
		}
	}

	// 名称长度限制
	if len(name) > 50 {
		return &types.ConfigValidationError{
			Field:   "name",
			Message: "tool name cannot exceed 50 characters",
			Value:   name,
		}
	}

	return nil
}

// ValidatePath 验证路径
func (v *DefaultValidator) ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return &types.ConfigValidationError{
			Field:   "path",
			Message: "path cannot be empty",
			Value:   path,
		}
	}

	// 检查路径是否为绝对路径或相对路径
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, ".") && !strings.HasPrefix(path, "~") {
		// 对于不以特殊字符开头的相对路径，也应该是合法的
		// 只有当路径包含非法字符时才报错
		if strings.ContainsAny(path, "<>:|?*") {
			return &types.ConfigValidationError{
				Field:   "path",
				Message: "path contains invalid characters",
				Value:   path,
			}
		}
	}

	return nil
}

// 私有验证方法

// validateConfigVersion 验证配置版本
func (v *DefaultValidator) validateConfigVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return &types.ConfigValidationError{
			Field:   "version",
			Message: "config version is required",
			Value:   version,
		}
	}

	// 支持的配置版本
	supportedVersions := map[string]bool{
		"1.0": true,
	}

	if !supportedVersions[version] {
		return &types.ConfigValidationError{
			Field:   "version",
			Message: fmt.Sprintf("unsupported config version: %s, supported versions: 1.0", version),
			Value:   version,
		}
	}

	return nil
}

// validateSettings 验证设置
func (v *DefaultValidator) validateSettings(settings *types.Settings) error {
	// 验证下载设置
	if err := v.validateDownloadSettings(&settings.Download); err != nil {
		return err
	}

	// 验证代理设置
	if err := v.validateProxySettings(&settings.Proxy); err != nil {
		return err
	}

	// 验证日志设置
	if err := v.validateLoggingSettings(&settings.Logging); err != nil {
		return err
	}

	return nil
}

// validateDownloadSettings 验证下载设置
func (v *DefaultValidator) validateDownloadSettings(settings *types.DownloadSettings) error {
	// 验证超时时间
	if settings.Timeout <= 0 {
		return &types.ConfigValidationError{
			Field:   "settings.download.timeout",
			Message: "timeout must be greater than 0",
			Value:   settings.Timeout,
		}
	}

	if settings.Timeout > 30*time.Minute {
		return &types.ConfigValidationError{
			Field:   "settings.download.timeout",
			Message: "timeout cannot exceed 30 minutes",
			Value:   settings.Timeout,
		}
	}

	// 验证重试次数
	if settings.Retries < 0 {
		return &types.ConfigValidationError{
			Field:   "settings.download.retries",
			Message: "retries must be >= 0",
			Value:   settings.Retries,
		}
	}

	if settings.Retries > 10 {
		return &types.ConfigValidationError{
			Field:   "settings.download.retries",
			Message: "retries cannot exceed 10",
			Value:   settings.Retries,
		}
	}

	// 验证并发下载数
	if settings.ConcurrentDownloads < 1 {
		return &types.ConfigValidationError{
			Field:   "settings.download.concurrent_downloads",
			Message: "concurrent_downloads must be >= 1",
			Value:   settings.ConcurrentDownloads,
		}
	}

	if settings.ConcurrentDownloads > 10 {
		return &types.ConfigValidationError{
			Field:   "settings.download.concurrent_downloads",
			Message: "concurrent_downloads cannot exceed 10",
			Value:   settings.ConcurrentDownloads,
		}
	}

	return nil
}

// validateProxySettings 验证代理设置
func (v *DefaultValidator) validateProxySettings(settings *types.ProxySettings) error {
	// 代理设置目前只是布尔值，不需要特殊验证
	return nil
}

// validateLoggingSettings 验证日志设置
func (v *DefaultValidator) validateLoggingSettings(settings *types.LoggingSettings) error {
	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[settings.Level] {
		return &types.ConfigValidationError{
			Field:   "settings.logging.level",
			Message: "invalid log level, must be one of: debug, info, warn, error",
			Value:   settings.Level,
		}
	}

	// 验证日志文件路径
	if strings.TrimSpace(settings.File) == "" {
		return &types.ConfigValidationError{
			Field:   "settings.logging.file",
			Message: "log file path is required",
			Value:   settings.File,
		}
	}

	return nil
}

// validateGlobalVersions 验证全局版本映射
func (v *DefaultValidator) validateGlobalVersions(versions map[string]string) error {
	for toolName, version := range versions {
		if err := v.ValidateToolName(toolName); err != nil {
			return fmt.Errorf("invalid tool name in global_versions: %w", err)
		}

		if err := v.ValidateVersion(version); err != nil {
			return fmt.Errorf("invalid version for tool %s in global_versions: %w", toolName, err)
		}
	}

	return nil
}

// validateToolsInfo 验证工具信息
func (v *DefaultValidator) validateToolsInfo(tools map[string]types.ToolInfo) error {
	for toolName, toolInfo := range tools {
		if err := v.ValidateToolName(toolName); err != nil {
			return fmt.Errorf("invalid tool name in tools: %w", err)
		}

		// 验证当前版本
		if toolInfo.CurrentVersion != "" {
			if err := v.ValidateVersion(toolInfo.CurrentVersion); err != nil {
				return fmt.Errorf("invalid current version for tool %s: %w", toolName, err)
			}
		}

		// 验证已安装版本列表
		for _, version := range toolInfo.InstalledVersions {
			if err := v.ValidateVersion(version); err != nil {
				return fmt.Errorf("invalid installed version %s for tool %s: %w", version, toolName, err)
			}
		}
	}

	return nil
}

// validateToolVersions 验证工具版本映射
func (v *DefaultValidator) validateToolVersions(tools map[string]string) error {
	for toolName, version := range tools {
		if err := v.ValidateToolName(toolName); err != nil {
			return fmt.Errorf("invalid tool name in project tools: %w", err)
		}

		if err := v.ValidateVersion(version); err != nil {
			return fmt.Errorf("invalid version for tool %s in project tools: %w", toolName, err)
		}
	}

	return nil
}

// validateDownloadConfig 验证下载配置
func (v *DefaultValidator) validateDownloadConfig(config *types.DownloadConfig) error {
	// 验证下载类型
	validTypes := map[string]bool{
		"direct":  true,
		"github":  true,
		"archive": true,
	}

	if !validTypes[config.Type] {
		return &types.ConfigValidationError{
			Field:   "download.type",
			Message: "invalid download type, must be one of: direct, github, archive",
			Value:   config.Type,
		}
	}

	// 根据类型验证相应字段
	switch config.Type {
	case "direct":
		if strings.TrimSpace(config.URLTemplate) == "" {
			return &types.ConfigValidationError{
				Field:   "download.url_template",
				Message: "url_template is required for direct download type",
				Value:   config.URLTemplate,
			}
		}
	case "github":
		if strings.TrimSpace(config.Repository) == "" {
			return &types.ConfigValidationError{
				Field:   "download.repository",
				Message: "repository is required for github download type",
				Value:   config.Repository,
			}
		}
	case "archive":
		if strings.TrimSpace(config.URLTemplate) == "" {
			return &types.ConfigValidationError{
				Field:   "download.url_template",
				Message: "url_template is required for archive download type",
				Value:   config.URLTemplate,
			}
		}
		if strings.TrimSpace(config.ExtractBinary) == "" {
			return &types.ConfigValidationError{
				Field:   "download.extract_binary",
				Message: "extract_binary is required for archive download type",
				Value:   config.ExtractBinary,
			}
		}
	}

	return nil
}

// validateVersionConfig 验证版本配置
func (v *DefaultValidator) validateVersionConfig(config *types.VersionConfig) error {
	// 验证版本别名
	for alias, version := range config.Aliases {
		if err := v.ValidateVersion(version); err != nil {
			return fmt.Errorf("invalid version %s for alias %s: %w", version, alias, err)
		}
	}

	// 验证版本约束
	if config.Constraints.MinVersion != "" {
		if err := v.ValidateVersion(config.Constraints.MinVersion); err != nil {
			return fmt.Errorf("invalid min_version: %w", err)
		}
	}

	if config.Constraints.MaxVersion != "" {
		if err := v.ValidateVersion(config.Constraints.MaxVersion); err != nil {
			return fmt.Errorf("invalid max_version: %w", err)
		}
	}

	// 如果同时设置了最小和最大版本，验证顺序
	if config.Constraints.MinVersion != "" && config.Constraints.MaxVersion != "" {
		minVer, err1 := semver.NewVersion(config.Constraints.MinVersion)
		maxVer, err2 := semver.NewVersion(config.Constraints.MaxVersion)

		if err1 == nil && err2 == nil && minVer.GreaterThan(maxVer) {
			return &types.ConfigValidationError{
				Field:   "versions.constraints",
				Message: "min_version cannot be greater than max_version",
				Value:   fmt.Sprintf("min: %s, max: %s", config.Constraints.MinVersion, config.Constraints.MaxVersion),
			}
		}
	}

	return nil
}

// validateURL 验证URL
func (v *DefaultValidator) validateURL(url, fieldName string) error {
	if strings.TrimSpace(url) == "" {
		return &types.ConfigValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s URL is required", fieldName),
			Value:   url,
		}
	}

	// 简单的URL格式验证
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return &types.ConfigValidationError{
			Field:   fieldName,
			Message: fmt.Sprintf("%s URL must start with http:// or https://", fieldName),
			Value:   url,
		}
	}

	return nil
}

// isValidVersionFormat 检查是否为有效的版本格式
func (v *DefaultValidator) isValidVersionFormat(version string) bool {
	// 支持的版本格式模式
	patterns := []string{
		`^\d+\.\d+\.\d+$`,          // 1.2.3
		`^\d+\.\d+\.\d+-\w+$`,      // 1.2.3-alpha
		`^\d+\.\d+\.\d+-\w+\.\d+$`, // 1.2.3-alpha.1
		`^v\d+\.\d+\.\d+$`,         // v1.2.3
		`^v\d+\.\d+\.\d+-\w+$`,     // v1.2.3-alpha
		`^\d+\.\d+$`,               // 1.2
		`^v\d+\.\d+$`,              // v1.2
		`^\d+\.\d+\.\d+\+\w+$`,     // 1.2.3+build
		`^latest$`,                 // latest
		`^stable$`,                 // stable
		`^main$`,                   // main
		`^master$`,                 // master
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, version); matched {
			return true
		}
	}

	return false
}
