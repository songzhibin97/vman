package proxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/version"
)

// CommandProxy 命令代理接口
type CommandProxy interface {
	// InterceptCommand 拦截并执行命令
	InterceptCommand(cmd string, args []string) error

	// ExecuteCommand 执行指定路径的命令
	ExecuteCommand(toolPath string, args []string) error

	// GenerateShim 生成命令垫片
	GenerateShim(tool, version string) error

	// RemoveShim 移除命令垫片
	RemoveShim(tool string) error

	// UpdateShims 更新所有垫片
	UpdateShims() error

	// GetShimPath 获取垫片路径
	GetShimPath(tool string) string

	// SetupProxy 设置代理环境
	SetupProxy() error

	// CleanupProxy 清理代理环境
	CleanupProxy() error

	// RehashShims 重新生成所有垫片
	RehashShims() error

	// GetProxyStatus 获取代理状态
	GetProxyStatus() *ProxyStatus
}

// ProxyStatus 代理状态
type ProxyStatus struct {
	Enabled      bool      `json:"enabled"`
	ShimsDir     string    `json:"shims_dir"`
	ShimCount    int       `json:"shim_count"`
	InPath       bool      `json:"in_path"`
	LastRehash   time.Time `json:"last_rehash"`
	ManagedTools []string  `json:"managed_tools"`
}

// DefaultCommandProxy 默认命令代理实现
type DefaultCommandProxy struct {
	fs              afero.Fs
	logger          *logrus.Logger
	configManager   config.Manager
	versionManager  version.Manager
	commandRouter   CommandRouter
	versionResolver VersionResolver
	contextManager  ContextManager
	pathManager     PathManager
	symlinkManager  SymlinkManager
	shellIntegrator ShellIntegrator
	shimsDir        string
	vmanPath        string
}

// NewCommandProxy 创建新的命令代理
func NewCommandProxy(
	configManager config.Manager,
	versionManager version.Manager,
) CommandProxy {
	return NewCommandProxyWithFs(afero.NewOsFs(), configManager, versionManager)
}

// NewCommandProxyWithFs 使用指定文件系统创建命令代理（用于测试）
func NewCommandProxyWithFs(
	fs afero.Fs,
	configManager config.Manager,
	versionManager version.Manager,
) CommandProxy {
	homeDir, _ := os.UserHomeDir()
	shimsDir := filepath.Join(homeDir, ".vman", "shims")
	vmanPath := "vman" // 假设vman在PATH中

	// 创建各个管理器
	pathManager := NewPathManagerWithFs(fs)
	symlinkManager := NewSymlinkManagerWithFs(fs)
	shellIntegrator := NewShellIntegratorWithFs(fs)
	contextManager := NewContextManagerWithFs(fs, configManager)
	versionResolver := NewVersionResolverWithFs(fs, configManager, versionManager)
	commandRouter := NewCommandRouterWithFs(fs, versionResolver, contextManager, pathManager)

	return &DefaultCommandProxy{
		fs:              fs,
		logger:          logrus.New(),
		configManager:   configManager,
		versionManager:  versionManager,
		commandRouter:   commandRouter,
		versionResolver: versionResolver,
		contextManager:  contextManager,
		pathManager:     pathManager,
		symlinkManager:  symlinkManager,
		shellIntegrator: shellIntegrator,
		shimsDir:        shimsDir,
		vmanPath:        vmanPath,
	}
}

// InterceptCommand 拦截并执行命令
func (cp *DefaultCommandProxy) InterceptCommand(cmd string, args []string) error {
	cp.logger.Debugf("Intercepting command: %s %v", cmd, args)

	ctx := context.Background()
	return cp.commandRouter.InterceptCommand(ctx, cmd, args)
}

// ExecuteCommand 执行指定路径的命令
func (cp *DefaultCommandProxy) ExecuteCommand(toolPath string, args []string) error {
	cp.logger.Debugf("Executing command: %s %v", toolPath, args)

	// 创建路由结果并执行
	result := &RouteResult{
		ExecutablePath: toolPath,
		Args:           args,
		WorkDir:        "",
		Env:            make(map[string]string),
	}

	ctx := context.Background()
	return cp.commandRouter.ExecuteCommand(ctx, result)
}

// GenerateShim 生成命令垫片
func (cp *DefaultCommandProxy) GenerateShim(tool, version string) error {
	cp.logger.Infof("Generating shim for %s@%s", tool, version)

	// 获取工具的二进制路径
	binaryPath, err := cp.versionManager.GetVersionPath(tool, version)
	if err != nil {
		return fmt.Errorf("failed to get version path: %w", err)
	}

	// 生成shim文件
	shimPath := filepath.Join(cp.shimsDir, tool)
	if err := cp.shellIntegrator.GenerateShim(tool, shimPath, cp.vmanPath); err != nil {
		return fmt.Errorf("failed to generate shim script: %w", err)
	}

	// 创建符号链接
	if err := cp.symlinkManager.CreateToolSymlinks(tool, version, binaryPath, cp.shimsDir); err != nil {
		cp.logger.Warnf("Failed to create symlinks for %s: %v", tool, err)
		// 继续执行，因为shim文件已经创建
	}

	cp.logger.Infof("Successfully generated shim for %s@%s", tool, version)
	return nil
}

// RemoveShim 移除命令垫片
func (cp *DefaultCommandProxy) RemoveShim(tool string) error {
	cp.logger.Infof("Removing shim for: %s", tool)

	// 移除shim文件
	shimPath := filepath.Join(cp.shimsDir, tool)
	if err := cp.fs.Remove(shimPath); err != nil && !os.IsNotExist(err) {
		cp.logger.Warnf("Failed to remove shim file %s: %v", shimPath, err)
	}

	// 移除符号链接
	if err := cp.symlinkManager.RemoveToolSymlinks(tool, cp.shimsDir); err != nil {
		cp.logger.Warnf("Failed to remove symlinks for %s: %v", tool, err)
	}

	cp.logger.Infof("Successfully removed shim for: %s", tool)
	return nil
}

// UpdateShims 更新所有垫片
func (cp *DefaultCommandProxy) UpdateShims() error {
	return cp.RehashShims()
}

// GetShimPath 获取垫片路径
func (cp *DefaultCommandProxy) GetShimPath(tool string) string {
	return filepath.Join(cp.shimsDir, tool)
}

// SetupProxy 设置代理环境
func (cp *DefaultCommandProxy) SetupProxy() error {
	cp.logger.Info("Setting up proxy environment")

	// 设置shims目录到PATH
	if err := cp.pathManager.SetupShimPath(cp.shimsDir); err != nil {
		return fmt.Errorf("failed to setup shim path: %w", err)
	}

	// 安装shell钩子
	shellType := cp.shellIntegrator.DetectShell()
	if err := cp.shellIntegrator.InstallShellHook(shellType, cp.vmanPath); err != nil {
		cp.logger.Warnf("Failed to install shell hook: %v", err)
		// 继续执行，因为PATH已经设置
	}

	// 生成所有已安装工具的shims
	if err := cp.RehashShims(); err != nil {
		return fmt.Errorf("failed to rehash shims: %w", err)
	}

	cp.logger.Info("Proxy environment setup completed")
	return nil
}

// CleanupProxy 清理代理环境
func (cp *DefaultCommandProxy) CleanupProxy() error {
	cp.logger.Info("Cleaning up proxy environment")

	// 从PATH中移除shims目录
	if err := cp.pathManager.CleanupShimPath(cp.shimsDir); err != nil {
		cp.logger.Warnf("Failed to cleanup shim path: %v", err)
	}

	// 卸载shell钩子
	shellType := cp.shellIntegrator.DetectShell()
	if err := cp.shellIntegrator.UninstallShellHook(shellType); err != nil {
		cp.logger.Warnf("Failed to uninstall shell hook: %v", err)
	}

	// 清理所有shims
	if err := cp.clearAllShims(); err != nil {
		cp.logger.Warnf("Failed to clear all shims: %v", err)
	}

	cp.logger.Info("Proxy environment cleanup completed")
	return nil
}

// RehashShims 重新生成所有垫片
func (cp *DefaultCommandProxy) RehashShims() error {
	cp.logger.Info("Rehashing all shims")

	// 确保shims目录存在
	if err := cp.fs.MkdirAll(cp.shimsDir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	// 清理现有的shims
	if err := cp.clearAllShims(); err != nil {
		cp.logger.Warnf("Failed to clear existing shims: %v", err)
	}

	// 获取所有已安装的工具
	tools, err := cp.versionManager.ListAllTools()
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// 为每个工具生成shim
	for _, tool := range tools {
		// 获取当前版本
		currentVersion, err := cp.versionManager.GetCurrentVersion(tool)
		if err != nil {
			cp.logger.Warnf("Failed to get current version for %s: %v", tool, err)
			
			// 尝试获取已安装版本列表作为fallback
			installedVersions, verErr := cp.versionManager.GetInstalledVersions(tool)
			if verErr != nil || len(installedVersions) == 0 {
				cp.logger.Warnf("No installed versions found for %s, skipping shim generation", tool)
				continue
			}
			
			// 使用第一个已安装的版本
			currentVersion = installedVersions[0]
			cp.logger.Infof("Using fallback version %s for %s", currentVersion, tool)
		}

		// 生成shim
		if err := cp.GenerateShim(tool, currentVersion); err != nil {
			cp.logger.Warnf("Failed to generate shim for %s@%s: %v", tool, currentVersion, err)
		}
	}

	cp.logger.Infof("Rehashed shims for %d tools", len(tools))
	return nil
}

// GetProxyStatus 获取代理状态
func (cp *DefaultCommandProxy) GetProxyStatus() *ProxyStatus {
	// 检查shims目录是否在PATH中
	inPath := cp.pathManager.IsInPath(cp.shimsDir)

	// 统计shim数量
	shimCount := 0
	var managedTools []string

	if entries, err := afero.ReadDir(cp.fs, cp.shimsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				shimCount++
				managedTools = append(managedTools, entry.Name())
			}
		}
	}

	return &ProxyStatus{
		Enabled:      inPath,
		ShimsDir:     cp.shimsDir,
		ShimCount:    shimCount,
		InPath:       inPath,
		LastRehash:   time.Now(), // TODO: 实际跟踪上次rehash时间
		ManagedTools: managedTools,
	}
}

// clearAllShims 清理所有shims
func (cp *DefaultCommandProxy) clearAllShims() error {
	if exists, _ := afero.Exists(cp.fs, cp.shimsDir); !exists {
		return nil
	}

	entries, err := afero.ReadDir(cp.fs, cp.shimsDir)
	if err != nil {
		return fmt.Errorf("failed to read shims directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(cp.shimsDir, entry.Name())
		if err := cp.fs.Remove(entryPath); err != nil {
			cp.logger.Warnf("Failed to remove shim %s: %v", entryPath, err)
		}
	}

	return nil
}
