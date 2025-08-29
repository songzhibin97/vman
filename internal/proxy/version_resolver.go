package proxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/version"
)

// VersionResolver 版本解析器接口
type VersionResolver interface {
	// ResolveVersion 解析工具版本
	ResolveVersion(ctx context.Context, toolName, projectPath string) (*VersionResolution, error)

	// GetVersionPath 获取版本路径
	GetVersionPath(toolName, version string) (string, error)

	// ResolveConstraint 解析版本约束
	ResolveConstraint(toolName, constraint string) (string, error)

	// GetLatestVersion 获取最新版本
	GetLatestVersion(toolName string) (string, error)

	// ValidateVersion 验证版本格式
	ValidateVersion(version string) error

	// CompareVersions 比较版本
	CompareVersions(v1, v2 string) (int, error)

	// GetAvailableVersions 获取可用版本列表
	GetAvailableVersions(toolName string) ([]string, error)

	// IsVersionInstalled 检查版本是否已安装
	IsVersionInstalled(toolName, version string) bool

	// ResolveAlias 解析版本别名
	ResolveAlias(toolName, alias string) (string, error)

	// SetVersionCache 设置版本缓存
	SetVersionCache(toolName, projectPath, version string) error

	// ClearVersionCache 清除版本缓存
	ClearVersionCache() error
}

// VersionResolution 版本解析结果
type VersionResolution struct {
	ToolName         string    `json:"tool_name"`
	RequestedVersion string    `json:"requested_version,omitempty"`
	Version          string    `json:"version"`
	Source           string    `json:"source"` // "global", "project", "env", "alias", "constraint", "latest"
	ProjectPath      string    `json:"project_path,omitempty"`
	ConfigPath       string    `json:"config_path,omitempty"`
	IsInstalled      bool      `json:"is_installed"`
	ResolvedAt       time.Time `json:"resolved_at"`
}

// VersionCache 版本缓存
type VersionCache struct {
	ProjectPath string        `json:"project_path"`
	ToolName    string        `json:"tool_name"`
	Version     string        `json:"version"`
	Source      string        `json:"source"`
	CachedAt    time.Time     `json:"cached_at"`
	TTL         time.Duration `json:"ttl"`
}

// DefaultVersionResolver 默认版本解析器实现
type DefaultVersionResolver struct {
	fs             afero.Fs
	logger         *logrus.Logger
	configManager  config.Manager
	versionManager version.Manager
	cache          map[string]*VersionCache // projectPath:toolName -> cache
	cacheTTL       time.Duration
}

// NewVersionResolver 创建新的版本解析器
func NewVersionResolver(configManager config.Manager, versionManager version.Manager) VersionResolver {
	return NewVersionResolverWithFs(afero.NewOsFs(), configManager, versionManager)
}

// NewVersionResolverWithFs 使用指定文件系统创建版本解析器（用于测试）
func NewVersionResolverWithFs(fs afero.Fs, configManager config.Manager, versionManager version.Manager) VersionResolver {
	return &DefaultVersionResolver{
		fs:             fs,
		logger:         logrus.New(),
		configManager:  configManager,
		versionManager: versionManager,
		cache:          make(map[string]*VersionCache),
		cacheTTL:       5 * time.Minute, // 默认缓存5分钟
	}
}

// ResolveVersion 解析工具版本
func (vr *DefaultVersionResolver) ResolveVersion(ctx context.Context, toolName, projectPath string) (*VersionResolution, error) {
	vr.logger.Debugf("Resolving version for %s in %s", toolName, projectPath)

	// 检查缓存
	if cached := vr.getFromCache(toolName, projectPath); cached != nil {
		vr.logger.Debugf("Using cached version for %s: %s", toolName, cached.Version)
		return &VersionResolution{
			ToolName:    toolName,
			Version:     cached.Version,
			Source:      cached.Source,
			ProjectPath: projectPath,
			IsInstalled: vr.IsVersionInstalled(toolName, cached.Version),
			ResolvedAt:  time.Now(),
		}, nil
	}

	resolution := &VersionResolution{
		ToolName:    toolName,
		ProjectPath: projectPath,
		ResolvedAt:  time.Now(),
	}

	// 优先级顺序解析版本：
	// 1. 环境变量
	// 2. 项目配置
	// 3. 全局配置
	// 4. 最新版本

	// 1. 检查环境变量
	if version := vr.resolveFromEnvironment(toolName); version != "" {
		if vr.IsVersionInstalled(toolName, version) {
			resolution.Version = version
			resolution.Source = "env"
			resolution.IsInstalled = true
			vr.setCache(toolName, projectPath, resolution)
			return resolution, nil
		}
	}

	// 2. 检查项目配置
	if version, configPath := vr.resolveFromProject(toolName, projectPath); version != "" {
		// 检查是否为别名或约束
		resolvedVersion, err := vr.resolveVersionString(toolName, version)
		if err != nil {
			// 如果解析失败，返回错误，不要继续到下一个源
			return nil, fmt.Errorf("failed to resolve project version %s for %s: %w", version, toolName, err)
		}
		resolution.RequestedVersion = version
		resolution.Version = resolvedVersion
		resolution.Source = "project"
		resolution.ConfigPath = configPath
		resolution.IsInstalled = vr.IsVersionInstalled(toolName, resolvedVersion)
		vr.setCache(toolName, projectPath, resolution)
		return resolution, nil
	}

	// 3. 检查全局配置
	if version := vr.resolveFromGlobal(toolName); version != "" {
		resolvedVersion, err := vr.resolveVersionString(toolName, version)
		if err != nil {
			// 如果解析失败，返回错误，不要继续到下一个源
			return nil, fmt.Errorf("failed to resolve global version %s for %s: %w", version, toolName, err)
		}
		resolution.RequestedVersion = version
		resolution.Version = resolvedVersion
		resolution.Source = "global"
		resolution.IsInstalled = vr.IsVersionInstalled(toolName, resolvedVersion)
		vr.setCache(toolName, projectPath, resolution)
		return resolution, nil
	}

	// 4. 使用最新版本
	latestVersion, err := vr.GetLatestVersion(toolName)
	if err != nil {
		return nil, fmt.Errorf("no version found for %s and failed to get latest: %w", toolName, err)
	}

	resolution.Version = latestVersion
	resolution.Source = "latest"
	resolution.IsInstalled = vr.IsVersionInstalled(toolName, latestVersion)
	vr.setCache(toolName, projectPath, resolution)

	vr.logger.Infof("Resolved %s to version %s from %s", toolName, resolution.Version, resolution.Source)
	return resolution, nil
}

// GetVersionPath 获取版本路径
func (vr *DefaultVersionResolver) GetVersionPath(toolName, version string) (string, error) {
	return vr.versionManager.GetVersionPath(toolName, version)
}

// ResolveConstraint 解析版本约束
func (vr *DefaultVersionResolver) ResolveConstraint(toolName, constraint string) (string, error) {
	vr.logger.Debugf("Resolving constraint %s for %s", constraint, toolName)

	// 获取可用版本
	availableVersions, err := vr.GetAvailableVersions(toolName)
	if err != nil {
		return "", fmt.Errorf("failed to get available versions: %w", err)
	}

	if len(availableVersions) == 0 {
		return "", fmt.Errorf("no versions available for %s", toolName)
	}

	// 解析约束
	constraintObj, err := semver.NewConstraint(constraint)
	if err != nil {
		// 如果约束解析失败，尝试作为精确版本
		for _, v := range availableVersions {
			if v == constraint {
				return v, nil
			}
		}
		return "", fmt.Errorf("invalid version constraint: %s", constraint)
	}

	// 找到满足约束的最高版本
	var bestVersion *semver.Version
	for _, v := range availableVersions {
		version, err := semver.NewVersion(v)
		if err != nil {
			vr.logger.Warnf("Invalid version format: %s", v)
			continue
		}

		if constraintObj.Check(version) {
			if bestVersion == nil || version.GreaterThan(bestVersion) {
				bestVersion = version
			}
		}
	}

	if bestVersion == nil {
		return "", fmt.Errorf("no version satisfies constraint %s for %s", constraint, toolName)
	}

	return bestVersion.String(), nil
}

// GetLatestVersion 获取最新版本
func (vr *DefaultVersionResolver) GetLatestVersion(toolName string) (string, error) {
	return vr.versionManager.GetLatestVersion(toolName)
}

// ValidateVersion 验证版本格式
func (vr *DefaultVersionResolver) ValidateVersion(version string) error {
	return vr.versionManager.ValidateVersion(version)
}

// CompareVersions 比较版本
func (vr *DefaultVersionResolver) CompareVersions(v1, v2 string) (int, error) {
	version1, err := semver.NewVersion(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version v1: %s", v1)
	}

	version2, err := semver.NewVersion(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version v2: %s", v2)
	}

	return version1.Compare(version2), nil
}

// GetAvailableVersions 获取可用版本列表
func (vr *DefaultVersionResolver) GetAvailableVersions(toolName string) ([]string, error) {
	return vr.versionManager.GetInstalledVersions(toolName)
}

// IsVersionInstalled 检查版本是否已安装
func (vr *DefaultVersionResolver) IsVersionInstalled(toolName, version string) bool {
	return vr.versionManager.IsVersionInstalled(toolName, version)
}

// ResolveAlias 解析版本别名
func (vr *DefaultVersionResolver) ResolveAlias(toolName, alias string) (string, error) {
	vr.logger.Debugf("Resolving alias %s for %s", alias, toolName)

	// 常用别名处理
	switch alias {
	case "latest":
		return vr.GetLatestVersion(toolName)
	case "system":
		// 系统版本，返回PATH中找到的版本（如果有的话）
		return vr.getSystemVersion(toolName)
	}

	// 从工具配置中查找别名
	toolConfig, err := vr.configManager.LoadToolConfig(toolName)
	if err != nil {
		return "", fmt.Errorf("failed to load tool config: %w", err)
	}

	if toolConfig.VersionConfig.Aliases != nil {
		if version, exists := toolConfig.VersionConfig.Aliases[alias]; exists {
			// 递归解析别名
			return vr.resolveVersionString(toolName, version)
		}
	}

	return "", fmt.Errorf("unknown alias: %s", alias)
}

// SetVersionCache 设置版本缓存
func (vr *DefaultVersionResolver) SetVersionCache(toolName, projectPath, version string) error {
	cacheKey := vr.getCacheKey(toolName, projectPath)
	vr.cache[cacheKey] = &VersionCache{
		ProjectPath: projectPath,
		ToolName:    toolName,
		Version:     version,
		Source:      "manual",
		CachedAt:    time.Now(),
		TTL:         vr.cacheTTL,
	}
	return nil
}

// ClearVersionCache 清除版本缓存
func (vr *DefaultVersionResolver) ClearVersionCache() error {
	vr.cache = make(map[string]*VersionCache)
	vr.logger.Info("Version cache cleared")
	return nil
}

// resolveVersionString 解析版本字符串（可能是别名、约束或精确版本）
func (vr *DefaultVersionResolver) resolveVersionString(toolName, versionStr string) (string, error) {
	// 首先验证版本格式是否有效
	if err := vr.ValidateVersion(versionStr); err == nil {
		// 这是一个有效的版本格式，检查是否已安装
		if vr.IsVersionInstalled(toolName, versionStr) {
			return versionStr, nil
		}
		// 精确版本未安装，返回错误而不是回退
		return "", fmt.Errorf("version %s is not installed for %s", versionStr, toolName)
	}

	// 不是有效版本格式，尝试作为别名解析
	if alias, err := vr.ResolveAlias(toolName, versionStr); err == nil {
		return alias, nil
	}

	// 尝试作为约束解析
	if constraint, err := vr.ResolveConstraint(toolName, versionStr); err == nil {
		return constraint, nil
	}

	// 都失败了，返回错误
	return "", fmt.Errorf("unable to resolve version string '%s' for %s", versionStr, toolName)
}

// resolveFromEnvironment 从环境变量解析版本
func (vr *DefaultVersionResolver) resolveFromEnvironment(toolName string) string {
	// 检查工具特定的环境变量
	envVar := strings.ToUpper(toolName) + "_VERSION"
	if version := os.Getenv(envVar); version != "" {
		vr.logger.Debugf("Found version in environment %s=%s", envVar, version)
		return version
	}

	// 检查通用环境变量
	if version := os.Getenv("VMAN_" + strings.ToUpper(toolName) + "_VERSION"); version != "" {
		vr.logger.Debugf("Found version in environment VMAN_%s_VERSION=%s", strings.ToUpper(toolName), version)
		return version
	}

	return ""
}

// resolveFromProject 从项目配置解析版本
func (vr *DefaultVersionResolver) resolveFromProject(toolName, projectPath string) (string, string) {
	// 向上查找项目配置文件
	currentDir := projectPath
	for {
		// 检查 .vman-version 文件
		vmanVersionFile := filepath.Join(currentDir, ".vman-version")
		if vr.fileExists(vmanVersionFile) {
			if version := vr.readVersionFromFile(vmanVersionFile, toolName); version != "" {
				vr.logger.Debugf("Found version in .vman-version: %s", version)
				return version, vmanVersionFile
			}
		}

		// 检查 .tool-versions 文件（asdf兼容）
		toolVersionsFile := filepath.Join(currentDir, ".tool-versions")
		if vr.fileExists(toolVersionsFile) {
			if version := vr.readVersionFromToolVersions(toolVersionsFile, toolName); version != "" {
				vr.logger.Debugf("Found version in .tool-versions: %s", version)
				return version, toolVersionsFile
			}
		}

		// 检查项目配置文件
		projectConfig, err := vr.configManager.LoadProject(currentDir)
		if err == nil && projectConfig.Tools != nil {
			if version, exists := projectConfig.Tools[toolName]; exists {
				configPath := vr.configManager.GetProjectConfigPath(currentDir)
				vr.logger.Debugf("Found version in project config: %s", version)
				return version, configPath
			}
		}

		// 向上一级目录
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// 到达根目录
			break
		}
		currentDir = parentDir
	}

	return "", ""
}

// resolveFromGlobal 从全局配置解析版本
func (vr *DefaultVersionResolver) resolveFromGlobal(toolName string) string {
	globalConfig, err := vr.configManager.LoadGlobal()
	if err != nil {
		vr.logger.Warnf("Failed to load global config: %v", err)
		return ""
	}

	if globalConfig.GlobalVersions != nil {
		if version, exists := globalConfig.GlobalVersions[toolName]; exists {
			vr.logger.Debugf("Found version in global config: %s", version)
			return version
		}
	}

	return ""
}

// getSystemVersion 获取系统版本
func (vr *DefaultVersionResolver) getSystemVersion(toolName string) (string, error) {
	// 这里应该调用系统命令来获取版本，暂时返回错误
	return "", fmt.Errorf("system version resolution not implemented")
}

// readVersionFromFile 从版本文件读取版本
func (vr *DefaultVersionResolver) readVersionFromFile(filePath, toolName string) string {
	content, err := afero.ReadFile(vr.fs, filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 假设每行格式为 "tool version" 或者只有 "version"
		parts := strings.Fields(line)
		if len(parts) == 1 {
			// 只有版本号，适用于单工具文件
			return parts[0]
		} else if len(parts) >= 2 && parts[0] == toolName {
			// 工具名和版本号
			return parts[1]
		}
	}

	return ""
}

// readVersionFromToolVersions 从.tool-versions文件读取版本
func (vr *DefaultVersionResolver) readVersionFromToolVersions(filePath, toolName string) string {
	content, err := afero.ReadFile(vr.fs, filePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == toolName {
			return parts[1]
		}
	}

	return ""
}

// getFromCache 从缓存获取版本
func (vr *DefaultVersionResolver) getFromCache(toolName, projectPath string) *VersionCache {
	cacheKey := vr.getCacheKey(toolName, projectPath)
	cached, exists := vr.cache[cacheKey]
	if !exists {
		return nil
	}

	// 检查缓存是否过期
	if time.Since(cached.CachedAt) > cached.TTL {
		delete(vr.cache, cacheKey)
		return nil
	}

	return cached
}

// setCache 设置缓存
func (vr *DefaultVersionResolver) setCache(toolName, projectPath string, resolution *VersionResolution) {
	cacheKey := vr.getCacheKey(toolName, projectPath)
	vr.cache[cacheKey] = &VersionCache{
		ProjectPath: projectPath,
		ToolName:    toolName,
		Version:     resolution.Version,
		Source:      resolution.Source,
		CachedAt:    time.Now(),
		TTL:         vr.cacheTTL,
	}
}

// getCacheKey 获取缓存键
func (vr *DefaultVersionResolver) getCacheKey(toolName, projectPath string) string {
	return fmt.Sprintf("%s:%s", projectPath, toolName)
}

// fileExists 检查文件是否存在
func (vr *DefaultVersionResolver) fileExists(path string) bool {
	_, err := vr.fs.Stat(path)
	return err == nil
}
