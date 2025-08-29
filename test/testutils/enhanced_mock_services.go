package testutils

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"

	"github.com/songzhibin97/vman/pkg/types"
)

// TestDataManager 测试数据管理器
type TestDataManager struct {
	fs      afero.Fs
	dataDir string
}

// NewTestDataManager 创建测试数据管理器
func NewTestDataManager(fs afero.Fs, dataDir string) *TestDataManager {
	return &TestDataManager{
		fs:      fs,
		dataDir: dataDir,
	}
}

// CreateStandardToolMetadata 创建标准工具元数据
func (m *TestDataManager) CreateStandardToolMetadata() map[string]*types.ToolMetadata {
	return map[string]*types.ToolMetadata{
		"kubectl": {
			Name:        "kubectl",
			Version:     "1.29.0",
			Description: "Kubernetes命令行工具",
			Source: &types.SourceConfig{
				Type: "github",
				URL:  "https://github.com/kubernetes/kubernetes",
			},
			Binary: &types.BinaryConfig{
				Name: "kubectl",
			},
			Install: &types.InstallConfig{
				Method: "download",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		"terraform": {
			Name:        "terraform",
			Version:     "1.6.0",
			Description: "基础设施即代码工具",
			Source: &types.SourceConfig{
				Type: "github",
				URL:  "https://github.com/hashicorp/terraform",
			},
			Binary: &types.BinaryConfig{
				Name: "terraform",
			},
			Install: &types.InstallConfig{
				Method: "extract",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		"helm": {
			Name:        "helm",
			Version:     "3.14.0",
			Description: "Kubernetes包管理器",
			Source: &types.SourceConfig{
				Type: "github",
				URL:  "https://github.com/helm/helm",
			},
			Binary: &types.BinaryConfig{
				Name: "helm",
			},
			Install: &types.InstallConfig{
				Method: "extract",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// CreateTestConfigurations 创建测试配置
func (m *TestDataManager) CreateTestConfigurations() (*types.GlobalConfig, map[string]*types.ProjectConfig) {
	// 全局配置
	globalConfig := types.GetDefaultGlobalConfig()
	globalConfig.GlobalVersions = map[string]string{
		"kubectl":   "1.29.0",
		"terraform": "1.6.0",
		"helm":      "3.14.0",
	}

	// 项目配置
	projectConfigs := map[string]*types.ProjectConfig{
		"go_project": {
			Version: "1.0",
			Tools: map[string]string{
				"kubectl": "1.28.0", // 不同于全局版本
				"go":      "1.21.0",
			},
		},
		"node_project": {
			Version: "1.0",
			Tools: map[string]string{
				"node": "20.11.0",
				"npm":  "10.2.0",
			},
		},
		"terraform_project": {
			Version: "1.0",
			Tools: map[string]string{
				"terraform": "1.5.0", // 旧版本
				"kubectl":   "1.29.0",
			},
		},
	}

	return globalConfig, projectConfigs
}

// CreateMockBinaries 创建模拟二进制文件
func (m *TestDataManager) CreateMockBinaries() map[string][]byte {
	return map[string][]byte{
		"kubectl": []byte("#!/bin/bash\necho 'kubectl version'"),
		"terraform": []byte("#!/bin/bash\necho 'Terraform v1.6.0'"),
		"helm": []byte("#!/bin/bash\necho 'helm version'"),
		"node": []byte("#!/bin/bash\necho 'v20.11.0'"),
	}
}

// CreateTestArchives 创建测试压缩包
func (m *TestDataManager) CreateTestArchives() map[string][]byte {
	archives := make(map[string][]byte)

	// 创建ZIP格式的kubectl
	kubectlZip := m.createZipArchive(map[string][]byte{
		"kubectl": []byte("#!/bin/bash\necho 'kubectl version v1.29.0'"),
		"LICENSE": []byte("Apache License 2.0"),
	})
	archives["kubectl-1.29.0.zip"] = kubectlZip

	// 创建tar.gz格式的terraform
	terraformTarGz := m.createTarGzArchive(map[string][]byte{
		"terraform": []byte("#!/bin/bash\necho 'Terraform v1.6.0'"),
		"README.md": []byte("# Terraform"),
	})
	archives["terraform-1.6.0.tar.gz"] = terraformTarGz

	// 创建tar格式的helm
	helmTar := m.createTarArchive(map[string][]byte{
		"helm": []byte("#!/bin/bash\necho 'helm version v3.14.0'"),
		"docs/README.md": []byte("# Helm Documentation"),
	})
	archives["helm-3.14.0.tar"] = helmTar

	return archives
}

// createZipArchive 创建ZIP压缩包
func (m *TestDataManager) createZipArchive(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for filename, content := range files {
		f, err := w.Create(filename)
		if err != nil {
			continue
		}
		f.Write(content)
	}

	w.Close()
	return buf.Bytes()
}

// createTarGzArchive 创建tar.gz压缩包
func (m *TestDataManager) createTarGzArchive(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	for filename, content := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0755,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write(content)
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// createTarArchive 创建tar压缩包
func (m *TestDataManager) createTarArchive(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	for filename, content := range files {
		hdr := &tar.Header{
			Name: filename,
			Mode: 0755,
			Size: int64(len(content)),
		}
		tw.WriteHeader(hdr)
		tw.Write(content)
	}

	tw.Close()
	return buf.Bytes()
}

// MockDownloadServer 模拟下载服务器
type MockDownloadServer struct {
	server   *httptest.Server
	files    map[string][]byte
	metadata map[string]DownloadMetadata
}

// DownloadMetadata 下载元数据
type DownloadMetadata struct {
	ContentType   string
	ContentLength int64
	Checksum      string
	LastModified  time.Time
}

// NewMockDownloadServer 创建模拟下载服务器
func NewMockDownloadServer() *MockDownloadServer {
	mds := &MockDownloadServer{
		files:    make(map[string][]byte),
		metadata: make(map[string]DownloadMetadata),
	}

	mds.server = httptest.NewServer(http.HandlerFunc(mds.handler))
	return mds
}

// Close 关闭服务器
func (mds *MockDownloadServer) Close() {
	mds.server.Close()
}

// URL 获取服务器URL
func (mds *MockDownloadServer) URL() string {
	return mds.server.URL
}

// AddFile 添加文件
func (mds *MockDownloadServer) AddFile(path string, content []byte, contentType string) {
	mds.files[path] = content
	mds.metadata[path] = DownloadMetadata{
		ContentType:   contentType,
		ContentLength: int64(len(content)),
		LastModified:  time.Now(),
	}
}

// handler HTTP处理器
func (mds *MockDownloadServer) handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	
	// 移除前导斜杠
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// 模拟延迟
	if r.URL.Query().Get("slow") == "true" {
		time.Sleep(100 * time.Millisecond)
	}

	// 模拟错误
	if r.URL.Query().Get("error") == "true" {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 检查文件是否存在
	content, exists := mds.files[path]
	if !exists {
		http.NotFound(w, r)
		return
	}

	// 获取元数据
	metadata := mds.metadata[path]

	// 设置响应头
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.ContentLength))
	w.Header().Set("Last-Modified", metadata.LastModified.Format(http.TimeFormat))

	// 支持Range请求（断点续传）
	if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
		// 简单的Range支持实现
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(http.StatusPartialContent)
	}

	// 写入内容
	w.Write(content)
}

// MockVersionChecker 模拟版本检查服务
type MockVersionChecker struct {
	server   *httptest.Server
	versions map[string][]string
}

// NewMockVersionChecker 创建模拟版本检查服务
func NewMockVersionChecker() *MockVersionChecker {
	mvc := &MockVersionChecker{
		versions: make(map[string][]string),
	}

	mvc.server = httptest.NewServer(http.HandlerFunc(mvc.handler))
	return mvc
}

// Close 关闭服务器
func (mvc *MockVersionChecker) Close() {
	mvc.server.Close()
}

// URL 获取服务器URL
func (mvc *MockVersionChecker) URL() string {
	return mvc.server.URL
}

// AddToolVersions 添加工具版本
func (mvc *MockVersionChecker) AddToolVersions(tool string, versions []string) {
	mvc.versions[tool] = versions
}

// handler HTTP处理器
func (mvc *MockVersionChecker) handler(w http.ResponseWriter, r *http.Request) {
	// 解析请求路径 /api/v1/tools/{tool}/versions
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "tools" {
		http.NotFound(w, r)
		return
	}

	tool := parts[3]
	if len(parts) == 5 && parts[4] == "versions" {
		// 获取版本列表
		versions, exists := mvc.versions[tool]
		if !exists {
			http.NotFound(w, r)
			return
		}

		response := map[string]interface{}{
			"tool":     tool,
			"versions": versions,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		http.NotFound(w, r)
	}
}

// ConfigGenerator 配置生成器
type ConfigGenerator struct {
	fs afero.Fs
}

// NewConfigGenerator 创建配置生成器
func NewConfigGenerator(fs afero.Fs) *ConfigGenerator {
	return &ConfigGenerator{fs: fs}
}

// GenerateGlobalConfig 生成全局配置文件
func (cg *ConfigGenerator) GenerateGlobalConfig(path string, config *types.GlobalConfig) error {
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return afero.WriteFile(cg.fs, path, content, 0644)
}

// GenerateProjectConfig 生成项目配置文件
func (cg *ConfigGenerator) GenerateProjectConfig(path string, config *types.ProjectConfig) error {
	// 生成YAML格式
	yamlContent := fmt.Sprintf("version: \"%s\"\ntools:\n", config.Version)
	for tool, version := range config.Tools {
		yamlContent += fmt.Sprintf("  %s: \"%s\"\n", tool, version)
	}

	return afero.WriteFile(cg.fs, path, []byte(yamlContent), 0644)
}

// GenerateToolConfig 生成工具配置文件
func (cg *ConfigGenerator) GenerateToolConfig(path string, metadata *types.ToolMetadata) error {
	// 生成TOML格式
	tomlContent := fmt.Sprintf(`name = "%s"
version = "%s"
description = "%s"

[source]
type = "%s"
url = "%s"

[binary]
name = "%s"

[install]
method = "%s"
`, metadata.Name, metadata.Version, metadata.Description,
		metadata.Source.Type, metadata.Source.URL,
		metadata.Binary.Name,
		metadata.Install.Method)

	return afero.WriteFile(cg.fs, path, []byte(tomlContent), 0644)
}

// TestEnvironmentSetup 测试环境设置
type TestEnvironmentSetup struct {
	fs       afero.Fs
	dataManager    *TestDataManager
	downloadServer *MockDownloadServer
	versionChecker *MockVersionChecker
	configGenerator *ConfigGenerator
	tempDir        string
}

// NewTestEnvironmentSetup 创建测试环境设置
func NewTestEnvironmentSetup(fs afero.Fs, tempDir string) *TestEnvironmentSetup {
	return &TestEnvironmentSetup{
		fs:              fs,
		tempDir:         tempDir,
		dataManager:     NewTestDataManager(fs, filepath.Join(tempDir, "data")),
		downloadServer:  NewMockDownloadServer(),
		versionChecker:  NewMockVersionChecker(),
		configGenerator: NewConfigGenerator(fs),
	}
}

// Setup 设置测试环境
func (tes *TestEnvironmentSetup) Setup() error {
	// 创建目录结构
	dirs := []string{
		filepath.Join(tes.tempDir, "config"),
		filepath.Join(tes.tempDir, "tools"),
		filepath.Join(tes.tempDir, "versions"),
		filepath.Join(tes.tempDir, "cache"),
		filepath.Join(tes.tempDir, "logs"),
		filepath.Join(tes.tempDir, "data"),
	}

	for _, dir := range dirs {
		if err := tes.fs.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 设置标准工具数据
	if err := tes.setupStandardTools(); err != nil {
		return err
	}

	// 设置模拟服务
	if err := tes.setupMockServices(); err != nil {
		return err
	}

	return nil
}

// Cleanup 清理测试环境
func (tes *TestEnvironmentSetup) Cleanup() {
	if tes.downloadServer != nil {
		tes.downloadServer.Close()
	}
	if tes.versionChecker != nil {
		tes.versionChecker.Close()
	}
}

// GetDownloadServerURL 获取下载服务器URL
func (tes *TestEnvironmentSetup) GetDownloadServerURL() string {
	return tes.downloadServer.URL()
}

// GetVersionCheckerURL 获取版本检查服务URL
func (tes *TestEnvironmentSetup) GetVersionCheckerURL() string {
	return tes.versionChecker.URL()
}

// setupStandardTools 设置标准工具
func (tes *TestEnvironmentSetup) setupStandardTools() error {
	// 获取标准工具元数据
	toolMetadata := tes.dataManager.CreateStandardToolMetadata()

	// 生成工具配置文件
	toolsDir := filepath.Join(tes.tempDir, "tools")
	for name, metadata := range toolMetadata {
		configPath := filepath.Join(toolsDir, name+".toml")
		if err := tes.configGenerator.GenerateToolConfig(configPath, metadata); err != nil {
			return err
		}
	}

	// 创建模拟二进制文件
	binaries := tes.dataManager.CreateMockBinaries()
	versionsDir := filepath.Join(tes.tempDir, "versions")
	
	for tool, content := range binaries {
		metadata := toolMetadata[tool]
		if metadata == nil {
			continue
		}

		versionDir := filepath.Join(versionsDir, tool, metadata.Version)
		if err := tes.fs.MkdirAll(versionDir, 0755); err != nil {
			return err
		}

		binaryPath := filepath.Join(versionDir, tool)
		if err := afero.WriteFile(tes.fs, binaryPath, content, 0755); err != nil {
			return err
		}
	}

	return nil
}

// setupMockServices 设置模拟服务
func (tes *TestEnvironmentSetup) setupMockServices() error {
	// 添加下载文件
	archives := tes.dataManager.CreateTestArchives()
	for filename, content := range archives {
		var contentType string
		switch {
		case strings.HasSuffix(filename, ".zip"):
			contentType = "application/zip"
		case strings.HasSuffix(filename, ".tar.gz"):
			contentType = "application/gzip"
		case strings.HasSuffix(filename, ".tar"):
			contentType = "application/x-tar"
		default:
			contentType = "application/octet-stream"
		}
		tes.downloadServer.AddFile(filename, content, contentType)
	}

	// 添加版本信息
	versions := map[string][]string{
		"kubectl":   {"1.28.0", "1.29.0", "1.30.0"},
		"terraform": {"1.5.0", "1.6.0", "1.7.0"},
		"helm":      {"3.13.0", "3.14.0", "3.15.0"},
		"node":      {"18.19.0", "20.11.0", "21.6.0"},
	}

	for tool, versionList := range versions {
		tes.versionChecker.AddToolVersions(tool, versionList)
	}

	return nil
}

// MockProgressReporter 模拟进度报告器
type MockProgressReporter struct {
	mock.Mock
	Updates []ProgressUpdate
}

// ProgressUpdate 进度更新
type ProgressUpdate struct {
	Current int64
	Total   int64
	Message string
}

// Report 报告进度
func (mpr *MockProgressReporter) Report(current, total int64, message string) {
	mpr.Updates = append(mpr.Updates, ProgressUpdate{
		Current: current,
		Total:   total,
		Message: message,
	})
	mpr.Called(current, total, message)
}

// GetUpdates 获取所有更新
func (mpr *MockProgressReporter) GetUpdates() []ProgressUpdate {
	return mpr.Updates
}

// Clear 清空更新记录
func (mpr *MockProgressReporter) Clear() {
	mpr.Updates = nil
}

// PerformanceProfiler 性能分析器
type PerformanceProfiler struct {
	operations map[string][]time.Duration
}

// NewPerformanceProfiler 创建性能分析器
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		operations: make(map[string][]time.Duration),
	}
}

// StartOperation 开始操作
func (pp *PerformanceProfiler) StartOperation(name string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		pp.operations[name] = append(pp.operations[name], duration)
	}
}

// GetStats 获取统计信息
func (pp *PerformanceProfiler) GetStats(operation string) map[string]interface{} {
	durations := pp.operations[operation]
	if len(durations) == 0 {
		return nil
	}

	var total time.Duration
	min := durations[0]
	max := durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	return map[string]interface{}{
		"count":   len(durations),
		"total":   total,
		"average": total / time.Duration(len(durations)),
		"min":     min,
		"max":     max,
	}
}

// GetAllStats 获取所有操作的统计信息
func (pp *PerformanceProfiler) GetAllStats() map[string]map[string]interface{} {
	stats := make(map[string]map[string]interface{})
	for operation := range pp.operations {
		stats[operation] = pp.GetStats(operation)
	}
	return stats
}

// MemoryTracker 内存跟踪器
type MemoryTracker struct {
	snapshots []MemorySnapshot
}

// MemorySnapshot 内存快照
type MemorySnapshot struct {
	Timestamp time.Time
	Label     string
	// 在实际实现中，这里会包含内存使用信息
	// 由于这是测试代码，我们简化处理
	AllocBytes uint64
	TotalBytes uint64
}

// NewMemoryTracker 创建内存跟踪器
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		snapshots: make([]MemorySnapshot, 0),
	}
}

// Snapshot 创建内存快照
func (mt *MemoryTracker) Snapshot(label string) {
	// 在实际实现中，这里会获取真实的内存使用情况
	// 这里我们创建一个模拟的快照
	snapshot := MemorySnapshot{
		Timestamp:  time.Now(),
		Label:      label,
		AllocBytes: 1024 * 1024, // 模拟值
		TotalBytes: 2048 * 1024, // 模拟值
	}
	mt.snapshots = append(mt.snapshots, snapshot)
}

// GetSnapshots 获取所有快照
func (mt *MemoryTracker) GetSnapshots() []MemorySnapshot {
	return mt.snapshots
}

// GetMemoryGrowth 获取内存增长情况
func (mt *MemoryTracker) GetMemoryGrowth() map[string]uint64 {
	if len(mt.snapshots) < 2 {
		return nil
	}

	first := mt.snapshots[0]
	last := mt.snapshots[len(mt.snapshots)-1]

	return map[string]uint64{
		"alloc_growth": last.AllocBytes - first.AllocBytes,
		"total_growth": last.TotalBytes - first.TotalBytes,
	}
}

// TestRunner 测试运行器
type TestRunner struct {
	ctx                context.Context
	setup              *TestEnvironmentSetup
	profiler           *PerformanceProfiler
	memoryTracker      *MemoryTracker
	progressReporter   *MockProgressReporter
}

// NewTestRunner 创建测试运行器
func NewTestRunner(ctx context.Context, setup *TestEnvironmentSetup) *TestRunner {
	return &TestRunner{
		ctx:              ctx,
		setup:            setup,
		profiler:         NewPerformanceProfiler(),
		memoryTracker:    NewMemoryTracker(),
		progressReporter: &MockProgressReporter{},
	}
}

// RunTest 运行测试
func (tr *TestRunner) RunTest(name string, testFunc func() error) error {
	// 开始性能分析
	stopProfiler := tr.profiler.StartOperation(name)
	defer stopProfiler()

	// 内存快照
	tr.memoryTracker.Snapshot(fmt.Sprintf("%s_start", name))

	// 运行测试
	err := testFunc()

	// 结束内存快照
	tr.memoryTracker.Snapshot(fmt.Sprintf("%s_end", name))

	return err
}

// GetResults 获取测试结果
func (tr *TestRunner) GetResults() map[string]interface{} {
	return map[string]interface{}{
		"performance": tr.profiler.GetAllStats(),
		"memory":      tr.memoryTracker.GetMemoryGrowth(),
		"progress":    tr.progressReporter.GetUpdates(),
	}
}