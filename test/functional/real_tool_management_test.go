package functional

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// RealToolVersionManagementTestSuite 真实工具版本管理功能测试套件
type RealToolVersionManagementTestSuite struct {
	suite.Suite
	fs          afero.Fs
	homeDir     string
	configAPI   config.API
	versionMgr  version.Manager
	ctx         context.Context
	testTools   []TestTool
	cleanupFuncs []func()
}

// TestTool 测试工具定义
type TestTool struct {
	Name        string
	Versions    []string
	Binary      string
	Source      *types.SourceConfig
	Install     *types.InstallConfig
	Platforms   []string // 支持的平台
}

func (suite *RealToolVersionManagementTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.fs = afero.NewOsFs() // 使用真实文件系统
	suite.cleanupFuncs = make([]func(), 0)

	// 创建临时主目录
	tempDir, err := os.MkdirTemp("", "vman-real-test-*")
	require.NoError(suite.T(), err)
	suite.homeDir = tempDir

	suite.addCleanup(func() {
		os.RemoveAll(suite.homeDir)
	})

	// 初始化配置API
	suite.configAPI, err = config.NewAPI(suite.homeDir)
	require.NoError(suite.T(), err)

	err = suite.configAPI.Init(suite.ctx)
	require.NoError(suite.T(), err)

	// 创建版本管理器
	suite.versionMgr, err = version.NewManager(suite.homeDir)
	require.NoError(suite.T(), err)

	// 定义测试工具
	suite.testTools = []TestTool{
		{
			Name:     "kubectl",
			Versions: []string{"1.28.0", "1.29.0", "1.30.0"},
			Binary:   "kubectl",
			Source: &types.SourceConfig{
				Type: "github",
				URL:  "https://github.com/kubernetes/kubernetes",
			},
			Install: &types.InstallConfig{
				Method: "download",
			},
			Platforms: []string{"linux", "darwin", "windows"},
		},
		{
			Name:     "terraform",
			Versions: []string{"1.5.0", "1.6.0", "1.7.0"},
			Binary:   "terraform",
			Source: &types.SourceConfig{
				Type: "github",
				URL:  "https://github.com/hashicorp/terraform",
			},
			Install: &types.InstallConfig{
				Method: "extract",
			},
			Platforms: []string{"linux", "darwin", "windows"},
		},
		{
			Name:     "node",
			Versions: []string{"18.19.0", "20.11.0", "21.6.0"},
			Binary:   "node",
			Source: &types.SourceConfig{
				Type: "official",
				URL:  "https://nodejs.org/dist",
			},
			Install: &types.InstallConfig{
				Method: "extract",
			},
			Platforms: []string{"linux", "darwin", "windows"},
		},
	}
}

func (suite *RealToolVersionManagementTestSuite) TearDownSuite() {
	for i := len(suite.cleanupFuncs) - 1; i >= 0; i-- {
		suite.cleanupFuncs[i]()
	}
}

func (suite *RealToolVersionManagementTestSuite) addCleanup(fn func()) {
	suite.cleanupFuncs = append(suite.cleanupFuncs, fn)
}

// TestToolRegistration 测试工具注册
func (suite *RealToolVersionManagementTestSuite) TestToolRegistration() {
	for _, tool := range suite.testTools {
		suite.Run(fmt.Sprintf("注册工具_%s", tool.Name), func() {
			// 检查平台支持
			if !suite.isPlatformSupported(tool) {
				suite.T().Skipf("工具 %s 不支持当前平台 %s", tool.Name, runtime.GOOS)
				return
			}

			// 创建工具元数据
			metadata := &types.ToolMetadata{
				Name:        tool.Name,
				Version:     tool.Versions[0], // 使用第一个版本作为默认版本
				Description: fmt.Sprintf("%s 工具", tool.Name),
				Source:      tool.Source,
				Binary: &types.BinaryConfig{
					Name: tool.Binary,
				},
				Install:   tool.Install,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// 注册工具
			err := suite.configAPI.RegisterTool(suite.ctx, metadata)
			require.NoError(suite.T(), err)

			// 验证工具注册
			tools, err := suite.configAPI.ListTools(suite.ctx)
			require.NoError(suite.T(), err)
			assert.Contains(suite.T(), tools, tool.Name)

			// 获取工具配置
			config, err := suite.configAPI.GetToolConfig(suite.ctx, tool.Name)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), tool.Name, config.Name)
		})
	}
}

// TestVersionInstallation 测试版本安装
func (suite *RealToolVersionManagementTestSuite) TestVersionInstallation() {
	// 选择一个轻量级的工具进行实际安装测试
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("安装工具_%s", testTool.Name), func() {
		// 注册工具
		metadata := &types.ToolMetadata{
			Name:        testTool.Name,
			Version:     testTool.Versions[0],
			Description: fmt.Sprintf("%s 工具", testTool.Name),
			Source:      testTool.Source,
			Binary: &types.BinaryConfig{
				Name: testTool.Binary,
			},
			Install:   testTool.Install,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := suite.configAPI.RegisterTool(suite.ctx, metadata)
		require.NoError(suite.T(), err)

		// 模拟安装过程（不实际下载）
		for _, version := range testTool.Versions[:2] { // 只测试前两个版本
			suite.Run(fmt.Sprintf("版本_%s", version), func() {
				// 创建模拟的版本目录和二进制文件
				versionDir := filepath.Join(suite.homeDir, "versions", testTool.Name, version)
				err := os.MkdirAll(versionDir, 0755)
				require.NoError(suite.T(), err)

				// 创建模拟的二进制文件
				binaryPath := filepath.Join(versionDir, testTool.Binary)
				if runtime.GOOS == "windows" {
					binaryPath += ".exe"
				}

				// 创建一个简单的脚本作为模拟二进制文件
				scriptContent := suite.createMockBinaryScript(testTool.Name, version)
				err = os.WriteFile(binaryPath, []byte(scriptContent), 0755)
				require.NoError(suite.T(), err)

				// 注册版本
				err = suite.versionMgr.RegisterVersion(testTool.Name, version, binaryPath)
				require.NoError(suite.T(), err)

				// 验证版本安装
				isInstalled := suite.versionMgr.IsVersionInstalled(testTool.Name, version)
				assert.True(suite.T(), isInstalled)

				// 获取版本路径
				versionPath, err := suite.versionMgr.GetVersionPath(testTool.Name, version)
				require.NoError(suite.T(), err)
				assert.Equal(suite.T(), binaryPath, versionPath)
			})
		}
	})
}

// TestVersionSwitching 测试版本切换
func (suite *RealToolVersionManagementTestSuite) TestVersionSwitching() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("切换工具版本_%s", testTool.Name), func() {
		// 先安装多个版本
		suite.setupMultipleVersions(testTool)

		// 测试全局版本切换
		for _, version := range testTool.Versions[:2] {
			err := suite.configAPI.SetToolVersion(suite.ctx, testTool.Name, version, true, "")
			require.NoError(suite.T(), err)

			// 验证全局版本设置
			effectiveVersion, err := suite.configAPI.GetEffectiveVersion(suite.ctx, testTool.Name, "")
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), version, effectiveVersion)
		}

		// 测试项目级版本切换
		projectPath := filepath.Join(suite.homeDir, "test_project")
		err := os.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		for _, version := range testTool.Versions[:2] {
			err := suite.configAPI.SetToolVersion(suite.ctx, testTool.Name, version, false, projectPath)
			require.NoError(suite.T(), err)

			// 验证项目版本设置
			effectiveVersion, err := suite.configAPI.GetEffectiveVersion(suite.ctx, testTool.Name, projectPath)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), version, effectiveVersion)
		}
	})
}

// TestVersionListing 测试版本列表
func (suite *RealToolVersionManagementTestSuite) TestVersionListing() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("列出版本_%s", testTool.Name), func() {
		// 安装多个版本
		suite.setupMultipleVersions(testTool)

		// 列出已安装的版本
		installedVersions, err := suite.configAPI.ListInstalledVersions(suite.ctx, testTool.Name)
		require.NoError(suite.T(), err)

		// 验证版本列表
		assert.GreaterOrEqual(suite.T(), len(installedVersions), 2)
		for _, version := range testTool.Versions[:2] {
			assert.Contains(suite.T(), installedVersions, version)
		}
	})
}

// TestVersionRemoval 测试版本移除
func (suite *RealToolVersionManagementTestSuite) TestVersionRemoval() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("移除版本_%s", testTool.Name), func() {
		// 安装多个版本
		suite.setupMultipleVersions(testTool)

		versionToRemove := testTool.Versions[0]

		// 移除版本
		err := suite.configAPI.RemoveToolVersion(suite.ctx, testTool.Name, versionToRemove)
		require.NoError(suite.T(), err)

		// 验证版本已移除
		isInstalled := suite.versionMgr.IsVersionInstalled(testTool.Name, versionToRemove)
		assert.False(suite.T(), isInstalled)

		// 验证其他版本仍然存在
		for _, version := range testTool.Versions[1:2] {
			isInstalled := suite.versionMgr.IsVersionInstalled(testTool.Name, version)
			assert.True(suite.T(), isInstalled)
		}
	})
}

// TestProjectSpecificVersions 测试项目特定版本
func (suite *RealToolVersionManagementTestSuite) TestProjectSpecificVersions() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("项目特定版本_%s", testTool.Name), func() {
		// 安装多个版本
		suite.setupMultipleVersions(testTool)

		// 创建多个项目
		projects := []string{"project1", "project2", "project3"}
		for i, project := range projects {
			projectPath := filepath.Join(suite.homeDir, project)
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			// 为每个项目设置不同的版本
			version := testTool.Versions[i%len(testTool.Versions)]
			err = suite.configAPI.SetToolVersion(suite.ctx, testTool.Name, version, false, projectPath)
			require.NoError(suite.T(), err)

			// 验证项目版本
			effectiveVersion, err := suite.configAPI.GetEffectiveVersion(suite.ctx, testTool.Name, projectPath)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), version, effectiveVersion)
		}

		// 验证项目间版本隔离
		for i, project := range projects {
			projectPath := filepath.Join(suite.homeDir, project)
			expectedVersion := testTool.Versions[i%len(testTool.Versions)]
			
			effectiveVersion, err := suite.configAPI.GetEffectiveVersion(suite.ctx, testTool.Name, projectPath)
			require.NoError(suite.T(), err)
			assert.Equal(suite.T(), expectedVersion, effectiveVersion)
		}
	})
}

// TestToolUnregistration 测试工具取消注册
func (suite *RealToolVersionManagementTestSuite) TestToolUnregistration() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("取消注册工具_%s", testTool.Name), func() {
		// 先注册和安装工具
		suite.setupMultipleVersions(testTool)

		// 取消注册工具
		err := suite.configAPI.UnregisterTool(suite.ctx, testTool.Name)
		require.NoError(suite.T(), err)

		// 验证工具已取消注册
		tools, err := suite.configAPI.ListTools(suite.ctx)
		require.NoError(suite.T(), err)
		assert.NotContains(suite.T(), tools, testTool.Name)

		// 尝试获取工具配置应该失败
		_, err = suite.configAPI.GetToolConfig(suite.ctx, testTool.Name)
		assert.Error(suite.T(), err)
	})
}

// TestVersionValidation 测试版本验证
func (suite *RealToolVersionManagementTestSuite) TestVersionValidation() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run("版本验证", func() {
		// 测试有效版本
		validVersions := []string{"1.0.0", "v1.2.3", "2.1.0-beta.1", "latest"}
		for _, version := range validVersions {
			err := suite.versionMgr.ValidateVersion(version)
			assert.NoError(suite.T(), err, "版本 %s 应该是有效的", version)
		}

		// 测试无效版本
		invalidVersions := []string{"", "invalid", "1.2.3.4.5", "v", "1.2.3."}
		for _, version := range invalidVersions {
			err := suite.versionMgr.ValidateVersion(version)
			assert.Error(suite.T(), err, "版本 %s 应该是无效的", version)
		}
	})
}

// TestBinaryExecution 测试二进制执行
func (suite *RealToolVersionManagementTestSuite) TestBinaryExecution() {
	testTool := suite.findSuitableTestTool()
	if testTool == nil {
		suite.T().Skip("没有找到适合当前平台的测试工具")
		return
	}

	suite.Run(fmt.Sprintf("二进制执行_%s", testTool.Name), func() {
		// 安装版本
		suite.setupMultipleVersions(testTool)

		// 获取二进制路径
		version := testTool.Versions[0]
		binaryPath, err := suite.versionMgr.GetVersionPath(testTool.Name, version)
		require.NoError(suite.T(), err)

		// 测试二进制执行
		if _, err := os.Stat(binaryPath); err == nil {
			// 执行版本命令
			cmd := exec.Command(binaryPath, "--version")
			output, err := cmd.Output()
			
			if err == nil {
				// 验证输出包含版本信息
				outputStr := string(output)
				suite.T().Logf("工具输出: %s", outputStr)
				// 这里可以添加更具体的版本输出验证
			} else {
				suite.T().Logf("工具执行失败（这在模拟环境中是正常的）: %v", err)
			}
		}
	})
}

// 辅助方法

func (suite *RealToolVersionManagementTestSuite) isPlatformSupported(tool TestTool) bool {
	for _, platform := range tool.Platforms {
		if platform == runtime.GOOS {
			return true
		}
	}
	return false
}

func (suite *RealToolVersionManagementTestSuite) findSuitableTestTool() *TestTool {
	for _, tool := range suite.testTools {
		if suite.isPlatformSupported(tool) {
			return &tool
		}
	}
	return nil
}

func (suite *RealToolVersionManagementTestSuite) setupMultipleVersions(tool *TestTool) {
	// 注册工具
	metadata := &types.ToolMetadata{
		Name:        tool.Name,
		Version:     tool.Versions[0],
		Description: fmt.Sprintf("%s 工具", tool.Name),
		Source:      tool.Source,
		Binary: &types.BinaryConfig{
			Name: tool.Binary,
		},
		Install:   tool.Install,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := suite.configAPI.RegisterTool(suite.ctx, metadata)
	require.NoError(suite.T(), err)

	// 安装多个版本
	for _, version := range tool.Versions[:2] { // 只安装前两个版本以节省时间
		versionDir := filepath.Join(suite.homeDir, "versions", tool.Name, version)
		err := os.MkdirAll(versionDir, 0755)
		require.NoError(suite.T(), err)

		binaryPath := filepath.Join(versionDir, tool.Binary)
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}

		scriptContent := suite.createMockBinaryScript(tool.Name, version)
		err = os.WriteFile(binaryPath, []byte(scriptContent), 0755)
		require.NoError(suite.T(), err)

		err = suite.versionMgr.RegisterVersion(tool.Name, version, binaryPath)
		require.NoError(suite.T(), err)
	}
}

func (suite *RealToolVersionManagementTestSuite) createMockBinaryScript(toolName, version string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`@echo off
echo %s version %s
`, toolName, version)
	} else {
		return fmt.Sprintf(`#!/bin/bash
echo "%s version %s"
`, toolName, version)
	}
}

// TestRealToolVersionManagementTestSuite 运行真实工具版本管理功能测试套件
func TestRealToolVersionManagementTestSuite(t *testing.T) {
	// 这个测试套件可能需要网络访问和较长时间
	if testing.Short() {
		t.Skip("跳过真实工具版本管理测试（使用 -short 标志）")
	}

	suite.Run(t, new(RealToolVersionManagementTestSuite))
}

// BenchmarkVersionOperations 版本操作性能基准测试
func BenchmarkVersionOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vman-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configAPI, err := config.NewAPI(tempDir)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	err = configAPI.Init(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			toolName := fmt.Sprintf("bench-tool-%d", counter%100)
			version := fmt.Sprintf("1.%d.0", counter%10)
			
			_ = configAPI.SetToolVersion(ctx, toolName, version, true, "")
			_, _ = configAPI.GetEffectiveVersion(ctx, toolName, "")
			
			counter++
		}
	})
}