package proxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/pkg/types"
)

// ContextManager 上下文管理器接口
type ContextManager interface {
	// DetectProjectContext 检测项目上下文
	DetectProjectContext(workingDir string) (*ProjectContext, error)

	// FindProjectRoot 查找项目根目录
	FindProjectRoot(startDir string) (string, error)

	// GetEffectiveConfig 获取有效配置（合并全局和项目配置）
	GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error)

	// WatchConfigChanges 监听配置变更
	WatchConfigChanges(ctx context.Context, callback ConfigChangeCallback) error

	// GetToolContext 获取工具上下文
	GetToolContext(toolName, projectPath string) (*ToolContext, error)

	// UpdateProjectContext 更新项目上下文
	UpdateProjectContext(projectPath string, context *ProjectContext) error

	// ClearContextCache 清除上下文缓存
	ClearContextCache() error

	// IsProjectDirectory 检查是否为项目目录
	IsProjectDirectory(dir string) bool

	// GetEnvironmentContext 获取环境上下文
	GetEnvironmentContext() *EnvironmentContext
}

// ProjectContext 项目上下文
type ProjectContext struct {
	RootPath      string                 `json:"root_path"`
	ConfigFiles   []string               `json:"config_files"`
	ProjectConfig *types.ProjectConfig   `json:"project_config,omitempty"`
	DetectedAt    time.Time              `json:"detected_at"`
	ProjectType   string                 `json:"project_type,omitempty"` // "node", "python", "go", etc.
	Framework     string                 `json:"framework,omitempty"`    // "react", "vue", "django", etc.
	BuildSystem   string                 `json:"build_system,omitempty"` // "npm", "yarn", "pip", "go mod", etc.
	Dependencies  map[string]string      `json:"dependencies,omitempty"`
	Scripts       map[string]string      `json:"scripts,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ToolContext 工具上下文
type ToolContext struct {
	ToolName    string            `json:"tool_name"`
	ProjectPath string            `json:"project_path"`
	Version     string            `json:"version"`
	Source      string            `json:"source"`
	ConfigPath  string            `json:"config_path,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"working_dir"`
	LastUpdated time.Time         `json:"last_updated"`
}

// EnvironmentContext 环境上下文
type EnvironmentContext struct {
	OS         string            `json:"os"`
	Arch       string            `json:"arch"`
	Shell      string            `json:"shell"`
	HomeDir    string            `json:"home_dir"`
	WorkingDir string            `json:"working_dir"`
	PathDirs   []string          `json:"path_dirs"`
	EnvVars    map[string]string `json:"env_vars"`
	VmanDir    string            `json:"vman_dir"`
	ShimsDir   string            `json:"shims_dir"`
	DetectedAt time.Time         `json:"detected_at"`
}

// ConfigChangeCallback 配置变更回调
type ConfigChangeCallback func(*types.ConfigChangeEvent)

// DefaultContextManager 默认上下文管理器实现
type DefaultContextManager struct {
	fs            afero.Fs
	logger        *logrus.Logger
	configManager config.Manager
	projectCache  map[string]*ProjectContext // projectPath -> context
	toolCache     map[string]*ToolContext    // projectPath:toolName -> context
	cacheTimeout  time.Duration
}

// NewContextManager 创建新的上下文管理器
func NewContextManager(configManager config.Manager) ContextManager {
	return NewContextManagerWithFs(afero.NewOsFs(), configManager)
}

// NewContextManagerWithFs 使用指定文件系统创建上下文管理器（用于测试）
func NewContextManagerWithFs(fs afero.Fs, configManager config.Manager) ContextManager {
	return &DefaultContextManager{
		fs:            fs,
		logger:        logrus.New(),
		configManager: configManager,
		projectCache:  make(map[string]*ProjectContext),
		toolCache:     make(map[string]*ToolContext),
		cacheTimeout:  10 * time.Minute,
	}
}

// DetectProjectContext 检测项目上下文
func (cm *DefaultContextManager) DetectProjectContext(workingDir string) (*ProjectContext, error) {
	cm.logger.Debugf("Detecting project context for: %s", workingDir)

	// 检查缓存
	if cached := cm.getProjectFromCache(workingDir); cached != nil {
		return cached, nil
	}

	// 查找项目根目录
	rootPath, err := cm.FindProjectRoot(workingDir)
	if err != nil {
		// 如果找不到项目根目录，使用当前目录
		rootPath = workingDir
	}

	context := &ProjectContext{
		RootPath:   rootPath,
		DetectedAt: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// 检测配置文件
	context.ConfigFiles = cm.findConfigFiles(rootPath)

	// 加载项目配置
	if projectConfig, err := cm.configManager.LoadProject(rootPath); err == nil {
		context.ProjectConfig = projectConfig
	}

	// 检测项目类型和特征
	cm.detectProjectFeatures(context)

	// 缓存结果
	cm.setProjectCache(workingDir, context)

	cm.logger.Infof("Detected project context: type=%s, root=%s", context.ProjectType, context.RootPath)
	return context, nil
}

// FindProjectRoot 查找项目根目录
func (cm *DefaultContextManager) FindProjectRoot(startDir string) (string, error) {
	cm.logger.Debugf("Finding project root from: %s", startDir)

	// 项目根目录标识文件
	rootMarkers := []string{
		".vman",
		".git",
		".vman-version",
		".tool-versions",
		"package.json",
		"go.mod",
		"Cargo.toml",
		"pyproject.toml",
		"requirements.txt",
		"pom.xml",
		"build.gradle",
		"Makefile",
		"CMakeLists.txt",
	}

	currentDir := startDir
	for {
		// 检查是否存在项目根目录标识
		for _, marker := range rootMarkers {
			markerPath := filepath.Join(currentDir, marker)
			if cm.fileExists(markerPath) {
				cm.logger.Debugf("Found project root marker %s in %s", marker, currentDir)
				return currentDir, nil
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

	return "", fmt.Errorf("no project root found from %s", startDir)
}

// GetEffectiveConfig 获取有效配置（合并全局和项目配置）
func (cm *DefaultContextManager) GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error) {
	cm.logger.Debugf("Getting effective config for: %s", projectPath)

	// 加载全局配置
	globalConfig, err := cm.configManager.LoadGlobal()
	if err != nil {
		return nil, fmt.Errorf("failed to load global config: %w", err)
	}

	// 加载项目配置
	projectConfig, err := cm.configManager.LoadProject(projectPath)
	if err != nil {
		// 项目配置不存在时使用空配置
		projectConfig = types.GetDefaultProjectConfig()
	}

	// 合并配置
	resolvedVersions := make(map[string]string)
	configSource := make(map[string]string)

	// 先添加全局版本
	for tool, version := range globalConfig.GlobalVersions {
		resolvedVersions[tool] = version
		configSource[tool] = "global"
	}

	// 项目版本覆盖全局版本
	for tool, version := range projectConfig.Tools {
		resolvedVersions[tool] = version
		configSource[tool] = projectPath
	}

	return &types.EffectiveConfig{
		Global:           globalConfig,
		Project:          projectConfig,
		ResolvedVersions: resolvedVersions,
		ConfigSource:     configSource,
	}, nil
}

// WatchConfigChanges 监听配置变更
func (cm *DefaultContextManager) WatchConfigChanges(ctx context.Context, callback ConfigChangeCallback) error {
	// 这是一个简化实现，实际应该使用文件系统监听
	cm.logger.Info("Starting config change watcher")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cm.logger.Info("Config change watcher stopped")
			return ctx.Err()
		case <-ticker.C:
			// 检查配置变更（简化实现）
			// 实际应该监听文件系统事件
			cm.checkConfigChanges(callback)
		}
	}
}

// GetToolContext 获取工具上下文
func (cm *DefaultContextManager) GetToolContext(toolName, projectPath string) (*ToolContext, error) {
	cm.logger.Debugf("Getting tool context for %s in %s", toolName, projectPath)

	cacheKey := fmt.Sprintf("%s:%s", projectPath, toolName)

	// 检查缓存
	if cached := cm.getToolFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	// 获取有效配置
	effectiveConfig, err := cm.GetEffectiveConfig(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get effective config: %w", err)
	}

	// 获取工具版本和来源
	version, exists := effectiveConfig.ResolvedVersions[toolName]
	if !exists {
		return nil, fmt.Errorf("tool %s not configured", toolName)
	}

	source := effectiveConfig.ConfigSource[toolName]

	// 创建工具上下文
	toolContext := &ToolContext{
		ToolName:    toolName,
		ProjectPath: projectPath,
		Version:     version,
		Source:      source,
		WorkingDir:  projectPath,
		Environment: make(map[string]string),
		LastUpdated: time.Now(),
	}

	// 设置环境变量
	toolContext.Environment[fmt.Sprintf("%s_VERSION", strings.ToUpper(toolName))] = version
	toolContext.Environment["VMAN_TOOL"] = toolName
	toolContext.Environment["VMAN_VERSION"] = version
	toolContext.Environment["VMAN_PROJECT_PATH"] = projectPath

	// 缓存结果
	cm.setToolCache(cacheKey, toolContext)

	return toolContext, nil
}

// UpdateProjectContext 更新项目上下文
func (cm *DefaultContextManager) UpdateProjectContext(projectPath string, context *ProjectContext) error {
	cm.logger.Debugf("Updating project context for: %s", projectPath)

	context.DetectedAt = time.Now()
	cm.setProjectCache(projectPath, context)

	return nil
}

// ClearContextCache 清除上下文缓存
func (cm *DefaultContextManager) ClearContextCache() error {
	cm.projectCache = make(map[string]*ProjectContext)
	cm.toolCache = make(map[string]*ToolContext)
	cm.logger.Info("Context cache cleared")
	return nil
}

// IsProjectDirectory 检查是否为项目目录
func (cm *DefaultContextManager) IsProjectDirectory(dir string) bool {
	_, err := cm.FindProjectRoot(dir)
	return err == nil
}

// GetEnvironmentContext 获取环境上下文
func (cm *DefaultContextManager) GetEnvironmentContext() *EnvironmentContext {
	homeDir, _ := os.UserHomeDir()
	workingDir, _ := os.Getwd()

	// 获取PATH目录
	pathDirs := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))

	// 获取环境变量
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	// 获取vman目录
	vmanDir := filepath.Join(homeDir, ".vman")
	shimsDir := filepath.Join(vmanDir, "shims")

	return &EnvironmentContext{
		OS:         cm.getOS(),
		Arch:       cm.getArch(),
		Shell:      cm.getShell(),
		HomeDir:    homeDir,
		WorkingDir: workingDir,
		PathDirs:   pathDirs,
		EnvVars:    envVars,
		VmanDir:    vmanDir,
		ShimsDir:   shimsDir,
		DetectedAt: time.Now(),
	}
}

// detectProjectFeatures 检测项目特征
func (cm *DefaultContextManager) detectProjectFeatures(context *ProjectContext) {
	rootPath := context.RootPath

	// 检测项目类型
	if cm.fileExists(filepath.Join(rootPath, "package.json")) {
		context.ProjectType = "node"
		cm.detectNodeFeatures(context)
	} else if cm.fileExists(filepath.Join(rootPath, "go.mod")) {
		context.ProjectType = "go"
		cm.detectGoFeatures(context)
	} else if cm.fileExists(filepath.Join(rootPath, "requirements.txt")) || cm.fileExists(filepath.Join(rootPath, "pyproject.toml")) {
		context.ProjectType = "python"
		cm.detectPythonFeatures(context)
	} else if cm.fileExists(filepath.Join(rootPath, "Cargo.toml")) {
		context.ProjectType = "rust"
		cm.detectRustFeatures(context)
	} else if cm.fileExists(filepath.Join(rootPath, "pom.xml")) {
		context.ProjectType = "java"
		cm.detectJavaFeatures(context)
	}
}

// detectNodeFeatures 检测Node.js项目特征
func (cm *DefaultContextManager) detectNodeFeatures(context *ProjectContext) {
	// 检测包管理器
	if cm.fileExists(filepath.Join(context.RootPath, "yarn.lock")) {
		context.BuildSystem = "yarn"
	} else if cm.fileExists(filepath.Join(context.RootPath, "pnpm-lock.yaml")) {
		context.BuildSystem = "pnpm"
	} else {
		context.BuildSystem = "npm"
	}

	// TODO: 检测框架（React, Vue, Angular等）
	// TODO: 解析package.json获取依赖和脚本
}

// detectGoFeatures 检测Go项目特征
func (cm *DefaultContextManager) detectGoFeatures(context *ProjectContext) {
	context.BuildSystem = "go mod"
	// TODO: 解析go.mod获取模块信息
}

// detectPythonFeatures 检测Python项目特征
func (cm *DefaultContextManager) detectPythonFeatures(context *ProjectContext) {
	if cm.fileExists(filepath.Join(context.RootPath, "poetry.lock")) {
		context.BuildSystem = "poetry"
	} else if cm.fileExists(filepath.Join(context.RootPath, "Pipfile")) {
		context.BuildSystem = "pipenv"
	} else {
		context.BuildSystem = "pip"
	}
}

// detectRustFeatures 检测Rust项目特征
func (cm *DefaultContextManager) detectRustFeatures(context *ProjectContext) {
	context.BuildSystem = "cargo"
}

// detectJavaFeatures 检测Java项目特征
func (cm *DefaultContextManager) detectJavaFeatures(context *ProjectContext) {
	if cm.fileExists(filepath.Join(context.RootPath, "build.gradle")) {
		context.BuildSystem = "gradle"
	} else {
		context.BuildSystem = "maven"
	}
}

// findConfigFiles 查找配置文件
func (cm *DefaultContextManager) findConfigFiles(rootPath string) []string {
	var configFiles []string

	configPatterns := []string{
		".vman-version",
		".tool-versions",
		"vman.yaml",
		"vman.yml",
		".vman.yaml",
		".vman.yml",
	}

	for _, pattern := range configPatterns {
		configPath := filepath.Join(rootPath, pattern)
		if cm.fileExists(configPath) {
			configFiles = append(configFiles, configPath)
		}
	}

	return configFiles
}

// checkConfigChanges 检查配置变更（简化实现）
func (cm *DefaultContextManager) checkConfigChanges(callback ConfigChangeCallback) {
	// 这里应该实现实际的文件监听逻辑
	// 当前是空实现
}

// 缓存相关方法
func (cm *DefaultContextManager) getProjectFromCache(projectPath string) *ProjectContext {
	cached, exists := cm.projectCache[projectPath]
	if !exists {
		return nil
	}

	// 检查缓存是否过期
	if time.Since(cached.DetectedAt) > cm.cacheTimeout {
		delete(cm.projectCache, projectPath)
		return nil
	}

	return cached
}

func (cm *DefaultContextManager) setProjectCache(projectPath string, context *ProjectContext) {
	cm.projectCache[projectPath] = context
}

func (cm *DefaultContextManager) getToolFromCache(cacheKey string) *ToolContext {
	cached, exists := cm.toolCache[cacheKey]
	if !exists {
		return nil
	}

	// 检查缓存是否过期
	if time.Since(cached.LastUpdated) > cm.cacheTimeout {
		delete(cm.toolCache, cacheKey)
		return nil
	}

	return cached
}

func (cm *DefaultContextManager) setToolCache(cacheKey string, context *ToolContext) {
	cm.toolCache[cacheKey] = context
}

// 辅助方法
func (cm *DefaultContextManager) fileExists(path string) bool {
	_, err := cm.fs.Stat(path)
	return err == nil
}

func (cm *DefaultContextManager) getOS() string {
	return os.Getenv("GOOS")
}

func (cm *DefaultContextManager) getArch() string {
	return os.Getenv("GOARCH")
}

func (cm *DefaultContextManager) getShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "unknown"
	}
	return filepath.Base(shell)
}
