package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/pkg/types"
)

// Manager 存储管理器接口
type Manager interface {
	// GetToolsDir 获取工具存储目录
	GetToolsDir() string

	// GetConfigDir 获取配置目录
	GetConfigDir() string

	// GetShimsDir 获取垫片目录
	GetShimsDir() string

	// GetCacheDir 获取缓存目录
	GetCacheDir() string

	// GetSourcesDir 获取下载源目录
	GetSourcesDir() string

	// EnsureDirectories 确保所有必要目录存在
	EnsureDirectories() error

	// CleanupOrphaned 清理孤立的文件和目录
	CleanupOrphaned() error

	// GetToolVersionPath 获取工具版本的存储路径
	GetToolVersionPath(tool, version string) string

	// GetToolVersions 获取工具的所有已安装版本
	GetToolVersions(tool string) ([]string, error)

	// GetVersionsDir 获取版本存储根目录
	GetVersionsDir() string

	// GetBinDir 获取二进制文件目录
	GetBinDir() string

	// GetLogsDir 获取日志目录
	GetLogsDir() string

	// GetTempDir 获取临时目录
	GetTempDir() string

	// CreateVersionDir 创建版本目录
	CreateVersionDir(tool, version string) error

	// RemoveVersionDir 删除版本目录
	RemoveVersionDir(tool, version string) error

	// IsVersionInstalled 检查版本是否已安装
	IsVersionInstalled(tool, version string) bool

	// GetVersionMetadataPath 获取版本元数据文件路径
	GetVersionMetadataPath(tool, version string) string

	// SaveVersionMetadata 保存版本元数据
	SaveVersionMetadata(tool, version string, metadata *types.VersionMetadata) error

	// LoadVersionMetadata 加载版本元数据
	LoadVersionMetadata(tool, version string) (*types.VersionMetadata, error)

	// GetBinaryPath 获取工具二进制文件路径
	GetBinaryPath(tool, version string) string
}

// FilesystemManager 文件系统存储管理器实现
type FilesystemManager struct {
	fs     afero.Fs
	paths  *types.ConfigPaths
	logger *logrus.Logger
}

// NewManager 创建新的存储管理器
func NewManager() Manager {
	homeDir, _ := os.UserHomeDir()
	configPaths := types.DefaultConfigPaths(homeDir)
	return NewFilesystemManager(configPaths)
}

// NewFilesystemManager 创建新的文件系统存储管理器
func NewFilesystemManager(configPaths *types.ConfigPaths) Manager {
	return &FilesystemManager{
		fs:     afero.NewOsFs(),
		paths:  configPaths,
		logger: logrus.New(),
	}
}

// NewFilesystemManagerWithFs 使用指定文件系统创建存储管理器（用于测试）
func NewFilesystemManagerWithFs(fs afero.Fs, configPaths *types.ConfigPaths) Manager {
	return &FilesystemManager{
		fs:     fs,
		paths:  configPaths,
		logger: logrus.New(),
	}
}

// GetToolsDir 获取工具存储目录
func (f *FilesystemManager) GetToolsDir() string {
	return f.paths.ToolsDir
}

// GetConfigDir 获取配置目录
func (f *FilesystemManager) GetConfigDir() string {
	return f.paths.ConfigDir
}

// GetShimsDir 获取垫片目录
func (f *FilesystemManager) GetShimsDir() string {
	return f.paths.ShimsDir
}

// GetCacheDir 获取缓存目录
func (f *FilesystemManager) GetCacheDir() string {
	return f.paths.CacheDir
}

// GetSourcesDir 获取下载源目录
func (f *FilesystemManager) GetSourcesDir() string {
	// 如果没有定义SourcesDir，则使用cache/sources
	return filepath.Join(f.paths.CacheDir, "sources")
}

// GetVersionsDir 获取版本存储根目录
func (f *FilesystemManager) GetVersionsDir() string {
	return f.paths.VersionsDir
}

// GetBinDir 获取二进制文件目录
func (f *FilesystemManager) GetBinDir() string {
	return f.paths.BinDir
}

// GetLogsDir 获取日志目录
func (f *FilesystemManager) GetLogsDir() string {
	return f.paths.LogsDir
}

// GetTempDir 获取临时目录
func (f *FilesystemManager) GetTempDir() string {
	return f.paths.TempDir
}

// EnsureDirectories 确保所有必要目录存在
func (f *FilesystemManager) EnsureDirectories() error {
	f.logger.Debug("Ensuring storage directories exist")

	dirs := []string{
		f.paths.ConfigDir,
		f.paths.ToolsDir,
		f.paths.BinDir,
		f.paths.ShimsDir,
		f.paths.VersionsDir,
		f.paths.LogsDir,
		f.paths.CacheDir,
		f.paths.TempDir,
		f.GetSourcesDir(),
	}

	for _, dir := range dirs {
		if err := f.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		f.logger.Debugf("Created directory: %s", dir)
	}

	f.logger.Debug("All storage directories ensured")
	return nil
}

// GetToolVersionPath 获取工具版本的存储路径
func (f *FilesystemManager) GetToolVersionPath(tool, version string) string {
	return filepath.Join(f.paths.VersionsDir, tool, version)
}

// GetVersionMetadataPath 获取版本元数据文件路径
func (f *FilesystemManager) GetVersionMetadataPath(tool, version string) string {
	return filepath.Join(f.GetToolVersionPath(tool, version), "metadata.json")
}

// GetBinaryPath 获取工具二进制文件路径
func (f *FilesystemManager) GetBinaryPath(tool, version string) string {
	return filepath.Join(f.GetToolVersionPath(tool, version), "bin", tool)
}

// CreateVersionDir 创建版本目录
func (f *FilesystemManager) CreateVersionDir(tool, version string) error {
	f.logger.Debugf("Creating version directory for %s@%s", tool, version)

	versionPath := f.GetToolVersionPath(tool, version)
	binPath := filepath.Join(versionPath, "bin")

	// 创建版本目录和bin子目录
	if err := f.fs.MkdirAll(binPath, 0755); err != nil {
		return fmt.Errorf("failed to create version directory %s: %w", versionPath, err)
	}

	f.logger.Debugf("Created version directory: %s", versionPath)
	return nil
}

// RemoveVersionDir 删除版本目录
func (f *FilesystemManager) RemoveVersionDir(tool, version string) error {
	f.logger.Debugf("Removing version directory for %s@%s", tool, version)

	versionPath := f.GetToolVersionPath(tool, version)
	if err := f.fs.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove version directory %s: %w", versionPath, err)
	}

	f.logger.Debugf("Removed version directory: %s", versionPath)
	return nil
}

// IsVersionInstalled 检查版本是否已安装
func (f *FilesystemManager) IsVersionInstalled(tool, version string) bool {
	versionPath := f.GetToolVersionPath(tool, version)
	binaryPath := f.GetBinaryPath(tool, version)

	// 检查版本目录是否存在
	if exists, err := afero.DirExists(f.fs, versionPath); err != nil || !exists {
		return false
	}

	// 检查二进制文件是否存在
	if exists, err := afero.Exists(f.fs, binaryPath); err != nil || !exists {
		return false
	}

	return true
}

// GetToolVersions 获取工具的所有已安装版本
func (f *FilesystemManager) GetToolVersions(tool string) ([]string, error) {
	f.logger.Debugf("Getting versions for tool: %s", tool)

	toolDir := filepath.Join(f.paths.VersionsDir, tool)
	if exists, err := afero.DirExists(f.fs, toolDir); err != nil {
		return nil, fmt.Errorf("failed to check tool directory %s: %w", toolDir, err)
	} else if !exists {
		return []string{}, nil
	}

	entries, err := afero.ReadDir(f.fs, toolDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool directory %s: %w", toolDir, err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			// 验证这是一个有效的版本目录（包含二进制文件）
			if f.IsVersionInstalled(tool, version) {
				versions = append(versions, version)
			}
		}
	}

	f.logger.Debugf("Found %d versions for tool %s: %v", len(versions), tool, versions)
	return versions, nil
}

// SaveVersionMetadata 保存版本元数据
func (f *FilesystemManager) SaveVersionMetadata(tool, version string, metadata *types.VersionMetadata) error {
	f.logger.Debugf("Saving metadata for %s@%s", tool, version)

	metadataPath := f.GetVersionMetadataPath(tool, version)
	metadataDir := filepath.Dir(metadataPath)

	// 确保目录存在
	if err := f.fs.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory %s: %w", metadataDir, err)
	}

	// 序列化元数据
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// 写入文件
	if err := afero.WriteFile(f.fs, metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file %s: %w", metadataPath, err)
	}

	f.logger.Debugf("Saved metadata to: %s", metadataPath)
	return nil
}

// LoadVersionMetadata 加载版本元数据
func (f *FilesystemManager) LoadVersionMetadata(tool, version string) (*types.VersionMetadata, error) {
	f.logger.Debugf("Loading metadata for %s@%s", tool, version)

	metadataPath := f.GetVersionMetadataPath(tool, version)
	if exists, err := afero.Exists(f.fs, metadataPath); err != nil {
		return nil, fmt.Errorf("failed to check metadata file %s: %w", metadataPath, err)
	} else if !exists {
		return nil, fmt.Errorf("metadata file not found: %s", metadataPath)
	}

	data, err := afero.ReadFile(f.fs, metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file %s: %w", metadataPath, err)
	}

	var metadata types.VersionMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	f.logger.Debugf("Loaded metadata from: %s", metadataPath)
	return &metadata, nil
}

// CleanupOrphaned 清理孤立的文件和目录
func (f *FilesystemManager) CleanupOrphaned() error {
	f.logger.Debug("Starting orphaned files cleanup")

	// 清理版本目录中没有二进制文件的目录
	if err := f.cleanupOrphanedVersions(); err != nil {
		return fmt.Errorf("failed to cleanup orphaned versions: %w", err)
	}

	// 清理临时目录
	if err := f.cleanupTempDir(); err != nil {
		return fmt.Errorf("failed to cleanup temp directory: %w", err)
	}

	f.logger.Debug("Orphaned files cleanup completed")
	return nil
}

// cleanupOrphanedVersions 清理孤立的版本目录
func (f *FilesystemManager) cleanupOrphanedVersions() error {
	entries, err := afero.ReadDir(f.fs, f.paths.VersionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 版本目录不存在，跳过清理
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			toolName := entry.Name()
			toolDir := filepath.Join(f.paths.VersionsDir, toolName)

			// 检查工具目录下的版本目录
			versionEntries, err := afero.ReadDir(f.fs, toolDir)
			if err != nil {
				f.logger.Warnf("Failed to read tool directory %s: %v", toolDir, err)
				continue
			}

			for _, versionEntry := range versionEntries {
				if versionEntry.IsDir() {
					version := versionEntry.Name()
					// 如果版本未正确安装，删除该目录
					if !f.IsVersionInstalled(toolName, version) {
						orphanedPath := filepath.Join(toolDir, version)
						f.logger.Infof("Removing orphaned version directory: %s", orphanedPath)
						if err := f.fs.RemoveAll(orphanedPath); err != nil {
							f.logger.Warnf("Failed to remove orphaned directory %s: %v", orphanedPath, err)
						}
					}
				}
			}

			// 如果工具目录为空，删除它
			remaining, err := afero.ReadDir(f.fs, toolDir)
			if err == nil && len(remaining) == 0 {
				f.logger.Infof("Removing empty tool directory: %s", toolDir)
				if err := f.fs.Remove(toolDir); err != nil {
					f.logger.Warnf("Failed to remove empty tool directory %s: %v", toolDir, err)
				}
			}
		}
	}

	return nil
}

// cleanupTempDir 清理临时目录
func (f *FilesystemManager) cleanupTempDir() error {
	if exists, err := afero.DirExists(f.fs, f.paths.TempDir); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// 删除超过24小时的临时文件
	entries, err := afero.ReadDir(f.fs, f.paths.TempDir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		filePath := filepath.Join(f.paths.TempDir, entry.Name())
		if now.Sub(entry.ModTime()) > 24*time.Hour {
			f.logger.Infof("Removing old temp file: %s", filePath)
			if err := f.fs.RemoveAll(filePath); err != nil {
				f.logger.Warnf("Failed to remove temp file %s: %v", filePath, err)
			}
		}
	}

	return nil
}
