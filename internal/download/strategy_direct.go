package download

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

// DirectStrategy 直接URL下载策略
type DirectStrategy struct {
	metadata   *types.ToolMetadata
	fs         afero.Fs
	logger     *logrus.Logger
	downloader Downloader
	extractor  *PackageProcessor
	client     *http.Client
}

// NewDirectStrategy 创建直接URL下载策略
func NewDirectStrategy(metadata *types.ToolMetadata, fs afero.Fs, logger *logrus.Logger) Strategy {
	return &DirectStrategy{
		metadata:   metadata,
		fs:         fs,
		logger:     logger,
		downloader: NewHTTPDownloader(fs, logger),
		extractor:  NewPackageProcessor(fs, logger),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDownloadInfo 获取下载信息
func (d *DirectStrategy) GetDownloadInfo(ctx context.Context, version string) (*types.DownloadInfo, error) {
	d.logger.Debugf("获取直接URL下载信息: %s@%s", d.metadata.Name, version)

	url, err := d.buildDownloadURL(version)
	if err != nil {
		return nil, fmt.Errorf("构建下载URL失败: %w", err)
	}

	// 获取文件名
	filename := d.extractFilename(url)

	// 尝试获取文件大小
	size, err := d.getFileSize(ctx, url)
	if err != nil {
		d.logger.Warnf("获取文件大小失败: %v", err)
		size = 0
	}

	return &types.DownloadInfo{
		URL:      url,
		Filename: filename,
		Size:     size,
		Headers:  d.metadata.DownloadConfig.Headers,
	}, nil
}

// GetDownloadURL 获取下载链接
func (d *DirectStrategy) GetDownloadURL(ctx context.Context, version string) (string, error) {
	return d.buildDownloadURL(version)
}

// Download 执行下载
func (d *DirectStrategy) Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	// 设置自定义请求头
	if options == nil {
		options = &DownloadOptions{}
	}
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	// 合并元数据中的请求头
	if d.metadata.DownloadConfig.Headers != nil {
		for key, value := range d.metadata.DownloadConfig.Headers {
			options.Headers[key] = value
		}
	}

	return d.downloader.Download(ctx, url, targetPath, options)
}

// DownloadWithProgress 带进度的下载
func (d *DirectStrategy) DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error {
	// 设置自定义请求头
	if options == nil {
		options = &DownloadOptions{}
	}
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	// 合并元数据中的请求头
	if d.metadata.DownloadConfig.Headers != nil {
		for key, value := range d.metadata.DownloadConfig.Headers {
			options.Headers[key] = value
		}
	}

	return d.downloader.DownloadWithProgress(ctx, url, targetPath, options, progress)
}

// ExtractArchive 解压下载的压缩包
func (d *DirectStrategy) ExtractArchive(archivePath, targetPath string) error {
	_, err := d.extractor.ProcessPackage(archivePath, targetPath, d.metadata.Name, d.metadata)
	return err
}

// GetLatestVersion 获取最新版本
func (d *DirectStrategy) GetLatestVersion(ctx context.Context) (string, error) {
	// 直接URL策略无法自动获取最新版本，需要用户手动指定
	return "", fmt.Errorf("直接URL策略不支持自动获取最新版本")
}

// ListVersions 列出所有可用版本
func (d *DirectStrategy) ListVersions(ctx context.Context) ([]*types.VersionInfo, error) {
	// 直接URL策略无法列出所有版本
	return nil, fmt.Errorf("直接URL策略不支持列出所有版本")
}

// ValidateVersion 验证版本是否存在
func (d *DirectStrategy) ValidateVersion(ctx context.Context, version string) error {
	// 构建URL并检查是否可访问
	url, err := d.buildDownloadURL(version)
	if err != nil {
		return fmt.Errorf("构建下载URL失败: %w", err)
	}

	// 发送HEAD请求检查文件是否存在
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("创建HEAD请求失败: %w", err)
	}

	// 设置自定义请求头
	if d.metadata.DownloadConfig.Headers != nil {
		for key, value := range d.metadata.DownloadConfig.Headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("HEAD请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("版本不存在或无法访问: %s (状态码: %d)", version, resp.StatusCode)
	}

	return nil
}

// GetChecksum 获取文件校验和
func (d *DirectStrategy) GetChecksum(ctx context.Context, version string) (string, error) {
	// 直接URL策略通常不提供校验和
	return "", nil
}

// SupportsResume 是否支持断点续传
func (d *DirectStrategy) SupportsResume() bool {
	return true // 大多数HTTP服务器支持Range请求
}

// GetToolMetadata 获取工具元数据
func (d *DirectStrategy) GetToolMetadata() *types.ToolMetadata {
	return d.metadata
}

// 私有方法

// buildDownloadURL 构建下载URL
func (d *DirectStrategy) buildDownloadURL(version string) (string, error) {
	template := d.metadata.DownloadConfig.URLTemplate
	if template == "" {
		return "", fmt.Errorf("未配置URL模板")
	}

	platform := types.GetCurrentPlatform()

	// 替换模板变量
	url := template
	url = strings.ReplaceAll(url, "{version}", version)
	url = strings.ReplaceAll(url, "{os}", d.mapOSName(platform.OS))
	url = strings.ReplaceAll(url, "{arch}", d.mapArchName(platform.Arch))

	// 处理版本别名
	if d.metadata.VersionConfig.Aliases != nil {
		if alias, exists := d.metadata.VersionConfig.Aliases[version]; exists {
			url = strings.ReplaceAll(url, version, alias)
		}
	}

	return url, nil
}

// extractFilename 从URL中提取文件名
func (d *DirectStrategy) extractFilename(url string) string {
	// 从URL中提取文件名
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]

	// 移除查询参数
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	// 如果没有文件扩展名，根据配置添加
	if filepath.Ext(filename) == "" {
		if d.metadata.DownloadConfig.ExtractBinary != "" {
			filename = d.metadata.DownloadConfig.ExtractBinary
		} else {
			filename = d.metadata.Name
		}

		// 在Windows上添加.exe扩展名
		if runtime.GOOS == "windows" && !strings.HasSuffix(filename, ".exe") {
			filename += ".exe"
		}
	}

	return filename
}

// getFileSize 获取文件大小
func (d *DirectStrategy) getFileSize(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	// 设置自定义请求头
	if d.metadata.DownloadConfig.Headers != nil {
		for key, value := range d.metadata.DownloadConfig.Headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HEAD请求失败，状态码: %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// mapOSName 映射操作系统名称
func (d *DirectStrategy) mapOSName(os string) string {
	mapping := map[string]string{
		"darwin":  "darwin",
		"linux":   "linux",
		"windows": "windows",
	}

	if mapped, exists := mapping[os]; exists {
		return mapped
	}

	return os
}

// mapArchName 映射架构名称
func (d *DirectStrategy) mapArchName(arch string) string {
	mapping := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	if mapped, exists := mapping[arch]; exists {
		return mapped
	}

	return arch
}

// ArchiveStrategy 归档文件下载策略
type ArchiveStrategy struct {
	metadata   *types.ToolMetadata
	fs         afero.Fs
	logger     *logrus.Logger
	downloader Downloader
	extractor  *PackageProcessor
	client     *http.Client
}

// NewArchiveStrategy 创建归档文件下载策略
func NewArchiveStrategy(metadata *types.ToolMetadata, fs afero.Fs, logger *logrus.Logger) Strategy {
	return &ArchiveStrategy{
		metadata:   metadata,
		fs:         fs,
		logger:     logger,
		downloader: NewHTTPDownloader(fs, logger),
		extractor:  NewPackageProcessor(fs, logger),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDownloadInfo 获取下载信息
func (a *ArchiveStrategy) GetDownloadInfo(ctx context.Context, version string) (*types.DownloadInfo, error) {
	a.logger.Debugf("获取归档文件下载信息: %s@%s", a.metadata.Name, version)

	url, err := a.buildDownloadURL(version)
	if err != nil {
		return nil, fmt.Errorf("构建下载URL失败: %w", err)
	}

	filename := a.extractFilename(url)
	size, err := a.getFileSize(ctx, url)
	if err != nil {
		a.logger.Warnf("获取文件大小失败: %v", err)
		size = 0
	}

	return &types.DownloadInfo{
		URL:      url,
		Filename: filename,
		Size:     size,
		Headers:  a.metadata.DownloadConfig.Headers,
	}, nil
}

// GetDownloadURL 获取下载链接
func (a *ArchiveStrategy) GetDownloadURL(ctx context.Context, version string) (string, error) {
	return a.buildDownloadURL(version)
}

// Download 执行下载
func (a *ArchiveStrategy) Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	if options == nil {
		options = &DownloadOptions{}
	}
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	if a.metadata.DownloadConfig.Headers != nil {
		for key, value := range a.metadata.DownloadConfig.Headers {
			options.Headers[key] = value
		}
	}

	return a.downloader.Download(ctx, url, targetPath, options)
}

// DownloadWithProgress 带进度的下载
func (a *ArchiveStrategy) DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error {
	if options == nil {
		options = &DownloadOptions{}
	}
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	if a.metadata.DownloadConfig.Headers != nil {
		for key, value := range a.metadata.DownloadConfig.Headers {
			options.Headers[key] = value
		}
	}

	return a.downloader.DownloadWithProgress(ctx, url, targetPath, options, progress)
}

// ExtractArchive 解压下载的压缩包
func (a *ArchiveStrategy) ExtractArchive(archivePath, targetPath string) error {
	_, err := a.extractor.ProcessPackage(archivePath, targetPath, a.metadata.Name, a.metadata)
	return err
}

// GetLatestVersion 获取最新版本
func (a *ArchiveStrategy) GetLatestVersion(ctx context.Context) (string, error) {
	return "", fmt.Errorf("归档文件策略不支持自动获取最新版本")
}

// ListVersions 列出所有可用版本
func (a *ArchiveStrategy) ListVersions(ctx context.Context) ([]*types.VersionInfo, error) {
	return nil, fmt.Errorf("归档文件策略不支持列出所有版本")
}

// ValidateVersion 验证版本是否存在
func (a *ArchiveStrategy) ValidateVersion(ctx context.Context, version string) error {
	url, err := a.buildDownloadURL(version)
	if err != nil {
		return fmt.Errorf("构建下载URL失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("创建HEAD请求失败: %w", err)
	}

	if a.metadata.DownloadConfig.Headers != nil {
		for key, value := range a.metadata.DownloadConfig.Headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("HEAD请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("版本不存在或无法访问: %s (状态码: %d)", version, resp.StatusCode)
	}

	return nil
}

// GetChecksum 获取文件校验和
func (a *ArchiveStrategy) GetChecksum(ctx context.Context, version string) (string, error) {
	return "", nil
}

// SupportsResume 是否支持断点续传
func (a *ArchiveStrategy) SupportsResume() bool {
	return true
}

// GetToolMetadata 获取工具元数据
func (a *ArchiveStrategy) GetToolMetadata() *types.ToolMetadata {
	return a.metadata
}

// buildDownloadURL 构建下载URL
func (a *ArchiveStrategy) buildDownloadURL(version string) (string, error) {
	template := a.metadata.DownloadConfig.URLTemplate
	if template == "" {
		return "", fmt.Errorf("未配置URL模板")
	}

	platform := types.GetCurrentPlatform()

	url := template
	url = strings.ReplaceAll(url, "{version}", version)
	url = strings.ReplaceAll(url, "{os}", a.mapOSName(platform.OS))
	url = strings.ReplaceAll(url, "{arch}", a.mapArchName(platform.Arch))

	if a.metadata.VersionConfig.Aliases != nil {
		if alias, exists := a.metadata.VersionConfig.Aliases[version]; exists {
			url = strings.ReplaceAll(url, version, alias)
		}
	}

	return url, nil
}

// extractFilename 从URL中提取文件名
func (a *ArchiveStrategy) extractFilename(url string) string {
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]

	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	return filename
}

// getFileSize 获取文件大小
func (a *ArchiveStrategy) getFileSize(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, err
	}

	if a.metadata.DownloadConfig.Headers != nil {
		for key, value := range a.metadata.DownloadConfig.Headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HEAD请求失败，状态码: %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// mapOSName 映射操作系统名称
func (a *ArchiveStrategy) mapOSName(os string) string {
	mapping := map[string]string{
		"darwin":  "darwin",
		"linux":   "linux",
		"windows": "windows",
	}

	if mapped, exists := mapping[os]; exists {
		return mapped
	}

	return os
}

// mapArchName 映射架构名称
func (a *ArchiveStrategy) mapArchName(arch string) string {
	mapping := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	if mapped, exists := mapping[arch]; exists {
		return mapped
	}

	return arch
}
