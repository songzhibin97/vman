package testutils

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/songzhibin97/vman/pkg/types"
)

// TestDataManager 测试数据管理器
type TestDataManager struct {
	fs      afero.Fs
	baseDir string
}

// NewTestDataManager 创建测试数据管理器
func NewTestDataManager(fs afero.Fs, baseDir string) *TestDataManager {
	return &TestDataManager{
		fs:      fs,
		baseDir: baseDir,
	}
}

// MockTool 模拟工具定义
type MockTool struct {
	Name        string
	Versions    []string
	Description string
	// Binary      string
	// Source      *types.SourceConfig
	// Install     *types.InstallConfig
	Metadata    *types.ToolMetadata
}

// GetCommonTestTools 获取常用测试工具
func (tdm *TestDataManager) GetCommonTestTools() []MockTool {
	return []MockTool{
		{
			Name:        "kubectl",
			Versions:    []string{"1.28.0", "1.29.0", "1.30.0"},
			Description: "Kubernetes命令行工具",
			// Binary:      "kubectl",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "https://github.com/kubernetes/kubernetes",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "download",
			// },
		},
		{
			Name:        "terraform",
			Versions:    []string{"1.5.0", "1.6.0", "1.7.0"},
			Description: "基础设施即代码工具",
			// Binary:      "terraform",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "https://github.com/hashicorp/terraform",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "extract",
			// },
		},
		{
			Name:        "node",
			Versions:    []string{"18.19.0", "20.11.0", "21.6.0"},
			Description: "Node.js运行时",
			// Binary:      "node",
			// Source: &types.SourceConfig{
			// 	Type: "official",
			// 	URL:  "https://nodejs.org/dist",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "extract",
			// },
		},
		{
			Name:        "go",
			Versions:    []string{"1.20.0", "1.21.0", "1.22.0"},
			Description: "Go编程语言",
			// Binary:      "go",
			// Source: &types.SourceConfig{
			// 	Type: "official",
			// 	URL:  "https://golang.org/dl",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "extract",
			// },
		},
		{
			Name:        "helm",
			Versions:    []string{"3.12.0", "3.13.0", "3.14.0"},
			Description: "Kubernetes包管理器",
			// Binary:      "helm",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "https://github.com/helm/helm",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "extract",
			// },
		},
	}
}

// CreateTestToolMetadata 创建测试工具元数据
func (tdm *TestDataManager) CreateTestToolMetadata(tool MockTool) *types.ToolMetadata {
	return &types.ToolMetadata{
		Name:        tool.Name,
		Description: tool.Description,
		// Version:     tool.Versions[0], // 使用第一个版本作为默认
		// Source:      tool.Source,
		// Binary: &types.BinaryConfig{
		// 	Name: tool.Binary,
		// },
		// Install:   tool.Install,
	}
}

// SetupMockToolBinaries 设置模拟工具二进制文件
func (tdm *TestDataManager) SetupMockToolBinaries(tool MockTool) error {
	for _, version := range tool.Versions {
		versionDir := filepath.Join(tdm.baseDir, "versions", tool.Name, version)
		if err := tdm.fs.MkdirAll(versionDir, 0755); err != nil {
			return fmt.Errorf("failed to create version directory: %w", err)
		}

		binaryPath := filepath.Join(versionDir, tool.Name)
		if strings.HasSuffix(os.Getenv("OS"), "Windows_NT") {
			binaryPath += ".exe"
		}

		// 创建模拟二进制脚本
		script := tdm.createMockBinaryScript(tool.Name, version)
		if err := afero.WriteFile(tdm.fs, binaryPath, []byte(script), 0755); err != nil {
			return fmt.Errorf("failed to create mock binary: %w", err)
		}
	}

	return nil
}

// createMockBinaryScript 创建模拟二进制脚本
func (tdm *TestDataManager) createMockBinaryScript(toolName, version string) string {
	if strings.HasSuffix(os.Getenv("OS"), "Windows_NT") {
		return fmt.Sprintf(`@echo off
echo %s version %s
if "%%1"=="--version" (
    echo %s version %s
    exit /b 0
)
if "%%1"=="version" (
    echo %s version %s
    exit /b 0
)
echo Mock %s called with arguments: %%*
`, toolName, version, toolName, version, toolName, version, toolName)
	} else {
		return fmt.Sprintf(`#!/bin/bash
if [ "$1" == "--version" ] || [ "$1" == "version" ]; then
    echo "%s version %s"
    exit 0
fi
echo "Mock %s called with arguments: $@"
`, toolName, version, toolName)
	}
}

// MockDownloadServer 模拟下载服务器
type MockDownloadServer struct {
	server   *httptest.Server
	files    map[string][]byte
	metadata map[string]MockFileMetadata
}

// MockFileMetadata 模拟文件元数据
type MockFileMetadata struct {
	Size         int64
	ContentType  string
	LastModified time.Time
}

// NewMockDownloadServer 创建模拟下载服务器
func NewMockDownloadServer() *MockDownloadServer {
	mds := &MockDownloadServer{
		files:    make(map[string][]byte),
		metadata: make(map[string]MockFileMetadata),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", mds.handleRequest)
	mds.server = httptest.NewServer(mux)

	// 添加默认文件
	mds.addDefaultFiles()

	return mds
}

// URL 获取服务器URL
func (mds *MockDownloadServer) URL() string {
	return mds.server.URL
}

// Close 关闭服务器
func (mds *MockDownloadServer) Close() {
	mds.server.Close()
}

// AddFile 添加文件
func (mds *MockDownloadServer) AddFile(path string, content []byte, contentType string) {
	mds.files[path] = content
	mds.metadata[path] = MockFileMetadata{
		Size:         int64(len(content)),
		ContentType:  contentType,
		LastModified: time.Now(),
	}
}

// addDefaultFiles 添加默认文件
func (mds *MockDownloadServer) addDefaultFiles() {
	// 添加一些常见工具的模拟下载文件
	tools := []struct {
		name     string
		version  string
		platform string
		arch     string
	}{
		{"kubectl", "1.29.0", "linux", "amd64"},
		{"kubectl", "1.29.0", "darwin", "amd64"},
		{"kubectl", "1.29.0", "windows", "amd64"},
		{"terraform", "1.6.0", "linux", "amd64"},
		{"terraform", "1.6.0", "darwin", "amd64"},
		{"terraform", "1.6.0", "windows", "amd64"},
	}

	for _, tool := range tools {
		path := fmt.Sprintf("/releases/%s/%s/%s_%s_%s.zip", tool.name, tool.version, tool.name, tool.platform, tool.arch)
		content := []byte(fmt.Sprintf("Mock %s %s binary for %s/%s", tool.name, tool.version, tool.platform, tool.arch))
		mds.AddFile(path, content, "application/zip")
	}

	// 添加版本信息API
	mds.AddFile("/api/kubectl/releases", []byte(`{
		"releases": [
			{"version": "1.28.0", "date": "2023-08-15"},
			{"version": "1.29.0", "date": "2023-12-13"},
			{"version": "1.30.0", "date": "2024-04-17"}
		]
	}`), "application/json")

	mds.AddFile("/api/terraform/releases", []byte(`{
		"releases": [
			{"version": "1.5.0", "date": "2023-06-12"},
			{"version": "1.6.0", "date": "2023-10-04"},
			{"version": "1.7.0", "date": "2024-01-17"}
		]
	}`), "application/json")
}

// handleRequest 处理HTTP请求
func (mds *MockDownloadServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// 检查文件是否存在
	content, exists := mds.files[path]
	if !exists {
		http.NotFound(w, r)
		return
	}

	metadata := mds.metadata[path]

	// 设置响应头
	w.Header().Set("Content-Type", metadata.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size))
	w.Header().Set("Last-Modified", metadata.LastModified.Format(http.TimeFormat))

	// 模拟慢速下载（用于测试超时）
	if r.URL.Query().Get("slow") == "true" {
		time.Sleep(5 * time.Second)
	}

	// 模拟下载错误
	if r.URL.Query().Get("error") == "true" {
		http.Error(w, "Simulated download error", http.StatusInternalServerError)
		return
	}

	// 发送内容
	w.Write(content)
}

// MockConfigGenerator 模拟配置生成器
type MockConfigGenerator struct {
	fs      afero.Fs
	baseDir string
}

// NewMockConfigGenerator 创建模拟配置生成器
func NewMockConfigGenerator(fs afero.Fs, baseDir string) *MockConfigGenerator {
	return &MockConfigGenerator{
		fs:      fs,
		baseDir: baseDir,
	}
}

// CreateProjectConfig 创建项目配置
func (mcg *MockConfigGenerator) CreateProjectConfig(projectPath string, tools map[string]string) error {
	config := &types.ProjectConfig{
		Version: "1.0",
		Tools:   tools,
	}

	// 这里应该使用YAML序列化，为简化直接创建字符串
	configContent := fmt.Sprintf("version: \"%s\"\ntools:\n", config.Version)
	for tool, version := range tools {
		configContent += fmt.Sprintf("  %s: \"%s\"\n", tool, version)
	}

	configFile := filepath.Join(projectPath, ".vman.yaml")
	return afero.WriteFile(mcg.fs, configFile, []byte(configContent), 0644)
}

// CreateGlobalConfig 创建全局配置
func (mcg *MockConfigGenerator) CreateGlobalConfig(tools map[string]string, settings *types.Settings) error {
	configContent := "version: \"1.0\"\n"

	if len(tools) > 0 {
		configContent += "global_versions:\n"
		for tool, version := range tools {
			configContent += fmt.Sprintf("  %s: \"%s\"\n", tool, version)
		}
	}

	if settings != nil {
		configContent += "settings:\n"
		configContent += "  download:\n"
		configContent += fmt.Sprintf("    timeout: %s\n", settings.Download.Timeout.String())
		configContent += fmt.Sprintf("    retries: %d\n", settings.Download.Retries)
		configContent += fmt.Sprintf("    concurrent_downloads: %d\n", settings.Download.ConcurrentDownloads)
	}

	configFile := filepath.Join(mcg.baseDir, "config.yaml")
	return afero.WriteFile(mcg.fs, configFile, []byte(configContent), 0644)
}

// TestProjectSetup 测试项目设置
type TestProjectSetup struct {
	Name      string
	Path      string
	Tools     map[string]string
	Files     map[string]string // 文件名 -> 内容
	Subdirs   []string
	GitRepo   bool
	ConfigFiles []string
}

// CreateTestProjects 创建测试项目
func (mcg *MockConfigGenerator) CreateTestProjects(projects []TestProjectSetup) error {
	for _, project := range projects {
		projectPath := filepath.Join(mcg.baseDir, project.Path)

		// 创建项目目录
		if err := mcg.fs.MkdirAll(projectPath, 0755); err != nil {
			return fmt.Errorf("failed to create project directory %s: %w", projectPath, err)
		}

		// 创建子目录
		for _, subdir := range project.Subdirs {
			subdirPath := filepath.Join(projectPath, subdir)
			if err := mcg.fs.MkdirAll(subdirPath, 0755); err != nil {
				return fmt.Errorf("failed to create subdirectory %s: %w", subdirPath, err)
			}
		}

		// 创建文件
		for filename, content := range project.Files {
			filePath := filepath.Join(projectPath, filename)
			fileDir := filepath.Dir(filePath)
			if err := mcg.fs.MkdirAll(fileDir, 0755); err != nil {
				return fmt.Errorf("failed to create file directory %s: %w", fileDir, err)
			}
			if err := afero.WriteFile(mcg.fs, filePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create file %s: %w", filePath, err)
			}
		}

		// 创建Git仓库标识
		if project.GitRepo {
			gitDir := filepath.Join(projectPath, ".git")
			if err := mcg.fs.MkdirAll(gitDir, 0755); err != nil {
				return fmt.Errorf("failed to create .git directory: %w", err)
			}
		}

		// 创建vman配置
		if len(project.Tools) > 0 {
			if err := mcg.CreateProjectConfig(projectPath, project.Tools); err != nil {
				return fmt.Errorf("failed to create project config: %w", err)
			}
		}

		// 创建额外的配置文件
		for _, configFile := range project.ConfigFiles {
			configPath := filepath.Join(projectPath, configFile)
			configDir := filepath.Dir(configPath)
			if err := mcg.fs.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
			}

			var content string
			switch filepath.Base(configFile) {
			case "package.json":
				content = `{"name": "test-project", "version": "1.0.0"}`
			case "go.mod":
				content = "module test-project\n\ngo 1.21"
			case "Cargo.toml":
				content = "[package]\nname = \"test-project\"\nversion = \"0.1.0\""
			case "pyproject.toml":
				content = "[tool.poetry]\nname = \"test-project\"\nversion = \"0.1.0\""
			default:
				content = "# Generated config file"
			}

			if err := afero.WriteFile(mcg.fs, configPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create config file %s: %w", configPath, err)
			}
		}
	}

	return nil
}

// GetDefaultTestProjects 获取默认测试项目
func GetDefaultTestProjects() []TestProjectSetup {
	return []TestProjectSetup{
		{
			Name: "Go项目",
			Path: "projects/go-project",
			Tools: map[string]string{
				"go":           "1.21.0",
				"golangci-lint": "1.55.0",
			},
			Files: map[string]string{
				"main.go":    "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
				"README.md":  "# Go测试项目",
			},
			Subdirs: []string{"cmd", "pkg", "internal"},
			GitRepo: true,
			ConfigFiles: []string{"go.mod"},
		},
		{
			Name: "Node.js项目",
			Path: "projects/node-project",
			Tools: map[string]string{
				"node": "20.11.0",
				"npm":  "10.2.0",
			},
			Files: map[string]string{
				"index.js":   "console.log('Hello, Node.js!');",
				"README.md":  "# Node.js测试项目",
			},
			Subdirs: []string{"src", "test"},
			GitRepo: true,
			ConfigFiles: []string{"package.json"},
		},
		{
			Name: "Kubernetes项目",
			Path: "projects/k8s-project",
			Tools: map[string]string{
				"kubectl": "1.29.0",
				"helm":    "3.14.0",
			},
			Files: map[string]string{
				"deployment.yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: test-app",
				"README.md":       "# Kubernetes测试项目",
			},
			Subdirs: []string{"manifests", "charts"},
			GitRepo: true,
			ConfigFiles: []string{},
		},
		{
			Name: "多工具项目",
			Path: "projects/multi-tool-project",
			Tools: map[string]string{
				"kubectl":   "1.29.0",
				"terraform": "1.6.0",
				"node":      "20.11.0",
				"go":        "1.21.0",
			},
			Files: map[string]string{
				"README.md": "# 多工具测试项目",
			},
			Subdirs: []string{"infrastructure", "applications", "scripts"},
			GitRepo: true,
			ConfigFiles: []string{"package.json", "go.mod"},
		},
		{
			Name: "嵌套项目",
			Path: "projects/nested/parent-project",
			Tools: map[string]string{
				"kubectl": "1.28.0",
			},
			Files: map[string]string{
				"README.md": "# 父项目",
			},
			Subdirs: []string{"child1", "child2"},
			GitRepo: true,
			ConfigFiles: []string{},
		},
	}
}

// MockVersionChecker 模拟版本检查器
type MockVersionChecker struct {
	availableVersions map[string][]string
	latestVersions    map[string]string
}

// NewMockVersionChecker 创建模拟版本检查器
func NewMockVersionChecker() *MockVersionChecker {
	mvc := &MockVersionChecker{
		availableVersions: make(map[string][]string),
		latestVersions:    make(map[string]string),
	}

	// 添加默认版本信息
	mvc.addDefaultVersions()

	return mvc
}

// addDefaultVersions 添加默认版本信息
func (mvc *MockVersionChecker) addDefaultVersions() {
	tools := map[string][]string{
		"kubectl":   {"1.26.0", "1.27.0", "1.28.0", "1.29.0", "1.30.0"},
		"terraform": {"1.3.0", "1.4.0", "1.5.0", "1.6.0", "1.7.0"},
		"node":      {"16.20.0", "18.19.0", "20.11.0", "21.6.0"},
		"go":        {"1.19.0", "1.20.0", "1.21.0", "1.22.0"},
		"helm":      {"3.10.0", "3.11.0", "3.12.0", "3.13.0", "3.14.0"},
	}

	for tool, versions := range tools {
		mvc.availableVersions[tool] = versions
		mvc.latestVersions[tool] = versions[len(versions)-1] // 最后一个版本作为最新版本
	}
}

// GetAvailableVersions 获取可用版本
func (mvc *MockVersionChecker) GetAvailableVersions(tool string) []string {
	if versions, exists := mvc.availableVersions[tool]; exists {
		return versions
	}
	return []string{}
}

// GetLatestVersion 获取最新版本
func (mvc *MockVersionChecker) GetLatestVersion(tool string) string {
	if version, exists := mvc.latestVersions[tool]; exists {
		return version
	}
	return ""
}

// IsVersionAvailable 检查版本是否可用
func (mvc *MockVersionChecker) IsVersionAvailable(tool, version string) bool {
	if versions, exists := mvc.availableVersions[tool]; exists {
		for _, v := range versions {
			if v == version {
				return true
			}
		}
	}
	return false
}

// TestEnvironmentSetup 测试环境设置
type TestEnvironmentSetup struct {
	TempDir         string
	DataManager     *TestDataManager
	ConfigGenerator *MockConfigGenerator
	DownloadServer  *MockDownloadServer
	VersionChecker  *MockVersionChecker
	CleanupFuncs    []func()
}

// SetupTestEnvironment 设置测试环境
func SetupTestEnvironment() (*TestEnvironmentSetup, error) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "vman-test-env-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	fs := afero.NewOsFs()

	setup := &TestEnvironmentSetup{
		TempDir:         tempDir,
		DataManager:     NewTestDataManager(fs, tempDir),
		ConfigGenerator: NewMockConfigGenerator(fs, tempDir),
		DownloadServer:  NewMockDownloadServer(),
		VersionChecker:  NewMockVersionChecker(),
		CleanupFuncs:    make([]func(), 0),
	}

	// 添加清理函数
	setup.addCleanup(func() {
		os.RemoveAll(tempDir)
	})

	setup.addCleanup(func() {
		setup.DownloadServer.Close()
	})

	return setup, nil
}

// addCleanup 添加清理函数
func (tes *TestEnvironmentSetup) addCleanup(fn func()) {
	tes.CleanupFuncs = append(tes.CleanupFuncs, fn)
}

// Cleanup 清理测试环境
func (tes *TestEnvironmentSetup) Cleanup() {
	for i := len(tes.CleanupFuncs) - 1; i >= 0; i-- {
		tes.CleanupFuncs[i]()
	}
}

// SetupCompleteTestEnvironment 设置完整的测试环境
func (tes *TestEnvironmentSetup) SetupCompleteTestEnvironment() error {
	// 创建测试工具
	tools := tes.DataManager.GetCommonTestTools()
	for _, tool := range tools {
		if err := tes.DataManager.SetupMockToolBinaries(tool); err != nil {
			return fmt.Errorf("failed to setup mock binaries for %s: %w", tool.Name, err)
		}
	}

	// 创建测试项目
	projects := GetDefaultTestProjects()
	if err := tes.ConfigGenerator.CreateTestProjects(projects); err != nil {
		return fmt.Errorf("failed to create test projects: %w", err)
	}

	// 创建全局配置
	globalTools := map[string]string{
		"kubectl":   "1.29.0",
		"terraform": "1.6.0",
	}

	globalSettings := &types.Settings{
		Download: types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             3,
			ConcurrentDownloads: 2,
		},
		Proxy: types.ProxySettings{
			Enabled:     true,
			ShimsInPath: true,
		},
		Logging: types.LoggingSettings{
			Level: "info",
			File:  "",
		},
	}

	if err := tes.ConfigGenerator.CreateGlobalConfig(globalTools, globalSettings); err != nil {
		return fmt.Errorf("failed to create global config: %w", err)
	}

	return nil
}

// WaitForCondition 等待条件满足
func WaitForCondition(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// CompareVersions 比较版本号
func CompareVersions(v1, v2 string) int {
	// 简化的版本比较，实际实现应该更复杂
	if v1 == v2 {
		return 0
	}
	if v1 < v2 {
		return -1
	}
	return 1
}

// GetMockContext 获取模拟上下文
func GetMockContext() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "test", true)
	ctx = context.WithValue(ctx, "mock", true)
	return ctx
}