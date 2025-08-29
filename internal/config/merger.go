package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
)

// Merger 配置合并器接口
type Merger interface {
	// MergeConfigs 合并配置
	MergeConfigs(global *types.GlobalConfig, project *types.ProjectConfig, strategy types.ConfigMergeStrategy) (*types.EffectiveConfig, error)

	// ResolveVersion 解析版本（考虑别名和约束）
	ResolveVersion(toolName, requestedVersion string, metadata *types.ToolMetadata) (*types.VersionResolution, error)

	// MergeSettings 合并设置
	MergeSettings(global *types.Settings, project *types.Settings) *types.Settings

	// GetVersionSource 获取版本来源
	GetVersionSource(toolName string, global *types.GlobalConfig, project *types.ProjectConfig) (string, string)
}

// DefaultMerger 默认配置合并器实现
type DefaultMerger struct {
	logger *logrus.Logger
}

// NewMerger 创建新的配置合并器
func NewMerger() Merger {
	return &DefaultMerger{
		logger: logrus.New(),
	}
}

// MergeConfigs 合并配置
func (m *DefaultMerger) MergeConfigs(global *types.GlobalConfig, project *types.ProjectConfig, strategy types.ConfigMergeStrategy) (*types.EffectiveConfig, error) {
	m.logger.Debugf("Merging configurations with strategy: %v", strategy)

	if global == nil {
		return nil, fmt.Errorf("global config cannot be nil")
	}

	// 如果项目配置为nil，使用默认配置
	if project == nil {
		project = types.GetDefaultProjectConfig()
	}

	// 初始化解析结果
	resolvedVersions := make(map[string]string)
	configSource := make(map[string]string)

	// 根据策略合并版本
	switch strategy {
	case types.OverrideStrategy:
		m.mergeWithOverride(global, project, resolvedVersions, configSource)
	case types.MergeStrategy:
		m.mergeWithMerge(global, project, resolvedVersions, configSource)
	case types.IgnoreStrategy:
		m.mergeWithIgnore(global, project, resolvedVersions, configSource)
	default:
		m.mergeWithOverride(global, project, resolvedVersions, configSource) // 默认使用覆盖策略
	}

	effective := &types.EffectiveConfig{
		Global:           global,
		Project:          project,
		ResolvedVersions: resolvedVersions,
		ConfigSource:     configSource,
	}

	m.logger.Debugf("Configuration merge completed, resolved %d versions", len(resolvedVersions))
	return effective, nil
}

// ResolveVersion 解析版本（考虑别名和约束）
func (m *DefaultMerger) ResolveVersion(toolName, requestedVersion string, metadata *types.ToolMetadata) (*types.VersionResolution, error) {
	m.logger.Debugf("Resolving version for tool %s, requested: %s", toolName, requestedVersion)

	resolution := &types.VersionResolution{
		ToolName:         toolName,
		RequestedVersion: requestedVersion,
		ResolvedVersion:  requestedVersion,
		Source:           "direct",
	}

	if metadata == nil {
		// 如果没有元数据，直接返回请求的版本
		return resolution, nil
	}

	// 检查是否为别名
	if resolvedFromAlias, exists := metadata.VersionConfig.Aliases[requestedVersion]; exists {
		resolution.ResolvedVersion = resolvedFromAlias
		resolution.Source = "alias"
		m.logger.Debugf("Resolved alias %s to version %s", requestedVersion, resolvedFromAlias)
	}

	// 应用版本约束
	if err := m.applyVersionConstraints(resolution, &metadata.VersionConfig.Constraints); err != nil {
		return nil, fmt.Errorf("version constraint violation: %w", err)
	}

	return resolution, nil
}

// MergeSettings 合并设置（项目级设置优先）
func (m *DefaultMerger) MergeSettings(global *types.Settings, project *types.Settings) *types.Settings {
	// 如果项目设置为nil，返回全局设置的副本
	if project == nil {
		return &types.Settings{
			Download: global.Download,
			Proxy:    global.Proxy,
			Logging:  global.Logging,
		}
	}

	// 合并设置（项目设置优先）
	merged := &types.Settings{}

	// 合并下载设置
	merged.Download = global.Download
	if project.Download.Timeout > 0 {
		merged.Download.Timeout = project.Download.Timeout
	}
	if project.Download.Retries > 0 {
		merged.Download.Retries = project.Download.Retries
	}
	if project.Download.ConcurrentDownloads > 0 {
		merged.Download.ConcurrentDownloads = project.Download.ConcurrentDownloads
	}

	// 合并代理设置
	merged.Proxy = global.Proxy
	// 项目级代理设置会覆盖全局设置

	// 合并日志设置
	merged.Logging = global.Logging
	if project.Logging.Level != "" {
		merged.Logging.Level = project.Logging.Level
	}
	if project.Logging.File != "" {
		merged.Logging.File = project.Logging.File
	}

	return merged
}

// GetVersionSource 获取版本来源
func (m *DefaultMerger) GetVersionSource(toolName string, global *types.GlobalConfig, project *types.ProjectConfig) (string, string) {
	// 检查项目配置
	if project != nil && project.Tools != nil {
		if version, exists := project.Tools[toolName]; exists && version != "" {
			return version, "project"
		}
	}

	// 检查全局版本配置
	if global.GlobalVersions != nil {
		if version, exists := global.GlobalVersions[toolName]; exists && version != "" {
			return version, "global"
		}
	}

	// 检查工具信息中的当前版本
	if global.Tools != nil {
		if toolInfo, exists := global.Tools[toolName]; exists && toolInfo.CurrentVersion != "" {
			return toolInfo.CurrentVersion, "tool_info"
		}
	}

	return "", "none"
}

// 私有方法

// mergeWithOverride 使用覆盖策略合并（项目配置覆盖全局配置）
func (m *DefaultMerger) mergeWithOverride(global *types.GlobalConfig, project *types.ProjectConfig, resolved map[string]string, source map[string]string) {
	m.logger.Debug("Applying override merge strategy")

	// 首先添加全局版本
	if global.GlobalVersions != nil {
		for toolName, version := range global.GlobalVersions {
			resolved[toolName] = version
			source[toolName] = "global"
		}
	}

	// 添加工具信息中的当前版本（如果全局版本中没有）
	if global.Tools != nil {
		for toolName, toolInfo := range global.Tools {
			if toolInfo.CurrentVersion != "" && resolved[toolName] == "" {
				resolved[toolName] = toolInfo.CurrentVersion
				source[toolName] = "global_tool"
			}
		}
	}

	// 项目配置覆盖全局配置
	if project.Tools != nil {
		for toolName, version := range project.Tools {
			if version != "" {
				resolved[toolName] = version
				source[toolName] = "project"
			}
		}
	}
}

// mergeWithMerge 使用合并策略合并（合并所有版本，项目优先）
func (m *DefaultMerger) mergeWithMerge(global *types.GlobalConfig, project *types.ProjectConfig, resolved map[string]string, source map[string]string) {
	m.logger.Debug("Applying merge strategy")

	// 合并全局版本
	if global.GlobalVersions != nil {
		for toolName, version := range global.GlobalVersions {
			resolved[toolName] = version
			source[toolName] = "global"
		}
	}

	// 合并工具信息中的当前版本
	if global.Tools != nil {
		for toolName, toolInfo := range global.Tools {
			if toolInfo.CurrentVersion != "" {
				if existing, exists := resolved[toolName]; !exists || existing == "" {
					resolved[toolName] = toolInfo.CurrentVersion
					source[toolName] = "global_tool"
				}
			}
		}
	}

	// 合并项目配置（项目优先）
	if project.Tools != nil {
		for toolName, version := range project.Tools {
			if version != "" {
				resolved[toolName] = version
				source[toolName] = "project"
			}
		}
	}
}

// mergeWithIgnore 使用忽略策略合并（只使用全局配置）
func (m *DefaultMerger) mergeWithIgnore(global *types.GlobalConfig, project *types.ProjectConfig, resolved map[string]string, source map[string]string) {
	m.logger.Debug("Applying ignore strategy")

	// 只使用全局版本
	if global.GlobalVersions != nil {
		for toolName, version := range global.GlobalVersions {
			resolved[toolName] = version
			source[toolName] = "global"
		}
	}

	// 添加工具信息中的当前版本
	if global.Tools != nil {
		for toolName, toolInfo := range global.Tools {
			if toolInfo.CurrentVersion != "" && resolved[toolName] == "" {
				resolved[toolName] = toolInfo.CurrentVersion
				source[toolName] = "global_tool"
			}
		}
	}
}

// applyVersionConstraints 应用版本约束
func (m *DefaultMerger) applyVersionConstraints(resolution *types.VersionResolution, constraints *types.VersionConstraints) error {
	if constraints == nil {
		return nil
	}

	version := resolution.ResolvedVersion

	// 跳过特殊版本（如latest、stable等）
	if m.isSpecialVersion(version) {
		return nil
	}

	// 这里应该使用semver来检查约束，但为了简化，我们只做基本检查
	// 在实际实现中，应该使用 github.com/Masterminds/semver 来进行精确的版本比较

	if constraints.MinVersion != "" {
		if m.isVersionLess(version, constraints.MinVersion) {
			return fmt.Errorf("version %s is less than minimum required version %s", version, constraints.MinVersion)
		}
	}

	if constraints.MaxVersion != "" {
		if m.isVersionGreater(version, constraints.MaxVersion) {
			return fmt.Errorf("version %s is greater than maximum allowed version %s", version, constraints.MaxVersion)
		}
	}

	return nil
}

// isSpecialVersion 检查是否为特殊版本
func (m *DefaultMerger) isSpecialVersion(version string) bool {
	specialVersions := map[string]bool{
		"latest": true,
		"stable": true,
		"main":   true,
		"master": true,
	}
	return specialVersions[version]
}

// isVersionLess 简单的版本比较（版本1 < 版本2）
func (m *DefaultMerger) isVersionLess(version1, version2 string) bool {
	// 移除v前缀
	v1 := strings.TrimPrefix(version1, "v")
	v2 := strings.TrimPrefix(version2, "v")

	// 简单的字符串比较（在实际实现中应该使用semver）
	return v1 < v2
}

// isVersionGreater 简单的版本比较（版本1 > 版本2）
func (m *DefaultMerger) isVersionGreater(version1, version2 string) bool {
	// 移除v前缀
	v1 := strings.TrimPrefix(version1, "v")
	v2 := strings.TrimPrefix(version2, "v")

	// 简单的字符串比较（在实际实现中应该使用semver）
	return v1 > v2
}

// AdvancedMerger 高级配置合并器（支持更复杂的合并逻辑）
type AdvancedMerger struct {
	*DefaultMerger
	validator Validator
}

// NewAdvancedMerger 创建新的高级配置合并器
func NewAdvancedMerger(validator Validator) Merger {
	return &AdvancedMerger{
		DefaultMerger: &DefaultMerger{
			logger: logrus.New(),
		},
		validator: validator,
	}
}

// MergeConfigs 高级配置合并（包含验证）
func (m *AdvancedMerger) MergeConfigs(global *types.GlobalConfig, project *types.ProjectConfig, strategy types.ConfigMergeStrategy) (*types.EffectiveConfig, error) {
	m.logger.Debug("Performing advanced configuration merge with validation")

	// 验证输入配置
	if err := m.validator.ValidateGlobalConfig(global); err != nil {
		return nil, fmt.Errorf("global config validation failed: %w", err)
	}

	if project != nil {
		if err := m.validator.ValidateProjectConfig(project); err != nil {
			return nil, fmt.Errorf("project config validation failed: %w", err)
		}
	}

	// 执行基础合并
	effective, err := m.DefaultMerger.MergeConfigs(global, project, strategy)
	if err != nil {
		return nil, err
	}

	// 额外的后处理验证
	if err := m.validateMergedConfig(effective); err != nil {
		return nil, fmt.Errorf("merged config validation failed: %w", err)
	}

	m.logger.Debug("Advanced configuration merge completed successfully")
	return effective, nil
}

// validateMergedConfig 验证合并后的配置
func (m *AdvancedMerger) validateMergedConfig(effective *types.EffectiveConfig) error {
	// 检查是否有工具版本冲突
	conflicts := m.detectVersionConflicts(effective)
	if len(conflicts) > 0 {
		m.logger.Warnf("Detected %d version conflicts", len(conflicts))
		for _, conflict := range conflicts {
			m.logger.Warnf("Version conflict: %s", conflict)
		}
	}

	// 检查工具依赖关系
	if err := m.validateToolDependencies(effective); err != nil {
		return fmt.Errorf("tool dependency validation failed: %w", err)
	}

	return nil
}

// detectVersionConflicts 检测版本冲突
func (m *AdvancedMerger) detectVersionConflicts(effective *types.EffectiveConfig) []string {
	var conflicts []string

	// 这里可以实现更复杂的冲突检测逻辑
	// 例如检查工具版本之间的兼容性

	return conflicts
}

// validateToolDependencies 验证工具依赖关系
func (m *AdvancedMerger) validateToolDependencies(effective *types.EffectiveConfig) error {
	// 这里可以实现工具依赖关系验证
	// 例如某些工具版本之间的依赖关系检查

	return nil
}
