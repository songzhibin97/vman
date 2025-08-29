package download

import (
	"context"
	"io"

	"github.com/songzhibin97/vman/pkg/types"
)

// Manager 下载管理器接口
type Manager interface {
	// Download 下载并安装工具版本
	Download(ctx context.Context, tool, version string, options *DownloadOptions) error

	// DownloadWithProgress 带进度显示的下载
	DownloadWithProgress(ctx context.Context, tool, version string, options *DownloadOptions, progress ProgressCallback) error

	// GetDownloadStrategy 获取下载策略
	GetDownloadStrategy(tool string) (Strategy, error)

	// AddSource 添加下载源
	AddSource(tool string, metadata *types.ToolMetadata) error

	// RemoveSource 移除下载源
	RemoveSource(tool string) error

	// ListSources 列出所有下载源
	ListSources() ([]string, error)

	// UpdateSources 更新下载源信息
	UpdateSources(ctx context.Context) error

	// SearchVersions 搜索可用版本
	SearchVersions(ctx context.Context, tool string) ([]*types.VersionInfo, error)

	// GetVersionInfo 获取版本详细信息
	GetVersionInfo(ctx context.Context, tool, version string) (*types.VersionInfo, error)

	// ClearCache 清理下载缓存
	ClearCache(tool string) error

	// GetCacheSize 获取缓存大小
	GetCacheSize(tool string) (int64, error)

	// ResumeDownload 恢复下载
	ResumeDownload(ctx context.Context, tool, version string, options *DownloadOptions) error
}

// Strategy 下载策略接口
type Strategy interface {
	// GetDownloadInfo 获取下载信息
	GetDownloadInfo(ctx context.Context, version string) (*types.DownloadInfo, error)

	// GetDownloadURL 获取下载链接
	GetDownloadURL(ctx context.Context, version string) (string, error)

	// Download 执行下载
	Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error

	// DownloadWithProgress 带进度的下载
	DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error

	// ExtractArchive 解压下载的压缩包
	ExtractArchive(archivePath, targetPath string) error

	// GetLatestVersion 获取最新版本
	GetLatestVersion(ctx context.Context) (string, error)

	// ListVersions 列出所有可用版本
	ListVersions(ctx context.Context) ([]*types.VersionInfo, error)

	// ValidateVersion 验证版本是否存在
	ValidateVersion(ctx context.Context, version string) error

	// GetChecksum 获取文件校验和
	GetChecksum(ctx context.Context, version string) (string, error)

	// SupportsResume 是否支持断点续传
	SupportsResume() bool

	// GetToolMetadata 获取工具元数据
	GetToolMetadata() *types.ToolMetadata
}

// DownloadOptions 下载选项
type DownloadOptions struct {
	// Force 强制重新下载
	Force bool

	// SkipChecksum 跳过校验和验证
	SkipChecksum bool

	// Timeout 下载超时时间（秒）
	Timeout int

	// Retries 重试次数
	Retries int

	// Resume 是否支持断点续传
	Resume bool

	// TempDir 临时目录
	TempDir string

	// KeepDownload 保留下载文件
	KeepDownload bool

	// Headers 自定义请求头
	Headers map[string]string
}

// ProgressInfo 下载进度信息
type ProgressInfo struct {
	// Total 总字节数
	Total int64

	// Downloaded 已下载字节数
	Downloaded int64

	// Percentage 下载百分比
	Percentage float64

	// Speed 下载速度 (字节/秒)
	Speed int64

	// ETA 预计剩余时间（秒）
	ETA int64

	// Status 状态信息
	Status string
}

// ProgressCallback 进度回调函数
type ProgressCallback func(*ProgressInfo)

// DownloadError 下载错误
type DownloadError struct {
	Tool    string
	Version string
	URL     string
	Cause   error
	Code    DownloadErrorCode
}

func (e *DownloadError) Error() string {
	return e.Cause.Error()
}

func (e *DownloadError) Unwrap() error {
	return e.Cause
}

// DownloadErrorCode 下载错误代码
type DownloadErrorCode int

const (
	// NetworkError 网络错误
	NetworkError DownloadErrorCode = iota
	// ChecksumMismatch 校验和不匹配
	ChecksumMismatch
	// VersionNotFound 版本不存在
	VersionNotFound
	// ExtractionError 解压错误
	ExtractionError
	// PermissionError 权限错误
	PermissionError
	// DiskSpaceError 磁盘空间不足
	DiskSpaceError
	// CorruptedFile 文件损坏
	CorruptedFile
)

// DownloadReader 可追踪下载进度的Reader
type DownloadReader struct {
	reader   io.Reader
	total    int64
	read     int64
	callback ProgressCallback
}

// NewDownloadReader 创建新的下载读取器
func NewDownloadReader(reader io.Reader, total int64, callback ProgressCallback) *DownloadReader {
	return &DownloadReader{
		reader:   reader,
		total:    total,
		callback: callback,
	}
}

// Read 实现io.Reader接口
func (dr *DownloadReader) Read(p []byte) (int, error) {
	n, err := dr.reader.Read(p)
	if n > 0 {
		dr.read += int64(n)
		if dr.callback != nil {
			percentage := float64(dr.read) / float64(dr.total) * 100
			dr.callback(&ProgressInfo{
				Total:      dr.total,
				Downloaded: dr.read,
				Percentage: percentage,
			})
		}
	}
	return n, err
}
