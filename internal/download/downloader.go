package download

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/utils"
	"github.com/spf13/afero"
)

// Downloader 核心下载器接口
type Downloader interface {
	// Download 下载文件
	Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error

	// DownloadWithProgress 带进度的下载
	DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error

	// Resume 恢复下载
	Resume(ctx context.Context, url, targetPath string, options *DownloadOptions) error

	// ValidateChecksum 验证校验和
	ValidateChecksum(filePath, expectedChecksum string) error

	// GetDownloadSize 获取文件大小
	GetDownloadSize(ctx context.Context, url string, headers map[string]string) (int64, error)

	// SupportsResume 检查是否支持断点续传
	SupportsResume(ctx context.Context, url string, headers map[string]string) (bool, error)
}

// HTTPDownloader HTTP下载器实现
type HTTPDownloader struct {
	fs     afero.Fs
	logger *logrus.Logger
	client *http.Client
}

// NewHTTPDownloader 创建HTTP下载器
func NewHTTPDownloader(fs afero.Fs, logger *logrus.Logger) Downloader {
	return &HTTPDownloader{
		fs:     fs,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// Download 下载文件
func (d *HTTPDownloader) Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	d.logger.Debugf("开始下载文件: %s -> %s", url, targetPath)

	// 创建目标目录
	if err := d.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 检查是否支持断点续传
	var startOffset int64 = 0
	if options != nil && options.Resume {
		if info, err := d.fs.Stat(targetPath); err == nil {
			startOffset = info.Size()
			d.logger.Debugf("文件已存在，从 %d 字节处恢复下载", startOffset)
		}
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置自定义请求头
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	// 设置Range头支持断点续传
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	// 执行请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 打开目标文件
	var file afero.File
	if startOffset > 0 {
		file, err = d.fs.OpenFile(targetPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = d.fs.Create(targetPath)
	}
	if err != nil {
		return fmt.Errorf("打开目标文件失败: %w", err)
	}
	defer file.Close()

	// 复制数据
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("下载数据失败: %w", err)
	}

	d.logger.Debugf("文件下载完成: %s", targetPath)
	return nil
}

// DownloadWithProgress 带进度的下载
func (d *HTTPDownloader) DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error {
	d.logger.Debugf("开始带进度下载文件: %s -> %s", url, targetPath)

	// 创建目标目录
	if err := d.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 获取文件大小
	totalSize, err := d.GetDownloadSize(ctx, url, options.Headers)
	if err != nil {
		d.logger.Warnf("获取文件大小失败: %v", err)
		totalSize = 0
	}

	// 检查断点续传
	var startOffset int64 = 0
	if options != nil && options.Resume {
		if info, err := d.fs.Stat(targetPath); err == nil {
			startOffset = info.Size()
			d.logger.Debugf("从 %d 字节处恢复下载", startOffset)
		}
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}

	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	// 执行请求
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 打开目标文件
	var file afero.File
	if startOffset > 0 {
		file, err = d.fs.OpenFile(targetPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		file, err = d.fs.Create(targetPath)
	}
	if err != nil {
		return fmt.Errorf("打开目标文件失败: %w", err)
	}
	defer file.Close()

	// 创建进度跟踪读取器
	reader := NewProgressReader(resp.Body, totalSize, startOffset, progress)

	// 复制数据
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("下载数据失败: %w", err)
	}

	// 发送完成进度
	if progress != nil {
		progress(&ProgressInfo{
			Total:      totalSize,
			Downloaded: totalSize,
			Percentage: 100.0,
			Status:     "完成",
		})
	}

	d.logger.Debugf("带进度下载完成: %s", targetPath)
	return nil
}

// Resume 恢复下载
func (d *HTTPDownloader) Resume(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	if options == nil {
		options = &DownloadOptions{}
	}
	options.Resume = true
	return d.Download(ctx, url, targetPath, options)
}

// ValidateChecksum 验证校验和
func (d *HTTPDownloader) ValidateChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // 没有期望的校验和，跳过验证
	}

	d.logger.Debugf("验证文件校验和: %s", filePath)

	// 计算文件的SHA256
	actualChecksum, err := utils.CalculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("计算文件校验和失败: %w", err)
	}

	// 比较校验和
	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("校验和不匹配: 期望 %s, 实际 %s", expectedChecksum, actualChecksum)
	}

	d.logger.Debugf("校验和验证通过: %s", actualChecksum)
	return nil
}

// GetDownloadSize 获取文件大小
func (d *HTTPDownloader) GetDownloadSize(ctx context.Context, url string, headers map[string]string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建HEAD请求失败: %w", err)
	}

	// 设置请求头
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HEAD请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HEAD请求失败，状态码: %d", resp.StatusCode)
	}

	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		return 0, fmt.Errorf("未找到Content-Length头")
	}

	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析Content-Length失败: %w", err)
	}

	return size, nil
}

// SupportsResume 检查是否支持断点续传
func (d *HTTPDownloader) SupportsResume(ctx context.Context, url string, headers map[string]string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, fmt.Errorf("创建HEAD请求失败: %w", err)
	}

	// 设置请求头
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("HEAD请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查是否支持Range请求
	acceptRanges := resp.Header.Get("Accept-Ranges")
	return acceptRanges == "bytes", nil
}

// ProgressReader 进度跟踪读取器
type ProgressReader struct {
	reader     io.Reader
	total      int64
	read       int64
	startTime  time.Time
	lastUpdate time.Time
	callback   ProgressCallback
}

// NewProgressReader 创建进度跟踪读取器
func NewProgressReader(reader io.Reader, total, startOffset int64, callback ProgressCallback) *ProgressReader {
	return &ProgressReader{
		reader:     reader,
		total:      total,
		read:       startOffset,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
		callback:   callback,
	}
}

// Read 实现io.Reader接口
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.read += int64(n)
		pr.updateProgress()
	}
	return n, err
}

// updateProgress 更新进度
func (pr *ProgressReader) updateProgress() {
	now := time.Now()

	// 限制更新频率（每100ms更新一次）
	if now.Sub(pr.lastUpdate) < 100*time.Millisecond {
		return
	}
	pr.lastUpdate = now

	if pr.callback == nil {
		return
	}

	var percentage float64
	var speed int64
	var eta int64

	if pr.total > 0 {
		percentage = float64(pr.read) / float64(pr.total) * 100
	}

	// 计算下载速度
	elapsed := now.Sub(pr.startTime).Seconds()
	if elapsed > 0 {
		speed = int64(float64(pr.read) / elapsed)
	}

	// 计算剩余时间
	if speed > 0 && pr.total > 0 {
		remaining := pr.total - pr.read
		eta = remaining / speed
	}

	// 调用回调函数
	pr.callback(&ProgressInfo{
		Total:      pr.total,
		Downloaded: pr.read,
		Percentage: percentage,
		Speed:      speed,
		ETA:        eta,
		Status:     "下载中",
	})
}

// CacheManager 缓存管理器
type CacheManager struct {
	fs       afero.Fs
	cacheDir string
	logger   *logrus.Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(fs afero.Fs, cacheDir string, logger *logrus.Logger) *CacheManager {
	return &CacheManager{
		fs:       fs,
		cacheDir: cacheDir,
		logger:   logger,
	}
}

// GetCachedFile 获取缓存文件路径
func (c *CacheManager) GetCachedFile(tool, version, filename string) string {
	return filepath.Join(c.cacheDir, tool, version, filename)
}

// IsCached 检查是否已缓存
func (c *CacheManager) IsCached(tool, version, filename string) bool {
	cachedPath := c.GetCachedFile(tool, version, filename)
	exists, err := afero.Exists(c.fs, cachedPath)
	return err == nil && exists
}

// SaveToCache 保存到缓存
func (c *CacheManager) SaveToCache(tool, version, filename, sourcePath string) error {
	cachedPath := c.GetCachedFile(tool, version, filename)

	// 创建缓存目录
	if err := c.fs.MkdirAll(filepath.Dir(cachedPath), 0755); err != nil {
		return fmt.Errorf("创建缓存目录失败: %w", err)
	}

	// 复制文件到缓存
	return c.copyFile(sourcePath, cachedPath)
}

// LoadFromCache 从缓存加载
func (c *CacheManager) LoadFromCache(tool, version, filename, targetPath string) error {
	cachedPath := c.GetCachedFile(tool, version, filename)

	if !c.IsCached(tool, version, filename) {
		return fmt.Errorf("文件未缓存: %s", cachedPath)
	}

	// 从缓存复制文件
	return c.copyFile(cachedPath, targetPath)
}

// ClearCache 清理缓存
func (c *CacheManager) ClearCache(tool string) error {
	toolCacheDir := filepath.Join(c.cacheDir, tool)

	if err := c.fs.RemoveAll(toolCacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理缓存失败: %w", err)
	}

	c.logger.Infof("已清理 %s 的缓存", tool)
	return nil
}

// GetCacheSize 获取缓存大小
func (c *CacheManager) GetCacheSize(tool string) (int64, error) {
	toolCacheDir := filepath.Join(c.cacheDir, tool)
	return c.calculateDirSize(toolCacheDir)
}

// CleanExpiredCache 清理过期缓存
func (c *CacheManager) CleanExpiredCache(maxAge time.Duration) error {
	entries, err := afero.ReadDir(c.fs, c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() && now.Sub(entry.ModTime()) > maxAge {
			toolCacheDir := filepath.Join(c.cacheDir, entry.Name())
			c.logger.Infof("清理过期缓存: %s", toolCacheDir)
			if err := c.fs.RemoveAll(toolCacheDir); err != nil {
				c.logger.Warnf("清理过期缓存失败: %v", err)
			}
		}
	}

	return nil
}

// copyFile 复制文件
func (c *CacheManager) copyFile(src, dst string) error {
	srcFile, err := c.fs.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 确保目标目录存在
	if err := c.fs.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	dstFile, err := c.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// calculateDirSize 计算目录大小
func (c *CacheManager) calculateDirSize(dirPath string) (int64, error) {
	var totalSize int64

	err := afero.Walk(c.fs, dirPath, func(path string, info os.FileInfo, err error) error {
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

// ChecksumValidator 校验和验证器
type ChecksumValidator struct {
	logger *logrus.Logger
}

// NewChecksumValidator 创建校验和验证器
func NewChecksumValidator(logger *logrus.Logger) *ChecksumValidator {
	return &ChecksumValidator{
		logger: logger,
	}
}

// ValidateSHA256 验证SHA256校验和
func (v *ChecksumValidator) ValidateSHA256(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil
	}

	v.logger.Debugf("验证SHA256校验和: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("计算校验和失败: %w", err)
	}

	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	if !strings.EqualFold(actualChecksum, expectedChecksum) {
		return fmt.Errorf("SHA256校验和不匹配: 期望 %s, 实际 %s", expectedChecksum, actualChecksum)
	}

	v.logger.Debugf("SHA256校验和验证通过")
	return nil
}

// ValidateFile 验证文件完整性
func (v *ChecksumValidator) ValidateFile(filePath string, expectedSize int64, expectedChecksum string) error {
	// 检查文件大小
	if expectedSize > 0 {
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("获取文件信息失败: %w", err)
		}

		if info.Size() != expectedSize {
			return fmt.Errorf("文件大小不匹配: 期望 %d 字节, 实际 %d 字节", expectedSize, info.Size())
		}
	}

	// 验证校验和
	return v.ValidateSHA256(filePath, expectedChecksum)
}
