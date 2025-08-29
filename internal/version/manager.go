package version

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/songzhibin97/vman/pkg/utils"
)

// ProgressCallback 进度回调函数
type ProgressCallback func(*types.ProgressInfo)

// VersionInfo 版本信息 (使用types包中的定义)

// Manager 版本管理器接口
type Manager interface {
	// RegisterVersion 注册工具版本
	RegisterVersion(tool, version, path string) error

	// ListVersions 列出工具的所有版本
	ListVersions(tool string) ([]string, error)

	// GetVersionPath 获取指定版本的路径
	GetVersionPath(tool, version string) (string, error)

	// RemoveVersion 移除工具版本
	RemoveVersion(tool, version string) error

	// SetGlobalVersion 设置全局版本
	SetGlobalVersion(tool, version string) error

	// SetLocalVersion 设置项目级版本
	SetLocalVersion(tool, version string) error

	// GetCurrentVersion 获取当前使用的版本
	GetCurrentVersion(tool string) (string, error)

	// IsVersionInstalled 检查版本是否已安装
	IsVersionInstalled(tool, version string) bool

	// GetInstalledVersions 获取已安装版本列表
	GetInstalledVersions(tool string) ([]string, error)

	// ValidateVersion 验证版本格式
	ValidateVersion(version string) error

	// GetLatestVersion 获取最新版本
	GetLatestVersion(tool string) (string, error)

	// GetVersionMetadata 获取版本元数据
	GetVersionMetadata(tool, version string) (*types.VersionMetadata, error)

	// SetProjectVersion 设置项目版本（带项目路径）
	SetProjectVersion(tool, version, projectPath string) error

	// GetEffectiveVersion 获取有效版本（考虑项目和全局配置）
	GetEffectiveVersion(tool, projectPath string) (string, error)

	// ListAllTools 列出所有已安装的工具
	ListAllTools() ([]string, error)

	// InstallVersion 自动下载并安装工具版本
	InstallVersion(tool, version string) error

	// InstallVersionWithProgress 带进度显示的安装
	InstallVersionWithProgress(tool, version string, progress ProgressCallback) error

	// InstallLatestVersion 安装最新版本
	InstallLatestVersion(tool string) (string, error)

	// SearchAvailableVersions 搜索可用版本
	SearchAvailableVersions(tool string) ([]*types.VersionInfo, error)

	// IsVersionAvailable 检查版本是否可下载
	IsVersionAvailable(tool, version string) bool

	// UpdateTool 更新工具到最新版本
	UpdateTool(tool string) (string, error)
}

// DefaultManager 默认版本管理器实现
type DefaultManager struct {
	storageManager storage.Manager
	configManager  config.Manager
	fs             afero.Fs
	logger         *logrus.Logger
}

// NewManager 创建新的版本管理器
func NewManager(storageManager storage.Manager, configManager config.Manager) Manager {
	return &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             afero.NewOsFs(),
		logger:         logrus.New(),
	}
}

// NewManagerWithFs 使用指定文件系统创建版本管理器（用于测试）
func NewManagerWithFs(storageManager storage.Manager, configManager config.Manager, fs afero.Fs) Manager {
	return &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             fs,
		logger:         logrus.New(),
	}
}

// RegisterVersion 注册工具版本
func (m *DefaultManager) RegisterVersion(tool, version, sourcePath string) error {
	m.logger.Debugf("Registering version %s@%s from %s", tool, version, sourcePath)

	// 验证版本格式
	if err := m.ValidateVersion(version); err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}

	// 检查版本是否已存在
	if m.IsVersionInstalled(tool, version) {
		return fmt.Errorf("version %s@%s is already installed", tool, version)
	}

	// 创建版本目录
	if err := m.storageManager.CreateVersionDir(tool, version); err != nil {
		return fmt.Errorf("failed to create version directory: %w", err)
	}

	// 复制或移动二进制文件
	targetPath := m.storageManager.GetBinaryPath(tool, version)
	if err := m.copyBinary(sourcePath, targetPath); err != nil {
		// 清理创建的目录
		m.storageManager.RemoveVersionDir(tool, version)
		return fmt.Errorf("failed to copy binary: %w", err)
	}

	// 创建版本元数据
	metadata := &types.VersionMetadata{
		Version:     version,
		ToolName:    tool,
		InstallPath: m.storageManager.GetToolVersionPath(tool, version),
		BinaryPath:  targetPath,
		InstalledAt: time.Now(),
		InstallType: "manual",
		Source:      sourcePath,
	}

	// 计算文件大小和校验和
	if info, err := m.fs.Stat(targetPath); err == nil {
		metadata.Size = info.Size()
	}

	if checksum, err := utils.CalculateFileChecksum(targetPath); err == nil {
		metadata.Checksum = checksum
	}

	// 保存元数据
	if err := m.storageManager.SaveVersionMetadata(tool, version, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// 更新配置中的已安装版本
	if err := m.updateInstalledVersions(tool, version); err != nil {
		m.logger.Warnf("Failed to update installed versions in config: %v", err)
	}

	m.logger.Infof("Successfully registered %s@%s", tool, version)
	return nil
}

// ListVersions 列出工具的所有版本
func (m *DefaultManager) ListVersions(tool string) ([]string, error) {
	return m.storageManager.GetToolVersions(tool)
}

// GetVersionPath 获取指定版本的路径
func (m *DefaultManager) GetVersionPath(tool, version string) (string, error) {
	if !m.IsVersionInstalled(tool, version) {
		return "", fmt.Errorf("version %s@%s is not installed", tool, version)
	}
	return m.storageManager.GetToolVersionPath(tool, version), nil
}

// RemoveVersion 移除工具版本
func (m *DefaultManager) RemoveVersion(tool, version string) error {
	m.logger.Debugf("Removing version %s@%s", tool, version)

	if !m.IsVersionInstalled(tool, version) {
		return fmt.Errorf("version %s@%s is not installed", tool, version)
	}

	// 检查是否为当前使用的版本
	currentVersion, err := m.GetCurrentVersion(tool)
	if err == nil && currentVersion == version {
		return fmt.Errorf("cannot remove currently active version %s@%s", tool, version)
	}

	// 删除版本目录
	if err := m.storageManager.RemoveVersionDir(tool, version); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	// 从配置中移除已安装版本
	if err := m.removeFromInstalledVersions(tool, version); err != nil {
		m.logger.Warnf("Failed to remove version from config: %v", err)
	}

	m.logger.Infof("Successfully removed %s@%s", tool, version)
	return nil
}
