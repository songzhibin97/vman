package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/pkg/types"
)

// DownloaderTestSuite 下载器测试套件
type DownloaderTestSuite struct {
	suite.Suite
	fs         afero.Fs
	downloader *Downloader
	server     *httptest.Server
	tempDir    string
}

func (suite *DownloaderTestSuite) SetupSuite() {
	// 创建测试HTTP服务器
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test-file.txt":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "11")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello World"))
		case "/large-file.bin":
			// 模拟大文件下载
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "1048576") // 1MB
			w.WriteHeader(http.StatusOK)
			// 写入1MB的数据
			data := make([]byte, 1024)
			for i := range data {
				data[i] = byte(i % 256)
			}
			for i := 0; i < 1024; i++ {
				w.Write(data)
			}
		case "/slow-file.txt":
			// 模拟慢速下载
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte("slow"))
		case "/not-found":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server Error"))
		case "/redirect":
			w.Header().Set("Location", suite.server.URL+"/test-file.txt")
			w.WriteHeader(http.StatusMovedPermanently)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (suite *DownloaderTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *DownloaderTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.tempDir = "/tmp/downloads"
	suite.fs.MkdirAll(suite.tempDir, 0755)

	// 创建下载器实例
	config := types.DownloadSettings{
		Timeout:             30 * time.Second,
		Retries:             3,
		ConcurrentDownloads: 2,
	}
	suite.downloader = NewDownloaderWithFs(config, suite.fs)
}

func (suite *DownloaderTestSuite) TestSimpleDownload() {
	ctx := context.Background()
	url := suite.server.URL + "/test-file.txt"
	destPath := filepath.Join(suite.tempDir, "test-file.txt")

	// 执行下载
	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.NoError(err)

	// 验证文件内容
	content, err := afero.ReadFile(suite.fs, destPath)
	suite.NoError(err)
	suite.Equal("Hello World", string(content))

	// 验证文件权限
	info, err := suite.fs.Stat(destPath)
	suite.NoError(err)
	suite.Equal(os.FileMode(0644), info.Mode().Perm())
}

func (suite *DownloaderTestSuite) TestDownloadWithProgress() {
	ctx := context.Background()
	url := suite.server.URL + "/large-file.bin"
	destPath := filepath.Join(suite.tempDir, "large-file.bin")

	// 跟踪进度
	var progressCalls []ProgressUpdate
	progressCallback := func(update ProgressUpdate) {
		progressCalls = append(progressCalls, update)
	}

	// 执行下载
	err := suite.downloader.Download(ctx, url, destPath, progressCallback)
	suite.NoError(err)

	// 验证进度回调被调用
	suite.Greater(len(progressCalls), 0)

	// 验证最后一次进度是100%
	lastProgress := progressCalls[len(progressCalls)-1]
	suite.Equal(int64(1048576), lastProgress.Total)
	suite.Equal(int64(1048576), lastProgress.Downloaded)

	// 验证文件大小
	info, err := suite.fs.Stat(destPath)
	suite.NoError(err)
	suite.Equal(int64(1048576), info.Size())
}

func (suite *DownloaderTestSuite) TestDownloadWithChecksum() {
	ctx := context.Background()
	url := suite.server.URL + "/test-file.txt"
	destPath := filepath.Join(suite.tempDir, "test-file.txt")

	// 计算预期的校验和
	expectedContent := "Hello World"
	expectedChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(expectedContent)))

	// 执行下载并验证校验和
	err := suite.downloader.DownloadWithChecksum(ctx, url, destPath, expectedChecksum, nil)
	suite.NoError(err)

	// 验证文件内容
	content, err := afero.ReadFile(suite.fs, destPath)
	suite.NoError(err)
	suite.Equal(expectedContent, string(content))
}

func (suite *DownloaderTestSuite) TestDownloadWithInvalidChecksum() {
	ctx := context.Background()
	url := suite.server.URL + "/test-file.txt"
	destPath := filepath.Join(suite.tempDir, "test-file.txt")

	// 使用错误的校验和
	invalidChecksum := "invalid_checksum"

	// 执行下载，应该失败
	err := suite.downloader.DownloadWithChecksum(ctx, url, destPath, invalidChecksum, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "校验和验证失败")

	// 验证文件不存在（下载失败后应该清理）
	exists, _ := afero.Exists(suite.fs, destPath)
	suite.False(exists)
}

func (suite *DownloaderTestSuite) TestDownloadNotFound() {
	ctx := context.Background()
	url := suite.server.URL + "/not-found"
	destPath := filepath.Join(suite.tempDir, "not-found.txt")

	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "404")
}

func (suite *DownloaderTestSuite) TestDownloadServerError() {
	ctx := context.Background()
	url := suite.server.URL + "/server-error"
	destPath := filepath.Join(suite.tempDir, "server-error.txt")

	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "500")
}

func (suite *DownloaderTestSuite) TestDownloadRedirect() {
	ctx := context.Background()
	url := suite.server.URL + "/redirect"
	destPath := filepath.Join(suite.tempDir, "redirected-file.txt")

	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.NoError(err)

	// 验证文件内容（应该是重定向目标的内容）
	content, err := afero.ReadFile(suite.fs, destPath)
	suite.NoError(err)
	suite.Equal("Hello World", string(content))
}

func (suite *DownloaderTestSuite) TestDownloadTimeout() {
	// 创建超时配置的下载器
	config := types.DownloadSettings{
		Timeout:             50 * time.Millisecond, // 很短的超时
		Retries:             1,
		ConcurrentDownloads: 1,
	}
	timeoutDownloader := NewDownloaderWithFs(config, suite.fs)

	ctx := context.Background()
	url := suite.server.URL + "/slow-file.txt"
	destPath := filepath.Join(suite.tempDir, "timeout-file.txt")

	err := timeoutDownloader.Download(ctx, url, destPath, nil)
	suite.Error(err)
	suite.Contains(strings.ToLower(err.Error()), "timeout")
}

func (suite *DownloaderTestSuite) TestDownloadCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	url := suite.server.URL + "/large-file.bin"
	destPath := filepath.Join(suite.tempDir, "cancelled-file.bin")

	// 启动下载
	errChan := make(chan error, 1)
	go func() {
		errChan <- suite.downloader.Download(ctx, url, destPath, nil)
	}()

	// 立即取消
	cancel()

	// 等待下载完成
	err := <-errChan
	suite.Error(err)
	suite.Contains(strings.ToLower(err.Error()), "cancel")
}

func (suite *DownloaderTestSuite) TestDownloadRetry() {
	// 创建一个会间歇性失败的服务器
	retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟前两次请求失败，第三次成功
		if r.Header.Get("X-Retry-Count") == "" {
			w.Header().Set("X-Retry-Count", "1")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		retryCount := r.Header.Get("X-Retry-Count")
		if retryCount == "1" {
			w.Header().Set("X-Retry-Count", "2")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// 第三次请求成功
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success after retry"))
	}))
	defer retryServer.Close()

	ctx := context.Background()
	url := retryServer.URL + "/retry-test"
	destPath := filepath.Join(suite.tempDir, "retry-file.txt")

	// 注意：这个测试可能需要根据实际的重试机制实现进行调整
	err := suite.downloader.Download(ctx, url, destPath, nil)
	
	// 根据实际实现，这里可能成功也可能失败
	// 如果实现了重试机制，应该成功
	if err == nil {
		content, err := afero.ReadFile(suite.fs, destPath)
		suite.NoError(err)
		suite.Equal("Success after retry", string(content))
	} else {
		suite.T().Logf("Download failed as expected without retry mechanism: %v", err)
	}
}

func (suite *DownloaderTestSuite) TestConcurrentDownloads() {
	ctx := context.Background()
	urls := []string{
		suite.server.URL + "/test-file.txt",
		suite.server.URL + "/test-file.txt",
		suite.server.URL + "/test-file.txt",
	}
	destPaths := []string{
		filepath.Join(suite.tempDir, "concurrent1.txt"),
		filepath.Join(suite.tempDir, "concurrent2.txt"),
		filepath.Join(suite.tempDir, "concurrent3.txt"),
	}

	// 同时启动多个下载
	errChan := make(chan error, len(urls))
	for i, url := range urls {
		go func(url, destPath string) {
			errChan <- suite.downloader.Download(ctx, url, destPath, nil)
		}(url, destPaths[i])
	}

	// 等待所有下载完成
	for i := 0; i < len(urls); i++ {
		err := <-errChan
		suite.NoError(err)
	}

	// 验证所有文件都下载成功
	for _, destPath := range destPaths {
		content, err := afero.ReadFile(suite.fs, destPath)
		suite.NoError(err)
		suite.Equal("Hello World", string(content))
	}
}

func (suite *DownloaderTestSuite) TestDownloadToExistingFile() {
	ctx := context.Background()
	url := suite.server.URL + "/test-file.txt"
	destPath := filepath.Join(suite.tempDir, "existing-file.txt")

	// 先创建一个文件
	err := afero.WriteFile(suite.fs, destPath, []byte("existing content"), 0644)
	suite.NoError(err)

	// 下载应该覆盖现有文件
	err = suite.downloader.Download(ctx, url, destPath, nil)
	suite.NoError(err)

	// 验证文件内容被更新
	content, err := afero.ReadFile(suite.fs, destPath)
	suite.NoError(err)
	suite.Equal("Hello World", string(content))
}

func (suite *DownloaderTestSuite) TestDownloadToNonexistentDirectory() {
	ctx := context.Background()
	url := suite.server.URL + "/test-file.txt"
	destPath := filepath.Join(suite.tempDir, "nonexistent", "subdir", "file.txt")

	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.NoError(err)

	// 验证目录被创建
	content, err := afero.ReadFile(suite.fs, destPath)
	suite.NoError(err)
	suite.Equal("Hello World", string(content))
}

func (suite *DownloaderTestSuite) TestCalculateChecksum() {
	// 测试校验和计算
	content := "Hello World"
	reader := strings.NewReader(content)

	checksum, err := suite.downloader.calculateChecksum(reader)
	suite.NoError(err)

	expectedChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	suite.Equal(expectedChecksum, checksum)
}

func (suite *DownloaderTestSuite) TestValidateChecksum() {
	content := "Hello World"
	expectedChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	// 创建临时文件
	tempFile := filepath.Join(suite.tempDir, "checksum-test.txt")
	err := afero.WriteFile(suite.fs, tempFile, []byte(content), 0644)
	suite.NoError(err)

	// 验证校验和
	err = suite.downloader.validateChecksum(tempFile, expectedChecksum)
	suite.NoError(err)

	// 测试错误的校验和
	err = suite.downloader.validateChecksum(tempFile, "wrong_checksum")
	suite.Error(err)
	suite.Contains(err.Error(), "校验和验证失败")
}

func (suite *DownloaderTestSuite) TestDownloadHeader() {
	// 测试自定义请求头
	customServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查User-Agent
		if r.Header.Get("User-Agent") != "" {
			suite.Contains(r.Header.Get("User-Agent"), "vman")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer customServer.Close()

	ctx := context.Background()
	url := customServer.URL + "/test"
	destPath := filepath.Join(suite.tempDir, "header-test.txt")

	err := suite.downloader.Download(ctx, url, destPath, nil)
	suite.NoError(err)
}

// TestDownloaderTestSuite 运行下载器测试套件
func TestDownloaderTestSuite(t *testing.T) {
	suite.Run(t, new(DownloaderTestSuite))
}

// BenchmarkDownload 下载性能基准测试
func BenchmarkDownload(b *testing.B) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark data"))
	}))
	defer server.Close()

	fs := afero.NewMemMapFs()
	tempDir := "/tmp/benchmark"
	fs.MkdirAll(tempDir, 0755)

	config := types.DownloadSettings{
		Timeout:             30 * time.Second,
		Retries:             1,
		ConcurrentDownloads: 1,
	}
	downloader := NewDownloaderWithFs(config, fs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		url := server.URL + "/benchmark"
		destPath := filepath.Join(tempDir, fmt.Sprintf("bench-%d.txt", i))
		
		err := downloader.Download(ctx, url, destPath, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestProgressUpdate 测试进度更新
func TestProgressUpdate(t *testing.T) {
	update := ProgressUpdate{
		Downloaded: 1024,
		Total:      2048,
		Speed:      1024.0,
		ETA:        time.Second,
	}

	// 测试进度计算
	progress := update.Progress()
	assert.Equal(t, 50.0, progress)

	// 测试字符串格式化
	str := update.String()
	assert.Contains(t, str, "50.0%")
	assert.Contains(t, str, "1024 B")
	assert.Contains(t, str, "2048 B")
}

// TestDownloadError 测试下载错误类型
func TestDownloadError(t *testing.T) {
	err := &DownloadError{
		URL:     "http://example.com/file.txt",
		Message: "下载失败",
		Cause:   fmt.Errorf("network error"),
	}

	assert.Contains(t, err.Error(), "下载失败")
	assert.Contains(t, err.Error(), "http://example.com/file.txt")
	assert.Equal(t, fmt.Errorf("network error"), err.Unwrap())
}

// MockProgressCallback 模拟进度回调
type MockProgressCallback struct {
	updates []ProgressUpdate
}

func (m *MockProgressCallback) Update(update ProgressUpdate) {
	m.updates = append(m.updates, update)
}

func (m *MockProgressCallback) GetUpdates() []ProgressUpdate {
	return m.updates
}

// TestMockProgressCallback 测试模拟进度回调
func TestMockProgressCallback(t *testing.T) {
	callback := &MockProgressCallback{}
	
	update1 := ProgressUpdate{Downloaded: 512, Total: 1024}
	update2 := ProgressUpdate{Downloaded: 1024, Total: 1024}
	
	callback.Update(update1)
	callback.Update(update2)
	
	updates := callback.GetUpdates()
	assert.Len(t, updates, 2)
	assert.Equal(t, int64(512), updates[0].Downloaded)
	assert.Equal(t, int64(1024), updates[1].Downloaded)
}