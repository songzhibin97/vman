package download

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ExtractorTestSuite 提取器测试套件
type ExtractorTestSuite struct {
	suite.Suite
	fs        afero.Fs
	extractor *Extractor
	tempDir   string
}

func (suite *ExtractorTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.tempDir = "/tmp/extract-test"
	suite.fs.MkdirAll(suite.tempDir, 0755)
	suite.extractor = NewExtractorWithFs(suite.fs)
}

// createTestZip 创建测试用的ZIP文件
func (suite *ExtractorTestSuite) createTestZip(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for filename, content := range files {
		f, err := w.Create(filename)
		suite.NoError(err)
		_, err = f.Write([]byte(content))
		suite.NoError(err)
	}

	err := w.Close()
	suite.NoError(err)
	return buf.Bytes()
}

// createTestTarGz 创建测试用的tar.gz文件
func (suite *ExtractorTestSuite) createTestTarGz(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	for filename, content := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		suite.NoError(err)
		_, err = tw.Write([]byte(content))
		suite.NoError(err)
	}

	err := tw.Close()
	suite.NoError(err)
	err = gw.Close()
	suite.NoError(err)
	return buf.Bytes()
}

// createTestTar 创建测试用的tar文件
func (suite *ExtractorTestSuite) createTestTar(files map[string]string) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for filename, content := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0644,
			Size: int64(len(content)),
		}
		err := tw.WriteHeader(hdr)
		suite.NoError(err)
		_, err = tw.Write([]byte(content))
		suite.NoError(err)
	}

	err := tw.Close()
	suite.NoError(err)
	return buf.Bytes()
}

func (suite *ExtractorTestSuite) TestExtractZip() {
	// 创建测试ZIP文件
	files := map[string]string{
		"file1.txt":       "content1",
		"dir/file2.txt":   "content2",
		"dir/file3.txt":   "content3",
		"binary/kubectl":  "fake kubectl binary",
	}
	zipData := suite.createTestZip(files)

	// 写入ZIP文件
	zipPath := filepath.Join(suite.tempDir, "test.zip")
	err := afero.WriteFile(suite.fs, zipPath, zipData, 0644)
	suite.NoError(err)

	// 提取目录
	extractDir := filepath.Join(suite.tempDir, "extracted")

	// 执行提取
	err = suite.extractor.Extract(zipPath, extractDir)
	suite.NoError(err)

	// 验证提取的文件
	for filename, expectedContent := range files {
		extractedPath := filepath.Join(extractDir, filename)
		exists, err := afero.Exists(suite.fs, extractedPath)
		suite.NoError(err)
		suite.True(exists, "File should exist: %s", filename)

		content, err := afero.ReadFile(suite.fs, extractedPath)
		suite.NoError(err)
		suite.Equal(expectedContent, string(content), "Content mismatch for: %s", filename)
	}
}

func (suite *ExtractorTestSuite) TestExtractTarGz() {
	// 创建测试tar.gz文件
	files := map[string]string{
		"file1.txt":      "content1",
		"dir/file2.txt":  "content2",
		"bin/terraform":  "fake terraform binary",
	}
	tarGzData := suite.createTestTarGz(files)

	// 写入tar.gz文件
	tarGzPath := filepath.Join(suite.tempDir, "test.tar.gz")
	err := afero.WriteFile(suite.fs, tarGzPath, tarGzData, 0644)
	suite.NoError(err)

	// 提取目录
	extractDir := filepath.Join(suite.tempDir, "extracted-tar-gz")

	// 执行提取
	err = suite.extractor.Extract(tarGzPath, extractDir)
	suite.NoError(err)

	// 验证提取的文件
	for filename, expectedContent := range files {
		extractedPath := filepath.Join(extractDir, filename)
		content, err := afero.ReadFile(suite.fs, extractedPath)
		suite.NoError(err)
		suite.Equal(expectedContent, string(content))
	}
}

func (suite *ExtractorTestSuite) TestExtractTar() {
	// 创建测试tar文件
	files := map[string]string{
		"file1.txt":     "content1",
		"dir/file2.txt": "content2",
	}
	tarData := suite.createTestTar(files)

	// 写入tar文件
	tarPath := filepath.Join(suite.tempDir, "test.tar")
	err := afero.WriteFile(suite.fs, tarPath, tarData, 0644)
	suite.NoError(err)

	// 提取目录
	extractDir := filepath.Join(suite.tempDir, "extracted-tar")

	// 执行提取
	err = suite.extractor.Extract(tarPath, extractDir)
	suite.NoError(err)

	// 验证提取的文件
	for filename, expectedContent := range files {
		extractedPath := filepath.Join(extractDir, filename)
		content, err := afero.ReadFile(suite.fs, extractedPath)
		suite.NoError(err)
		suite.Equal(expectedContent, string(content))
	}
}

func (suite *ExtractorTestSuite) TestFindBinaryInZip() {
	// 创建包含二进制文件的ZIP
	files := map[string]string{
		"README.md":      "readme content",
		"bin/kubectl":    "kubectl binary",
		"bin/helm":       "helm binary",
		"docs/usage.txt": "usage docs",
	}
	zipData := suite.createTestZip(files)

	zipPath := filepath.Join(suite.tempDir, "binary-test.zip")
	err := afero.WriteFile(suite.fs, zipPath, zipData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "binary-extracted")

	// 提取并查找kubectl二进制文件
	binaryPath, err := suite.extractor.ExtractAndFindBinary(zipPath, extractDir, "kubectl")
	suite.NoError(err)
	suite.Equal(filepath.Join(extractDir, "bin/kubectl"), binaryPath)

	// 验证二进制文件内容
	content, err := afero.ReadFile(suite.fs, binaryPath)
	suite.NoError(err)
	suite.Equal("kubectl binary", string(content))
}

func (suite *ExtractorTestSuite) TestFindBinaryInTarGz() {
	// 创建包含二进制文件的tar.gz
	files := map[string]string{
		"terraform":           "terraform binary at root",
		"bin/terraform-alt":   "alternative terraform binary",
		"docs/README.md":      "documentation",
	}
	tarGzData := suite.createTestTarGz(files)

	tarGzPath := filepath.Join(suite.tempDir, "terraform.tar.gz")
	err := afero.WriteFile(suite.fs, tarGzPath, tarGzData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "terraform-extracted")

	// 提取并查找terraform二进制文件
	binaryPath, err := suite.extractor.ExtractAndFindBinary(tarGzPath, extractDir, "terraform")
	suite.NoError(err)
	
	// 应该找到根目录下的terraform文件
	suite.Equal(filepath.Join(extractDir, "terraform"), binaryPath)

	// 验证二进制文件内容
	content, err := afero.ReadFile(suite.fs, binaryPath)
	suite.NoError(err)
	suite.Equal("terraform binary at root", string(content))
}

func (suite *ExtractorTestSuite) TestBinaryNotFound() {
	// 创建不包含目标二进制文件的ZIP
	files := map[string]string{
		"README.md":   "readme content",
		"other-tool":  "other tool binary",
	}
	zipData := suite.createTestZip(files)

	zipPath := filepath.Join(suite.tempDir, "no-binary.zip")
	err := afero.WriteFile(suite.fs, zipPath, zipData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "no-binary-extracted")

	// 尝试查找不存在的二进制文件
	_, err = suite.extractor.ExtractAndFindBinary(zipPath, extractDir, "kubectl")
	suite.Error(err)
	suite.Contains(err.Error(), "未找到二进制文件")
}

func (suite *ExtractorTestSuite) TestUnsupportedArchiveFormat() {
	// 创建不支持的文件格式
	invalidData := []byte("not an archive")
	invalidPath := filepath.Join(suite.tempDir, "invalid.rar")
	err := afero.WriteFile(suite.fs, invalidPath, invalidData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "invalid-extracted")

	// 尝试提取不支持的格式
	err = suite.extractor.Extract(invalidPath, extractDir)
	suite.Error(err)
	suite.Contains(err.Error(), "不支持的压缩格式")
}

func (suite *ExtractorTestSuite) TestCorruptedArchive() {
	// 创建损坏的ZIP文件
	corruptedData := []byte("PK corrupted zip data")
	corruptedPath := filepath.Join(suite.tempDir, "corrupted.zip")
	err := afero.WriteFile(suite.fs, corruptedPath, corruptedData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "corrupted-extracted")

	// 尝试提取损坏的文件
	err = suite.extractor.Extract(corruptedPath, extractDir)
	suite.Error(err)
}

func (suite *ExtractorTestSuite) TestExtractWithProgress() {
	// 创建大一些的ZIP文件用于测试进度
	files := make(map[string]string)
	for i := 0; i < 50; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("content for file %d", i)
	}
	zipData := suite.createTestZip(files)

	zipPath := filepath.Join(suite.tempDir, "progress-test.zip")
	err := afero.WriteFile(suite.fs, zipPath, zipData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "progress-extracted")

	// 跟踪进度
	var progressCalls []ExtractProgress
	progressCallback := func(progress ExtractProgress) {
		progressCalls = append(progressCalls, progress)
	}

	// 执行带进度的提取
	err = suite.extractor.ExtractWithProgress(zipPath, extractDir, progressCallback)
	suite.NoError(err)

	// 验证进度回调被调用
	suite.Greater(len(progressCalls), 0)

	// 验证最后一次进度是100%
	if len(progressCalls) > 0 {
		lastProgress := progressCalls[len(progressCalls)-1]
		suite.Equal(lastProgress.Processed, lastProgress.Total)
	}

	// 验证所有文件都被提取
	for filename := range files {
		extractedPath := filepath.Join(extractDir, filename)
		exists, err := afero.Exists(suite.fs, extractedPath)
		suite.NoError(err)
		suite.True(exists)
	}
}

func (suite *ExtractorTestSuite) TestGetArchiveType() {
	// 测试通过文件扩展名检测压缩格式
	testCases := []struct {
		filename string
		expected ArchiveType
	}{
		{"file.zip", ArchiveTypeZip},
		{"file.tar.gz", ArchiveTypeTarGz},
		{"file.tgz", ArchiveTypeTarGz},
		{"file.tar", ArchiveTypeTar},
		{"file.txt", ArchiveTypeUnsupported},
		{"file.rar", ArchiveTypeUnsupported},
	}

	for _, tc := range testCases {
		archiveType := suite.extractor.getArchiveType(tc.filename)
		suite.Equal(tc.expected, archiveType, "Archive type mismatch for: %s", tc.filename)
	}
}

func (suite *ExtractorTestSuite) TestSanitizePath() {
	// 测试路径清理功能（防止目录遍历攻击）
	testCases := []struct {
		input    string
		expected string
		safe     bool
	}{
		{"normal/path/file.txt", "normal/path/file.txt", true},
		{"../../../etc/passwd", "etc/passwd", true},
		{"/absolute/path", "absolute/path", true},
		{"./relative/path", "relative/path", true},
		{"", "", true},
		{"..\\..\\windows\\path", "windows/path", true},
	}

	for _, tc := range testCases {
		sanitized, err := suite.extractor.sanitizePath(tc.input)
		if tc.safe {
			suite.NoError(err, "Path should be safe: %s", tc.input)
			suite.Equal(tc.expected, sanitized, "Path sanitization failed for: %s", tc.input)
		} else {
			suite.Error(err, "Path should be unsafe: %s", tc.input)
		}
	}
}

func (suite *ExtractorTestSuite) TestZipSlip() {
	// 测试ZIP Slip攻击防护
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// 创建一个试图写入父目录的文件
	f, err := w.Create("../../../etc/passwd")
	suite.NoError(err)
	_, err = f.Write([]byte("malicious content"))
	suite.NoError(err)

	err = w.Close()
	suite.NoError(err)

	// 写入恶意ZIP文件
	maliciousZip := filepath.Join(suite.tempDir, "malicious.zip")
	err = afero.WriteFile(suite.fs, maliciousZip, buf.Bytes(), 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "malicious-extracted")

	// 提取应该成功，但恶意文件应该被安全地放置在提取目录内
	err = suite.extractor.Extract(maliciousZip, extractDir)
	suite.NoError(err)

	// 验证文件被安全地提取到预期位置
	safeFile := filepath.Join(extractDir, "etc/passwd")
	exists, err := afero.Exists(suite.fs, safeFile)
	suite.NoError(err)
	suite.True(exists)

	// 验证恶意文件没有被写入到父目录
	maliciousFile := filepath.Join(suite.tempDir, "../../../etc/passwd")
	exists, _ = afero.Exists(suite.fs, maliciousFile)
	suite.False(exists)
}

func (suite *ExtractorTestSuite) TestExtractSpecificFile() {
	// 创建包含多个文件的ZIP
	files := map[string]string{
		"file1.txt":      "content1",
		"dir/file2.txt":  "content2",
		"bin/kubectl":    "kubectl binary",
		"bin/helm":       "helm binary",
	}
	zipData := suite.createTestZip(files)

	zipPath := filepath.Join(suite.tempDir, "specific-test.zip")
	err := afero.WriteFile(suite.fs, zipPath, zipData, 0644)
	suite.NoError(err)

	extractDir := filepath.Join(suite.tempDir, "specific-extracted")

	// 只提取特定文件
	err = suite.extractor.ExtractFile(zipPath, "bin/kubectl", extractDir)
	suite.NoError(err)

	// 验证只有指定文件被提取
	kubectlPath := filepath.Join(extractDir, "kubectl")
	exists, err := afero.Exists(suite.fs, kubectlPath)
	suite.NoError(err)
	suite.True(exists)

	content, err := afero.ReadFile(suite.fs, kubectlPath)
	suite.NoError(err)
	suite.Equal("kubectl binary", string(content))

	// 验证其他文件没有被提取
	otherFile := filepath.Join(extractDir, "file1.txt")
	exists, _ = afero.Exists(suite.fs, otherFile)
	suite.False(exists)
}

// TestExtractorTestSuite 运行提取器测试套件
func TestExtractorTestSuite(t *testing.T) {
	suite.Run(t, new(ExtractorTestSuite))
}

// BenchmarkExtractZip ZIP提取性能基准测试
func BenchmarkExtractZip(b *testing.B) {
	fs := afero.NewMemMapFs()
	extractor := NewExtractorWithFs(fs)

	// 创建测试ZIP文件
	files := make(map[string]string)
	for i := 0; i < 100; i++ {
		files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("content for file %d", i)
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for filename, content := range files {
		f, err := w.Create(filename)
		if err != nil {
			b.Fatal(err)
		}
		_, err = f.Write([]byte(content))
		if err != nil {
			b.Fatal(err)
		}
	}
	err := w.Close()
	if err != nil {
		b.Fatal(err)
	}
	zipData := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tempDir := fmt.Sprintf("/tmp/bench-%d", i)
		fs.MkdirAll(tempDir, 0755)

		zipPath := filepath.Join(tempDir, "test.zip")
		err := afero.WriteFile(fs, zipPath, zipData, 0644)
		if err != nil {
			b.Fatal(err)
		}

		extractDir := filepath.Join(tempDir, "extracted")
		err = extractor.Extract(zipPath, extractDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestArchiveType 测试压缩格式类型
func TestArchiveType(t *testing.T) {
	assert.Equal(t, "zip", ArchiveTypeZip.String())
	assert.Equal(t, "tar.gz", ArchiveTypeTarGz.String())
	assert.Equal(t, "tar", ArchiveTypeTar.String())
	assert.Equal(t, "unsupported", ArchiveTypeUnsupported.String())
}

// TestExtractProgress 测试提取进度
func TestExtractProgress(t *testing.T) {
	progress := ExtractProgress{
		Processed: 50,
		Total:     100,
		Current:   "file.txt",
	}

	assert.Equal(t, 50.0, progress.Percentage())
	
	str := progress.String()
	assert.Contains(t, str, "50/100")
	assert.Contains(t, str, "50.0%")
	assert.Contains(t, str, "file.txt")
}

// TestExtractError 测试提取错误类型
func TestExtractError(t *testing.T) {
	err := &ExtractError{
		Archive: "/path/to/archive.zip",
		Message: "提取失败",
		Cause:   fmt.Errorf("underlying error"),
	}

	assert.Contains(t, err.Error(), "提取失败")
	assert.Contains(t, err.Error(), "/path/to/archive.zip")
	assert.Equal(t, fmt.Errorf("underlying error"), err.Unwrap())
}