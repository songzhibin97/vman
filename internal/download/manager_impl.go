package download

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/songzhibin97/vman/pkg/utils"
)

// DefaultManager 默认下载管理器实现
type DefaultManager struct {
	storageManager storage.Manager
	configManager  config.Manager
	fs             afero.Fs
	logger         *logrus.Logger
	strategies     map[string]Strategy
	mu             sync.RWMutex
}

// NewManager 创建新的下载管理器
func NewManager(storageManager storage.Manager, configManager config.Manager) Manager {
	return &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             afero.NewOsFs(),
		logger:         logrus.New(),
		strategies:     make(map[string]Strategy),
	}
}

// NewManagerWithFs 使用指定文件系统创建下载管理器（用于测试）
func NewManagerWithFs(storageManager storage.Manager, configManager config.Manager, fs afero.Fs) Manager {
	return &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             fs,
		logger:         logrus.New(),
		strategies:     make(map[string]Strategy),
	}
}

// Download 下载并安装工具版本
func (m *DefaultManager) Download(ctx context.Context, tool, version string, options *DownloadOptions) error {
	m.logger.Debugf("开始下载 %s@%s", tool, version)

	// 获取下载策略
	strategy, err := m.GetDownloadStrategy(tool)
	if err != nil {
		return fmt.Errorf("获取下载策略失败: %w", err)
	}

	// 验证版本是否存在
	if err := strategy.ValidateVersion(ctx, version); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			Cause:   err,
			Code:    VersionNotFound,
		}
	}

	// 获取下载信息
	downloadInfo, err := strategy.GetDownloadInfo(ctx, version)
	if err != nil {
		return fmt.Errorf("获取下载信息失败: %w", err)
	}

	// 设置默认选项
	if options == nil {
		options = &DownloadOptions{}
	}
	m.setDefaultOptions(options)

	// 创建临时目录
	tempDir := filepath.Join(m.storageManager.GetTempDir(), fmt.Sprintf("%s-%s-%d", tool, version, time.Now().Unix()))
	if err := m.fs.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() {
		if !options.KeepDownload {
			m.fs.RemoveAll(tempDir)
		}
	}()

	// 下载文件
	downloadPath := filepath.Join(tempDir, downloadInfo.Filename)
	if err := strategy.Download(ctx, downloadInfo.URL, downloadPath, options); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			URL:     downloadInfo.URL,
			Cause:   err,
			Code:    NetworkError,
		}
	}

	// 验证校验和
	if !options.SkipChecksum && downloadInfo.Checksum != "" {
		if err := m.validateChecksum(downloadPath, downloadInfo.Checksum); err != nil {
			return &DownloadError{
				Tool:    tool,
				Version: version,
				URL:     downloadInfo.URL,
				Cause:   err,
				Code:    ChecksumMismatch,
			}
		}
	}

	// 提取文件
	extractDir := filepath.Join(tempDir, "extracted")
	if err := m.fs.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("创建提取目录失败: %w", err)
	}

	if err := strategy.ExtractArchive(downloadPath, extractDir); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			Cause:   err,
			Code:    ExtractionError,
		}
	}

	// 安装到版本目录
	if err := m.installVersion(tool, version, extractDir); err != nil {
		return fmt.Errorf("安装版本失败: %w", err)
	}

	m.logger.Infof("成功下载并安装 %s@%s", tool, version)
	return nil
}

// DownloadWithProgress 带进度显示的下载
func (m *DefaultManager) DownloadWithProgress(ctx context.Context, tool, version string, options *DownloadOptions, progress ProgressCallback) error {
	m.logger.Debugf("开始下载 %s@%s (带进度)", tool, version)

	strategy, err := m.GetDownloadStrategy(tool)
	if err != nil {
		return fmt.Errorf("获取下载策略失败: %w", err)
	}

	// 验证版本
	if err := strategy.ValidateVersion(ctx, version); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			Cause:   err,
			Code:    VersionNotFound,
		}
	}

	downloadInfo, err := strategy.GetDownloadInfo(ctx, version)
	if err != nil {
		return fmt.Errorf("获取下载信息失败: %w", err)
	}

	if options == nil {
		options = &DownloadOptions{}
	}
	m.setDefaultOptions(options)

	// 创建临时目录
	tempDir := filepath.Join(m.storageManager.GetTempDir(), fmt.Sprintf("%s-%s-%d", tool, version, time.Now().Unix()))
	if err := m.fs.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() {
		if !options.KeepDownload {
			m.fs.RemoveAll(tempDir)
		}
	}()

	// 带进度下载
	downloadPath := filepath.Join(tempDir, downloadInfo.Filename)
	if err := strategy.DownloadWithProgress(ctx, downloadInfo.URL, downloadPath, options, progress); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			URL:     downloadInfo.URL,
			Cause:   err,
			Code:    NetworkError,
		}
	}

	// 验证和安装步骤与普通下载相同
	if !options.SkipChecksum && downloadInfo.Checksum != "" {
		if err := m.validateChecksum(downloadPath, downloadInfo.Checksum); err != nil {
			return &DownloadError{
				Tool:    tool,
				Version: version,
				URL:     downloadInfo.URL,
				Cause:   err,
				Code:    ChecksumMismatch,
			}
		}
	}

	extractDir := filepath.Join(tempDir, "extracted")
	if err := m.fs.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("创建提取目录失败: %w", err)
	}

	if err := strategy.ExtractArchive(downloadPath, extractDir); err != nil {
		return &DownloadError{
			Tool:    tool,
			Version: version,
			Cause:   err,
			Code:    ExtractionError,
		}
	}

	if err := m.installVersion(tool, version, extractDir); err != nil {
		return fmt.Errorf("安装版本失败: %w", err)
	}

	m.logger.Infof("成功下载并安装 %s@%s", tool, version)
	return nil
}

// GetDownloadStrategy 获取下载策略
func (m *DefaultManager) GetDownloadStrategy(tool string) (Strategy, error) {
	m.mu.RLock()
	strategy, exists := m.strategies[tool]
	m.mu.RUnlock()

	if exists {
		return strategy, nil
	}

	// 从配置中加载工具元数据
	metadata, err := m.configManager.LoadToolConfig(tool)
	if err != nil {
		return nil, fmt.Errorf("加载工具配置失败: %w", err)
	}

	// 创建下载策略
	strategy, err = m.createStrategy(metadata)
	if err != nil {
		return nil, fmt.Errorf("创建下载策略失败: %w", err)
	}

	// 缓存策略
	m.mu.Lock()
	m.strategies[tool] = strategy
	m.mu.Unlock()

	return strategy, nil
}

// AddSource 添加下载源
func (m *DefaultManager) AddSource(tool string, metadata *types.ToolMetadata) error {
	m.logger.Debugf("添加下载源: %s", tool)

	// 验证元数据
	if err := m.validateToolMetadata(metadata); err != nil {
		return fmt.Errorf("工具元数据验证失败: %w", err)
	}

	// 保存工具配置到文件
	toolConfigPath := filepath.Join(m.storageManager.GetSourcesDir(), tool+".toml")
	if err := m.saveToolMetadata(toolConfigPath, metadata); err != nil {
		return fmt.Errorf("保存工具配置失败: %w", err)
	}

	// 创建并缓存策略
	strategy, err := m.createStrategy(metadata)
	if err != nil {
		return fmt.Errorf("创建下载策略失败: %w", err)
	}

	m.mu.Lock()
	m.strategies[tool] = strategy
	m.mu.Unlock()

	m.logger.Infof("成功添加下载源: %s", tool)
	return nil
}

// RemoveSource 移除下载源
func (m *DefaultManager) RemoveSource(tool string) error {
	m.logger.Debugf("移除下载源: %s", tool)

	// 删除配置文件
	toolConfigPath := filepath.Join(m.storageManager.GetSourcesDir(), tool+".toml")
	if err := m.fs.Remove(toolConfigPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除工具配置文件失败: %w", err)
	}

	// 移除缓存的策略
	m.mu.Lock()
	delete(m.strategies, tool)
	m.mu.Unlock()

	m.logger.Infof("成功移除下载源: %s", tool)
	return nil
}

// ListSources 列出所有下载源
func (m *DefaultManager) ListSources() ([]string, error) {
	sourcesDir := m.storageManager.GetSourcesDir()

	entries, err := afero.ReadDir(m.fs, sourcesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("读取下载源目录失败: %w", err)
	}

	var sources []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
			toolName := entry.Name()[:len(entry.Name())-5] // 移除.toml扩展名
			sources = append(sources, toolName)
		}
	}

	return sources, nil
}

// UpdateSources 更新下载源信息
func (m *DefaultManager) UpdateSources(ctx context.Context) error {
	m.logger.Debug("更新所有下载源信息")

	sources, err := m.ListSources()
	if err != nil {
		return fmt.Errorf("获取下载源列表失败: %w", err)
	}

	for _, tool := range sources {
		strategy, err := m.GetDownloadStrategy(tool)
		if err != nil {
			m.logger.Warnf("获取 %s 的下载策略失败: %v", tool, err)
			continue
		}

		// 这里可以添加更新版本列表缓存等操作
		_, err = strategy.GetLatestVersion(ctx)
		if err != nil {
			m.logger.Warnf("获取 %s 的最新版本失败: %v", tool, err)
		}
	}

	m.logger.Info("下载源信息更新完成")
	return nil
}

// SearchVersions 搜索可用版本
func (m *DefaultManager) SearchVersions(ctx context.Context, tool string) ([]*types.VersionInfo, error) {
	strategy, err := m.GetDownloadStrategy(tool)
	if err != nil {
		return nil, fmt.Errorf("获取下载策略失败: %w", err)
	}

	return strategy.ListVersions(ctx)
}

// GetVersionInfo 获取版本详细信息
func (m *DefaultManager) GetVersionInfo(ctx context.Context, tool, version string) (*types.VersionInfo, error) {
	strategy, err := m.GetDownloadStrategy(tool)
	if err != nil {
		return nil, fmt.Errorf("获取下载策略失败: %w", err)
	}

	if err := strategy.ValidateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("版本验证失败: %w", err)
	}

	downloadInfo, err := strategy.GetDownloadInfo(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("获取下载信息失败: %w", err)
	}

	platform := types.GetCurrentPlatform()
	downloads := make(map[string]types.DownloadInfo)
	downloads[platform.GetPlatformKey()] = *downloadInfo

	return &types.VersionInfo{
		Version:   version,
		Downloads: downloads,
	}, nil
}

// ClearCache 清理下载缓存
func (m *DefaultManager) ClearCache(tool string) error {
	cacheDir := filepath.Join(m.storageManager.GetCacheDir(), tool)
	if err := m.fs.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理缓存失败: %w", err)
	}
	return nil
}

// GetCacheSize 获取缓存大小
func (m *DefaultManager) GetCacheSize(tool string) (int64, error) {
	cacheDir := filepath.Join(m.storageManager.GetCacheDir(), tool)
	return m.calculateDirSize(cacheDir)
}

// ResumeDownload 恢复下载
func (m *DefaultManager) ResumeDownload(ctx context.Context, tool, version string, options *DownloadOptions) error {
	// 设置恢复选项
	if options == nil {
		options = &DownloadOptions{}
	}
	options.Resume = true

	return m.Download(ctx, tool, version, options)
}

// 私有方法

// setDefaultOptions 设置默认选项
func (m *DefaultManager) setDefaultOptions(options *DownloadOptions) {
	config, _ := m.configManager.LoadGlobal()

	if options.Timeout == 0 {
		options.Timeout = int(config.Settings.Download.Timeout.Seconds())
	}
	if options.Retries == 0 {
		options.Retries = config.Settings.Download.Retries
	}
	if options.TempDir == "" {
		options.TempDir = m.storageManager.GetTempDir()
	}
}

// validateChecksum 验证校验和
func (m *DefaultManager) validateChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // 没有期望的校验和，跳过验证
	}

	m.logger.Debugf("验证文件校验和: %s", filePath)

	// 计算文件的SHA256
	actualChecksum, err := utils.CalculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("计算文件校验和失败: %w", err)
	}

	// 比较校验和
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("校验和不匹配: 期望 %s, 实际 %s", expectedChecksum, actualChecksum)
	}

	m.logger.Debugf("校验和验证通过: %s", actualChecksum)
	return nil
}

// installVersion 安装版本到目标目录
func (m *DefaultManager) installVersion(tool, version, extractDir string) error {
	// 创建版本目录
	if err := m.storageManager.CreateVersionDir(tool, version); err != nil {
		return fmt.Errorf("创建版本目录失败: %w", err)
	}

	targetPath := m.storageManager.GetToolVersionPath(tool, version)

	// 复制文件到目标目录
	return m.copyDirectory(extractDir, targetPath)
}

// createStrategy 创建下载策略
func (m *DefaultManager) createStrategy(metadata *types.ToolMetadata) (Strategy, error) {
	switch metadata.DownloadConfig.Type {
	case "github":
		return NewGitHubStrategy(metadata, m.fs, m.logger), nil
	case "direct":
		return NewDirectStrategy(metadata, m.fs, m.logger), nil
	case "archive":
		return NewArchiveStrategy(metadata, m.fs, m.logger), nil
	default:
		return nil, fmt.Errorf("不支持的下载类型: %s", metadata.DownloadConfig.Type)
	}
}

// validateToolMetadata 验证工具元数据
func (m *DefaultManager) validateToolMetadata(metadata *types.ToolMetadata) error {
	if metadata.Name == "" {
		return fmt.Errorf("工具名称不能为空")
	}
	if metadata.DownloadConfig.Type == "" {
		return fmt.Errorf("下载类型不能为空")
	}
	return nil
}

// saveToolMetadata 保存工具元数据到文件
func (m *DefaultManager) saveToolMetadata(path string, metadata *types.ToolMetadata) error {
	// 使用JSON暂时替代TOML，后续可以改为TOML
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化元数据失败: %w", err)
	}

	// 确保目录存在
	if err := m.fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	return afero.WriteFile(m.fs, path, data, 0644)
}

// copyDirectory 复制目录
func (m *DefaultManager) copyDirectory(src, dst string) error {
	// 确保目标目录存在
	if err := m.fs.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 遍历源目录
	return afero.Walk(m.fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// 创建目录
			return m.fs.MkdirAll(targetPath, info.Mode())
		} else {
			// 复制文件
			return m.copyFile(path, targetPath)
		}
	})
}

// copyFile 复制文件
func (m *DefaultManager) copyFile(src, dst string) error {
	srcFile, err := m.fs.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 确保目标目录存在
	if err := m.fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	dstFile, err := m.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// calculateDirSize 计算目录大小
func (m *DefaultManager) calculateDirSize(dirPath string) (int64, error) {
	var totalSize int64

	err := afero.Walk(m.fs, dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}
