package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

// Extractor 解压器接口
type Extractor interface {
	// Extract 解压文件
	Extract(archivePath, targetDir string) error

	// ExtractFile 解压指定文件
	ExtractFile(archivePath, fileName, targetPath string) error

	// ListContents 列出压缩包内容
	ListContents(archivePath string) ([]string, error)

	// SupportsFormat 是否支持格式
	SupportsFormat(filename string) bool
}

// BinaryExtractor 二进制文件提取器
type BinaryExtractor interface {
	// ExtractBinary 提取二进制文件
	ExtractBinary(extractDir, toolName string, metadata *types.ToolMetadata) (string, error)

	// FindBinaries 查找二进制文件
	FindBinaries(extractDir string) ([]string, error)

	// SetExecutablePermissions 设置可执行权限
	SetExecutablePermissions(filePath string) error

	// ValidateBinary 验证二进制文件
	ValidateBinary(filePath string) error
}

// ArchiveExtractor 压缩包解压器
type ArchiveExtractor struct {
	fs     afero.Fs
	logger *logrus.Logger
}

// NewArchiveExtractor 创建压缩包解压器
func NewArchiveExtractor(fs afero.Fs, logger *logrus.Logger) Extractor {
	return &ArchiveExtractor{
		fs:     fs,
		logger: logger,
	}
}

// Extract 解压文件
func (e *ArchiveExtractor) Extract(archivePath, targetDir string) error {
	e.logger.Debugf("解压文件: %s -> %s", archivePath, targetDir)

	// 确保目标目录存在
	if err := e.fs.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 根据文件扩展名选择解压方法
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return e.extractTarGz(archivePath, targetDir)
	case strings.HasSuffix(archivePath, ".tar.bz2"):
		return e.extractTarBz2(archivePath, targetDir)
	case strings.HasSuffix(archivePath, ".tar.xz"):
		return e.extractTarXz(archivePath, targetDir)
	case strings.HasSuffix(archivePath, ".zip"):
		return e.extractZip(archivePath, targetDir)
	case strings.HasSuffix(archivePath, ".tar"):
		return e.extractTar(archivePath, targetDir)
	default:
		// 如果不是压缩包，直接复制文件
		return e.copyBinaryFile(archivePath, targetDir)
	}
}

// ExtractFile 解压指定文件
func (e *ArchiveExtractor) ExtractFile(archivePath, fileName, targetPath string) error {
	e.logger.Debugf("解压指定文件: %s 中的 %s -> %s", archivePath, fileName, targetPath)

	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return e.extractTarGzFile(archivePath, fileName, targetPath)
	case strings.HasSuffix(archivePath, ".zip"):
		return e.extractZipFile(archivePath, fileName, targetPath)
	default:
		return fmt.Errorf("不支持的压缩格式: %s", archivePath)
	}
}

// ListContents 列出压缩包内容
func (e *ArchiveExtractor) ListContents(archivePath string) ([]string, error) {
	switch {
	case strings.HasSuffix(archivePath, ".tar.gz") || strings.HasSuffix(archivePath, ".tgz"):
		return e.listTarGzContents(archivePath)
	case strings.HasSuffix(archivePath, ".zip"):
		return e.listZipContents(archivePath)
	default:
		return nil, fmt.Errorf("不支持的压缩格式: %s", archivePath)
	}
}

// SupportsFormat 是否支持格式
func (e *ArchiveExtractor) SupportsFormat(filename string) bool {
	supportedExts := []string{
		".tar.gz", ".tgz", ".tar.bz2", ".tar.xz", ".tar", ".zip",
	}

	for _, ext := range supportedExts {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return true
		}
	}

	return false
}

// extractTarGz 解压tar.gz文件
func (e *ArchiveExtractor) extractTarGz(archivePath, targetDir string) error {
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开压缩文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzReader.Close()

	return e.extractTarReader(gzReader, targetDir)
}

// extractTar 解压tar文件
func (e *ArchiveExtractor) extractTar(archivePath, targetDir string) error {
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开tar文件失败: %w", err)
	}
	defer file.Close()

	return e.extractTarReader(file, targetDir)
}

// extractTarReader 解压tar读取器
func (e *ArchiveExtractor) extractTarReader(reader io.Reader, targetDir string) error {
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取tar条目失败: %w", err)
		}

		targetPath := filepath.Join(targetDir, header.Name)

		// 安全性检查：防止路径遍历攻击
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			e.logger.Warnf("跳过不安全的路径: %s", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := e.fs.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}

		case tar.TypeReg:
			// 创建父目录
			if err := e.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %w", err)
			}

			// 创建文件
			outFile, err := e.fs.Create(targetPath)
			if err != nil {
				return fmt.Errorf("创建文件失败: %w", err)
			}

			// 复制内容
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("写入文件失败: %w", err)
			}
			outFile.Close()

			// 设置权限
			if err := e.fs.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				e.logger.Warnf("设置文件权限失败: %v", err)
			}

		case tar.TypeSymlink, tar.TypeLink:
			// 处理符号链接
			e.logger.Debugf("跳过链接文件: %s", header.Name)
		}
	}

	return nil
}

// extractZip 解压zip文件
func (e *ArchiveExtractor) extractZip(archivePath, targetDir string) error {
	// 对于afero，需要特殊处理zip文件
	if osFs, ok := e.fs.(*afero.OsFs); ok {
		return e.extractZipOS(archivePath, targetDir, osFs)
	}

	return fmt.Errorf("zip解压暂时只支持操作系统文件系统")
}

// extractZipOS 在操作系统文件系统上解压zip
func (e *ArchiveExtractor) extractZipOS(archivePath, targetDir string, osFs *afero.OsFs) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath := filepath.Join(targetDir, file.Name)

		// 安全性检查
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			e.logger.Warnf("跳过不安全的路径: %s", file.Name)
			continue
		}

		if file.FileInfo().IsDir() {
			if err := e.fs.MkdirAll(targetPath, file.FileInfo().Mode()); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		// 创建父目录
		if err := e.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}

		// 打开zip文件中的文件
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开zip中的文件失败: %w", err)
		}

		// 创建目标文件
		dstFile, err := e.fs.Create(targetPath)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("创建目标文件失败: %w", err)
		}

		// 复制内容
		if _, err := io.Copy(dstFile, srcFile); err != nil {
			srcFile.Close()
			dstFile.Close()
			return fmt.Errorf("复制文件内容失败: %w", err)
		}

		srcFile.Close()
		dstFile.Close()

		// 设置权限
		if err := e.fs.Chmod(targetPath, file.FileInfo().Mode()); err != nil {
			e.logger.Warnf("设置文件权限失败: %v", err)
		}
	}

	return nil
}

// extractTarBz2 解压tar.bz2文件
func (e *ArchiveExtractor) extractTarBz2(archivePath, targetDir string) error {
	// 这里需要使用bzip2包，暂时返回不支持
	return fmt.Errorf("tar.bz2格式暂未支持")
}

// extractTarXz 解压tar.xz文件
func (e *ArchiveExtractor) extractTarXz(archivePath, targetDir string) error {
	// 这里需要使用xz包，暂时返回不支持
	return fmt.Errorf("tar.xz格式暂未支持")
}

// copyBinaryFile 复制二进制文件
func (e *ArchiveExtractor) copyBinaryFile(srcPath, targetDir string) error {
	filename := filepath.Base(srcPath)
	targetPath := filepath.Join(targetDir, filename)

	srcFile, err := e.fs.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	if err := e.fs.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	dstFile, err := e.fs.Create(targetPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	// 设置可执行权限
	return e.fs.Chmod(targetPath, 0755)
}

// extractTarGzFile 从tar.gz中解压指定文件
func (e *ArchiveExtractor) extractTarGzFile(archivePath, fileName, targetPath string) error {
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return fmt.Errorf("打开压缩文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取tar条目失败: %w", err)
		}

		if header.Name == fileName || strings.HasSuffix(header.Name, "/"+fileName) {
			// 找到目标文件
			if err := e.fs.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("创建目标目录失败: %w", err)
			}

			outFile, err := e.fs.Create(targetPath)
			if err != nil {
				return fmt.Errorf("创建目标文件失败: %w", err)
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("写入文件失败: %w", err)
			}

			// 设置权限
			return e.fs.Chmod(targetPath, os.FileMode(header.Mode))
		}
	}

	return fmt.Errorf("在压缩包中未找到文件: %s", fileName)
}

// extractZipFile 从zip中解压指定文件
func (e *ArchiveExtractor) extractZipFile(archivePath, fileName, targetPath string) error {
	// 暂时返回不支持
	return fmt.Errorf("从zip中提取指定文件暂未支持")
}

// listTarGzContents 列出tar.gz内容
func (e *ArchiveExtractor) listTarGzContents(archivePath string) ([]string, error) {
	file, err := e.fs.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开压缩文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var files []string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取tar条目失败: %w", err)
		}

		files = append(files, header.Name)
	}

	return files, nil
}

// listZipContents 列出zip内容
func (e *ArchiveExtractor) listZipContents(archivePath string) ([]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开zip文件失败: %w", err)
	}
	defer reader.Close()

	var files []string
	for _, file := range reader.File {
		files = append(files, file.Name)
	}

	return files, nil
}

// DefaultBinaryExtractor 默认二进制文件提取器
type DefaultBinaryExtractor struct {
	fs     afero.Fs
	logger *logrus.Logger
}

// NewBinaryExtractor 创建二进制文件提取器
func NewBinaryExtractor(fs afero.Fs, logger *logrus.Logger) BinaryExtractor {
	return &DefaultBinaryExtractor{
		fs:     fs,
		logger: logger,
	}
}

// ExtractBinary 提取二进制文件
func (e *DefaultBinaryExtractor) ExtractBinary(extractDir, toolName string, metadata *types.ToolMetadata) (string, error) {
	fmt.Fprintf(os.Stderr, "[DEBUG] 提取二进制文件: %s 从 %s\n", toolName, extractDir)

	// 如果配置了具体的二进制文件名
	if metadata != nil && metadata.DownloadConfig.ExtractBinary != "" {
		binaryName := metadata.DownloadConfig.ExtractBinary
		fmt.Fprintf(os.Stderr, "[DEBUG] 配置的二进制文件名: %s\n", binaryName)
		// 尝试多种可能的路径
		possiblePaths := []string{
			filepath.Join(extractDir, binaryName),
			filepath.Join(extractDir, "bin", binaryName),
			filepath.Join(extractDir, toolName, binaryName),
			filepath.Join(extractDir, toolName, "bin", binaryName),
		}

		for _, path := range possiblePaths {
			fmt.Fprintf(os.Stderr, "[DEBUG] 检查路径: %s\n", path)
			if exists, _ := afero.Exists(e.fs, path); exists {
				if info, err := e.fs.Stat(path); err == nil && !info.IsDir() {
					fmt.Fprintf(os.Stderr, "[DEBUG] 找到配置的二进制文件: %s\n", path)
					return path, nil
				} else if info.IsDir() {
					fmt.Fprintf(os.Stderr, "[DEBUG] 路径是目录而不是文件: %s\n", path)
				}
			}
		}
	}

	// 查找二进制文件
	binaries, err := e.FindBinaries(extractDir)
	if err != nil {
		return "", fmt.Errorf("查找二进制文件失败: %w", err)
	}

	if len(binaries) == 0 {
		return "", fmt.Errorf("未找到二进制文件")
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] 找到 %d 个二进制文件: %v\n", len(binaries), binaries)

	// 优先选择与工具名称匹配的文件
	for _, binary := range binaries {
		filename := filepath.Base(binary)
		if strings.EqualFold(filename, toolName) ||
			strings.EqualFold(filename, toolName+".exe") {
			fmt.Fprintf(os.Stderr, "[DEBUG] 找到匹配的二进制文件: %s\n", binary)
			return binary, nil
		}
	}

	// 如果没有精确匹配，选择名称中包含工具名的文件
	for _, binary := range binaries {
		filename := filepath.Base(binary)
		if strings.Contains(strings.ToLower(filename), strings.ToLower(toolName)) {
			fmt.Fprintf(os.Stderr, "[DEBUG] 找到相关的二进制文件: %s\n", binary)
			return binary, nil
		}
	}

	// 如果还是没有，返回第一个找到的二进制文件
	fmt.Fprintf(os.Stderr, "[DEBUG] 使用第一个找到的二进制文件: %s\n", binaries[0])
	return binaries[0], nil
}

// FindBinaries 查找二进制文件
func (e *DefaultBinaryExtractor) FindBinaries(extractDir string) ([]string, error) {
	var binaries []string

	err := afero.Walk(e.fs, extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && e.isBinaryFile(path, info) {
			binaries = append(binaries, path)
		}

		return nil
	})

	return binaries, err
}

// SetExecutablePermissions 设置可执行权限
func (e *DefaultBinaryExtractor) SetExecutablePermissions(filePath string) error {
	e.logger.Debugf("设置可执行权限: %s", filePath)

	// 在Unix系统上设置执行权限
	if runtime.GOOS != "windows" {
		return e.fs.Chmod(filePath, 0755)
	}

	return nil
}

// ValidateBinary 验证二进制文件
func (e *DefaultBinaryExtractor) ValidateBinary(filePath string) error {
	e.logger.Debugf("验证二进制文件: %s", filePath)

	// 检查文件是否存在
	if exists, err := afero.Exists(e.fs, filePath); err != nil {
		return fmt.Errorf("检查文件存在性失败: %w", err)
	} else if !exists {
		return fmt.Errorf("二进制文件不存在: %s", filePath)
	}

	// 检查文件大小
	info, err := e.fs.Stat(filePath)
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("二进制文件为空: %s", filePath)
	}

	// 在Unix系统上检查执行权限
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0111 == 0 {
			e.logger.Warnf("文件没有执行权限，尝试添加: %s", filePath)
			if err := e.SetExecutablePermissions(filePath); err != nil {
				return fmt.Errorf("设置执行权限失败: %w", err)
			}
		}
	}

	return nil
}

// isBinaryFile 判断是否为二进制文件
func (e *DefaultBinaryExtractor) isBinaryFile(path string, info os.FileInfo) bool {
	// 跳过目录
	if info.IsDir() {
		return false
	}

	filename := filepath.Base(path)

	// Windows可执行文件
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(filename), ".exe")
	}

	// Unix系统：检查是否有执行权限
	mode := info.Mode()
	if mode&0111 != 0 {
		return true
	}

	// 检查文件是否没有扩展名（通常是Unix可执行文件）
	ext := filepath.Ext(filename)
	if ext == "" {
		// 进一步检查：排除常见的非可执行文件
		nonExecutableFiles := []string{
			"readme", "license", "changelog", "authors", "contributors",
			"copying", "install", "news", "todo", "version", "history",
		}

		lowerName := strings.ToLower(filename)
		for _, nonExec := range nonExecutableFiles {
			if strings.Contains(lowerName, nonExec) {
				return false
			}
		}

		// 如果在bin/、sbin/目录下且没有扩展名，很可能是可执行文件
		if strings.Contains(path, "/bin/") || strings.Contains(path, "/sbin/") {
			return true
		}

		// 其他没有扩展名的文件也可能是可执行文件
		return true
	}

	return false
}

// PackageProcessor 软件包处理器
type PackageProcessor struct {
	extractor       Extractor
	binaryExtractor BinaryExtractor
	fs              afero.Fs
	logger          *logrus.Logger
}

// NewPackageProcessor 创建软件包处理器
func NewPackageProcessor(fs afero.Fs, logger *logrus.Logger) *PackageProcessor {
	return &PackageProcessor{
		extractor:       NewArchiveExtractor(fs, logger),
		binaryExtractor: NewBinaryExtractor(fs, logger),
		fs:              fs,
		logger:          logger,
	}
}

// ProcessPackage 处理软件包
func (p *PackageProcessor) ProcessPackage(packagePath, targetDir, toolName string, metadata *types.ToolMetadata) (string, error) {
	// 如果toolName为空，尝试使用ExtractBinary作为fallback
	if toolName == "" && metadata != nil && metadata.DownloadConfig.ExtractBinary != "" {
		toolName = metadata.DownloadConfig.ExtractBinary
		p.logger.Debugf("使用ExtractBinary作为toolName: '%s'", toolName)
	}

	p.logger.Debugf("处理软件包: %s", packagePath)

	// 创建临时解压目录
	tempExtractDir := filepath.Join(filepath.Dir(targetDir), "extract_temp")
	if err := p.fs.MkdirAll(tempExtractDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时解压目录失败: %w", err)
	}
	defer p.fs.RemoveAll(tempExtractDir)

	// 解压软件包
	if err := p.extractor.Extract(packagePath, tempExtractDir); err != nil {
		return "", fmt.Errorf("解压软件包失败: %w", err)
	}

	// 调试：列出解压后的文件结构
	p.logger.Debugf("解压后的文件结构:")
	afero.Walk(p.fs, tempExtractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(tempExtractDir, path)
		if info.IsDir() {
			p.logger.Debugf("  [DIR]  %s", relPath)
		} else {
			p.logger.Debugf("  [FILE] %s (size: %d, mode: %s)", relPath, info.Size(), info.Mode())
		}
		return nil
	})

	// 提取二进制文件
	binaryPath, err := p.binaryExtractor.ExtractBinary(tempExtractDir, toolName, metadata)
	if err != nil {
		return "", fmt.Errorf("提取二进制文件失败: %w", err)
	}

	p.logger.Debugf("找到的二进制文件路径: %s", binaryPath)

	// 验证是否为文件而不是目录
	if info, err := p.fs.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("检查二进制文件失败: %w", err)
	} else if info.IsDir() {
		return "", fmt.Errorf("二进制文件路径指向目录而不是文件: %s", binaryPath)
	}

	// 确保目标目录存在
	binDir := filepath.Join(targetDir, "bin")
	if err := p.fs.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("创建二进制目录失败: %w", err)
	}

	// 复制二进制文件到目标位置
	targetBinaryPath := filepath.Join(binDir, toolName)
	if runtime.GOOS == "windows" && !strings.HasSuffix(targetBinaryPath, ".exe") {
		targetBinaryPath += ".exe"
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] 目标二进制路径: %s\n", targetBinaryPath)
	fmt.Fprintf(os.Stderr, "[DEBUG] binDir: %s, toolName: %s\n", binDir, toolName)

	if err := p.copyFile(binaryPath, targetBinaryPath); err != nil {
		return "", fmt.Errorf("复制二进制文件失败: %w", err)
	}

	// 设置可执行权限
	if err := p.binaryExtractor.SetExecutablePermissions(targetBinaryPath); err != nil {
		return "", fmt.Errorf("设置可执行权限失败: %w", err)
	}

	// 验证二进制文件
	if err := p.binaryExtractor.ValidateBinary(targetBinaryPath); err != nil {
		return "", fmt.Errorf("验证二进制文件失败: %w", err)
	}

	p.logger.Infof("软件包处理完成: %s -> %s", packagePath, targetBinaryPath)
	return targetBinaryPath, nil
}

// copyFile 复制文件
func (p *PackageProcessor) copyFile(src, dst string) error {
	srcFile, err := p.fs.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := p.fs.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	return nil
}
