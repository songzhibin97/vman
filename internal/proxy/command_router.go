package proxy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// CommandRouter 命令路由器接口
type CommandRouter interface {
	// RouteCommand 路由命令到正确的版本
	RouteCommand(ctx context.Context, toolName string, args []string) (*RouteResult, error)

	// ExecuteCommand 执行路由后的命令
	ExecuteCommand(ctx context.Context, result *RouteResult) error

	// InterceptCommand 拦截并执行命令（组合路由和执行）
	InterceptCommand(ctx context.Context, toolName string, args []string) error

	// FindExecutable 查找可执行文件路径
	FindExecutable(toolName, version string) (string, error)

	// ValidateCommand 验证命令是否可执行
	ValidateCommand(execPath string) error

	// GetCommandInfo 获取命令信息
	GetCommandInfo(toolName string) (*CommandInfo, error)

	// RegisterCommand 注册命令
	RegisterCommand(toolName string, info *CommandInfo) error

	// UnregisterCommand 注销命令
	UnregisterCommand(toolName string) error
}

// RouteResult 路由结果
type RouteResult struct {
	ToolName       string            `json:"tool_name"`
	Version        string            `json:"version"`
	ExecutablePath string            `json:"executable_path"`
	Args           []string          `json:"args"`
	Env            map[string]string `json:"env,omitempty"`
	WorkDir        string            `json:"work_dir,omitempty"`
	Context        *RouteContext     `json:"context,omitempty"`
}

// RouteContext 路由上下文
type RouteContext struct {
	ProjectPath    string        `json:"project_path,omitempty"`
	ConfigSource   string        `json:"config_source,omitempty"` // "global", "project", "env"
	ResolvedAt     time.Time     `json:"resolved_at"`
	ResolutionTime time.Duration `json:"resolution_time"`
}

// CommandInfo 命令信息
type CommandInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Path        string            `json:"path"`
	LastUsed    time.Time         `json:"last_used,omitempty"`
	UsageCount  int               `json:"usage_count"`
	Env         map[string]string `json:"env,omitempty"`
	Aliases     []string          `json:"aliases,omitempty"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	StartTime time.Time
	PID       int
	ExitCode  int
	Duration  time.Duration
	Error     error
}

// DefaultCommandRouter 默认命令路由器实现
type DefaultCommandRouter struct {
	fs             afero.Fs
	logger         *logrus.Logger
	versionManager VersionResolver
	contextManager ContextManager
	pathManager    PathManager
	commands       map[string]*CommandInfo // 命令注册表
}

// NewCommandRouter 创建新的命令路由器
func NewCommandRouter(versionManager VersionResolver, contextManager ContextManager, pathManager PathManager) CommandRouter {
	return NewCommandRouterWithFs(afero.NewOsFs(), versionManager, contextManager, pathManager)
}

// NewCommandRouterWithFs 使用指定文件系统创建命令路由器（用于测试）
func NewCommandRouterWithFs(fs afero.Fs, versionManager VersionResolver, contextManager ContextManager, pathManager PathManager) CommandRouter {
	return &DefaultCommandRouter{
		fs:             fs,
		logger:         logrus.New(),
		versionManager: versionManager,
		contextManager: contextManager,
		pathManager:    pathManager,
		commands:       make(map[string]*CommandInfo),
	}
}

// RouteCommand 路由命令到正确的版本
func (cr *DefaultCommandRouter) RouteCommand(ctx context.Context, toolName string, args []string) (*RouteResult, error) {
	startTime := time.Now()
	cr.logger.Debugf("Routing command: %s %v", toolName, args)

	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// 解析版本
	versionResolution, err := cr.versionManager.ResolveVersion(ctx, toolName, workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve version for %s: %w", toolName, err)
	}

	// 检查版本是否已安装
	if !cr.versionManager.IsVersionInstalled(toolName, versionResolution.Version) {
		return nil, fmt.Errorf("version %s for %s is not installed. Please install it first using 'vman install %s %s'", 
			versionResolution.Version, toolName, toolName, versionResolution.Version)
	}

	// 查找可执行文件
	execPath, err := cr.FindExecutable(toolName, versionResolution.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to find executable for %s@%s: %w", toolName, versionResolution.Version, err)
	}

	// 验证可执行文件
	if err := cr.ValidateCommand(execPath); err != nil {
		return nil, fmt.Errorf("invalid executable %s: %w", execPath, err)
	}

	// 获取环境变量
	env := cr.buildEnvironment(toolName, versionResolution.Version, workDir)

	// 创建路由结果
	result := &RouteResult{
		ToolName:       toolName,
		Version:        versionResolution.Version,
		ExecutablePath: execPath,
		Args:           args,
		Env:            env,
		WorkDir:        workDir,
		Context: &RouteContext{
			ProjectPath:    versionResolution.ProjectPath,
			ConfigSource:   versionResolution.Source,
			ResolvedAt:     time.Now(),
			ResolutionTime: time.Since(startTime),
		},
	}

	cr.logger.Infof("Routed %s to %s@%s (%s)", toolName, toolName, versionResolution.Version, execPath)
	return result, nil
}

// ExecuteCommand 执行路由后的命令
func (cr *DefaultCommandRouter) ExecuteCommand(ctx context.Context, result *RouteResult) error {
	cr.logger.Debugf("Executing command: %s %v", result.ExecutablePath, result.Args)

	// 创建命令
	cmd := exec.CommandContext(ctx, result.ExecutablePath, result.Args...)

	// 设置工作目录
	if result.WorkDir != "" {
		cmd.Dir = result.WorkDir
	}

	// 设置环境变量
	cmd.Env = os.Environ()
	for key, value := range result.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// 连接标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// 记录执行信息
	var exitCode int
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			}
		}
		cr.logger.Debugf("Command execution failed: %v (exit code: %d, duration: %v)", err, exitCode, duration)
	} else {
		cr.logger.Debugf("Command executed successfully (duration: %v)", duration)
	}

	// 更新命令使用统计
	cr.updateCommandStats(result.ToolName, err == nil)

	return err
}

// InterceptCommand 拦截并执行命令（组合路由和执行）
func (cr *DefaultCommandRouter) InterceptCommand(ctx context.Context, toolName string, args []string) error {
	// 路由命令
	result, err := cr.RouteCommand(ctx, toolName, args)
	if err != nil {
		return fmt.Errorf("failed to route command: %w", err)
	}

	// 执行命令
	return cr.ExecuteCommand(ctx, result)
}

// FindExecutable 查找可执行文件路径
func (cr *DefaultCommandRouter) FindExecutable(toolName, version string) (string, error) {
	// 检查版本管理器中的路径
	versionPath, err := cr.versionManager.GetVersionPath(toolName, version)
	if err != nil {
		return "", fmt.Errorf("failed to get version path for %s@%s: %w", toolName, version, err)
	}

	// 在版本目录中查找可执行文件
	binPath := filepath.Join(versionPath, "bin", toolName)
	if cr.fileExists(binPath) {
		return binPath, nil
	}

	// 直接在版本目录中查找
	directPath := filepath.Join(versionPath, toolName)
	if cr.fileExists(directPath) {
		return directPath, nil
	}

	// Windows平台需要检查.exe扩展名
	if runtime.GOOS == "windows" {
		exePath := directPath + ".exe"
		if cr.fileExists(exePath) {
			return exePath, nil
		}
		// 也检查bin目录下的.exe文件
		binExePath := binPath + ".exe"
		if cr.fileExists(binExePath) {
			return binExePath, nil
		}
	}

	// 如果在版本目录中找不到，返回错误，不回退到PATH
	return "", fmt.Errorf("executable not found for %s@%s in version directory %s", toolName, version, versionPath)
}

// ValidateCommand 验证命令是否可执行
func (cr *DefaultCommandRouter) ValidateCommand(execPath string) error {
	// 检查文件是否存在
	info, err := cr.fs.Stat(execPath)
	if err != nil {
		return fmt.Errorf("executable not found: %w", err)
	}

	// 检查是否为普通文件
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file: %s", execPath)
	}

	// 检查是否有执行权限
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable: %s", execPath)
	}

	return nil
}

// GetCommandInfo 获取命令信息
func (cr *DefaultCommandRouter) GetCommandInfo(toolName string) (*CommandInfo, error) {
	info, exists := cr.commands[toolName]
	if !exists {
		return nil, fmt.Errorf("command not registered: %s", toolName)
	}

	return info, nil
}

// RegisterCommand 注册命令
func (cr *DefaultCommandRouter) RegisterCommand(toolName string, info *CommandInfo) error {
	cr.logger.Debugf("Registering command: %s", toolName)

	// 验证命令信息
	if info.Name == "" {
		info.Name = toolName
	}

	if info.Path != "" {
		if err := cr.ValidateCommand(info.Path); err != nil {
			return fmt.Errorf("invalid command path: %w", err)
		}
	}

	// 注册命令
	cr.commands[toolName] = info

	cr.logger.Infof("Successfully registered command: %s", toolName)
	return nil
}

// UnregisterCommand 注销命令
func (cr *DefaultCommandRouter) UnregisterCommand(toolName string) error {
	cr.logger.Debugf("Unregistering command: %s", toolName)

	if _, exists := cr.commands[toolName]; !exists {
		return fmt.Errorf("command not registered: %s", toolName)
	}

	delete(cr.commands, toolName)

	cr.logger.Infof("Successfully unregistered command: %s", toolName)
	return nil
}

// buildEnvironment 构建执行环境变量
func (cr *DefaultCommandRouter) buildEnvironment(toolName, version, workDir string) map[string]string {
	env := make(map[string]string)

	// 添加工具特定的环境变量
	env[fmt.Sprintf("%s_VERSION", strings.ToUpper(toolName))] = version
	env["VMAN_TOOL"] = toolName
	env["VMAN_VERSION"] = version
	env["VMAN_WORKDIR"] = workDir

	// 从命令信息中获取额外的环境变量
	if info, exists := cr.commands[toolName]; exists && info.Env != nil {
		for key, value := range info.Env {
			env[key] = value
		}
	}

	return env
}

// updateCommandStats 更新命令使用统计
func (cr *DefaultCommandRouter) updateCommandStats(toolName string, success bool) {
	info, exists := cr.commands[toolName]
	if !exists {
		// 如果命令未注册，创建基本信息
		info = &CommandInfo{
			Name:       toolName,
			UsageCount: 0,
		}
		cr.commands[toolName] = info
	}

	// 更新统计信息
	info.UsageCount++
	info.LastUsed = time.Now()

	cr.logger.Debugf("Updated stats for %s: usage=%d, last_used=%v", toolName, info.UsageCount, info.LastUsed)
}

// fileExists 检查文件是否存在
func (cr *DefaultCommandRouter) fileExists(path string) bool {
	_, err := cr.fs.Stat(path)
	return err == nil
}

// ExecuteDirectly 直接执行命令（绕过版本管理）
func (cr *DefaultCommandRouter) ExecuteDirectly(ctx context.Context, execPath string, args []string) error {
	cr.logger.Debugf("Executing directly: %s %v", execPath, args)

	// 验证可执行文件
	if err := cr.ValidateCommand(execPath); err != nil {
		return fmt.Errorf("invalid executable: %w", err)
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, execPath, args...)

	// 设置标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	return cmd.Run()
}

// GetRegisteredCommands 获取所有已注册的命令
func (cr *DefaultCommandRouter) GetRegisteredCommands() map[string]*CommandInfo {
	result := make(map[string]*CommandInfo)
	for name, info := range cr.commands {
		// 创建副本以避免外部修改
		infoCopy := *info
		result[name] = &infoCopy
	}
	return result
}

// ClearCommandStats 清除命令统计信息
func (cr *DefaultCommandRouter) ClearCommandStats() {
	for _, info := range cr.commands {
		info.UsageCount = 0
		info.LastUsed = time.Time{}
	}
	cr.logger.Info("Cleared all command statistics")
}
