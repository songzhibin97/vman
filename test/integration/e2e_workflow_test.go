package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// E2EWorkflowTestSuite 端到端工作流测试套件
type E2EWorkflowTestSuite struct {
	suite.Suite
	tmpDir         string
	configManager  config.Manager
	storageManager storage.Manager
	versionManager version.Manager
	configAPI      config.API
}

// SetupSuite 设置测试套件
func (suite *E2EWorkflowTestSuite) SetupSuite() {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "vman-e2e-test-*")
	require.NoError(suite.T(), err)
	suite.tmpDir = tmpDir

	// 创建管理器实例
	suite.configManager, err = config.NewManager(tmpDir)
	require.NoError(suite.T(), err)

	configPaths := types.DefaultConfigPaths(tmpDir)
	suite.storageManager = storage.NewFilesystemManager(configPaths)
	suite.versionManager = version.NewManager(suite.storageManager, suite.configManager)

	suite.configAPI, err = config.NewAPI(tmpDir)
	require.NoError(suite.T(), err)

	// 初始化
	err = suite.storageManager.EnsureDirectories()
	require.NoError(suite.T(), err)

	err = suite.configManager.Initialize()
	require.NoError(suite.T(), err)
}

// TearDownSuite 清理测试套件
func (suite *E2EWorkflowTestSuite) TearDownSuite() {
	if suite.tmpDir != "" {
		os.RemoveAll(suite.tmpDir)
	}
}

// TestCompleteToolManagementWorkflow 测试完整的工具管理工作流
func (suite *E2EWorkflowTestSuite) TestCompleteToolManagementWorkflow() {
	ctx := context.Background()
	toolName := "test-tool"
	version1 := "1.0.0"
	version2 := "1.1.0"

	// 1. 初始化配置
	err := suite.configAPI.Init(ctx)
	suite.NoError(err)

	// 2. 创建测试二进制文件
	testBinary1 := suite.createTestBinary(toolName, version1, "#!/bin/bash\necho 'test-tool v1.0.0'")
	testBinary2 := suite.createTestBinary(toolName, version2, "#!/bin/bash\necho 'test-tool v1.1.0'")

	// 3. 注册工具版本
	err = suite.versionManager.RegisterVersion(toolName, version1, testBinary1)
	suite.NoError(err)

	err = suite.versionManager.RegisterVersion(toolName, version2, testBinary2)
	suite.NoError(err)

	// 4. 验证版本已注册
	versions, err := suite.versionManager.ListVersions(toolName)
	suite.NoError(err)
	suite.Contains(versions, version1)
	suite.Contains(versions, version2)

	// 5. 设置全局版本
	err = suite.versionManager.SetGlobalVersion(toolName, version1)
	suite.NoError(err)

	// 6. 验证全局版本设置
	currentVersion, err := suite.versionManager.GetCurrentVersion(toolName)
	suite.NoError(err)
	suite.Equal(version1, currentVersion)

	// 7. 创建项目配置
	projectPath := filepath.Join(suite.tmpDir, "test-project")
	err = os.MkdirAll(projectPath, 0755)
	suite.NoError(err)

	err = suite.configAPI.CreateProjectConfig(ctx, projectPath)
	suite.NoError(err)

	// 8. 设置项目版本
	err = suite.configAPI.SetToolVersion(ctx, toolName, version2, false, projectPath)
	suite.NoError(err)

	// 9. 验证项目版本覆盖全局版本
	effectiveVersion, err := suite.configAPI.GetEffectiveVersion(ctx, toolName, projectPath)
	suite.NoError(err)
	suite.Equal(version2, effectiveVersion)

	// 10. 验证在项目外仍使用全局版本
	effectiveVersionGlobal, err := suite.configAPI.GetEffectiveVersion(ctx, toolName, "/some/other/path")
	suite.NoError(err)
	suite.Equal(version1, effectiveVersionGlobal)

	// 11. 获取有效配置
	effectiveConfig, err := suite.configAPI.GetEffectiveConfig(ctx, projectPath)
	suite.NoError(err)
	suite.NotNil(effectiveConfig)
	suite.Equal(version2, effectiveConfig.ResolvedVersions[toolName])
	// ConfigSource可能返回"project"或完整路径，只要不是"global"就行
	suite.NotEqual("global", effectiveConfig.ConfigSource[toolName])

	// 12. 列出已安装的工具
	tools, err := suite.versionManager.ListAllTools()
	suite.NoError(err)
	suite.Contains(tools, toolName)

	// 13. 获取版本元数据
	metadata, err := suite.versionManager.GetVersionMetadata(toolName, version1)
	suite.NoError(err)
	suite.Equal(toolName, metadata.ToolName)
	suite.Equal(version1, metadata.Version)
	suite.Equal("manual", metadata.InstallType)

	// 14. 切换到另一个版本
	err = suite.versionManager.SetGlobalVersion(toolName, version2)
	suite.NoError(err)

	currentVersion, err = suite.versionManager.GetCurrentVersion(toolName)
	suite.NoError(err)
	suite.Equal(version2, currentVersion)

	// 15. 移除一个版本
	err = suite.versionManager.RemoveVersion(toolName, version1)
	suite.NoError(err)

	// 16. 验证版本已移除
	versions, err = suite.versionManager.ListVersions(toolName)
	suite.NoError(err)
	suite.NotContains(versions, version1)
	suite.Contains(versions, version2)

	// 17. 验证版本文件已删除
	suite.False(suite.versionManager.IsVersionInstalled(toolName, version1))
	suite.True(suite.versionManager.IsVersionInstalled(toolName, version2))
}

// TestConfigurationPersistence 测试配置持久化
func (suite *E2EWorkflowTestSuite) TestConfigurationPersistence() {
	ctx := context.Background()
	_ = ctx // 使用ctx变量

	// 设置配置
	err := suite.configAPI.SetGlobalSetting(ctx, "download.timeout", 600*time.Second)
	suite.NoError(err)

	err = suite.configAPI.SetGlobalSetting(ctx, "proxy.enabled", true)
	suite.NoError(err)

	// 创建新的管理器实例（模拟重启）
	newConfigAPI, err := config.NewAPI(suite.tmpDir)
	suite.NoError(err)

	err = newConfigAPI.Init(ctx)
	suite.NoError(err)

	// 验证配置已持久化
	timeout, err := newConfigAPI.GetGlobalSetting(ctx, "download.timeout")
	suite.NoError(err)
	suite.Equal(600*time.Second, timeout)

	proxyEnabled, err := newConfigAPI.GetGlobalSetting(ctx, "proxy.enabled")
	suite.NoError(err)
	suite.Equal(true, proxyEnabled)
}

// TestConcurrentOperations 测试并发操作
func (suite *E2EWorkflowTestSuite) TestConcurrentOperations() {
	toolName := "concurrent-tool"

	// 创建多个测试二进制文件
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"}
	binaries := make([]string, len(versions))

	for i, ver := range versions {
		binaries[i] = suite.createTestBinary(toolName, ver, "#!/bin/bash\necho 'concurrent-tool "+ver+"'")
	}

	// 并发注册版本
	done := make(chan bool, len(versions))
	errors := make(chan error, len(versions))

	for i, ver := range versions {
		go func(version, binary string) {
			err := suite.versionManager.RegisterVersion(toolName, version, binary)
			if err != nil {
				errors <- err
			}
			done <- true
		}(ver, binaries[i])
	}

	// 等待所有goroutine完成
	for i := 0; i < len(versions); i++ {
		select {
		case <-done:
			// 成功
		case err := <-errors:
			suite.Fail("Concurrent registration failed", err)
		case <-time.After(30 * time.Second):
			suite.Fail("Concurrent registration timeout")
		}
	}

	// 验证所有版本都已注册
	installedVersions, err := suite.versionManager.ListVersions(toolName)
	suite.NoError(err)
	suite.Len(installedVersions, len(versions))

	for _, ver := range versions {
		suite.Contains(installedVersions, ver)
	}
}

// TestErrorHandling 测试错误处理
func (suite *E2EWorkflowTestSuite) TestErrorHandling() {
	toolName := "error-tool"
	version := "1.0.0"

	// 测试注册不存在的二进制文件
	err := suite.versionManager.RegisterVersion(toolName, version, "/non/existent/binary")
	suite.Error(err)

	// 测试获取不存在的版本
	_, err = suite.versionManager.GetVersionPath(toolName, version)
	suite.Error(err)

	// 测试设置不存在的版本为当前版本
	err = suite.versionManager.SetGlobalVersion(toolName, version)
	suite.Error(err)

	// 测试移除不存在的版本
	err = suite.versionManager.RemoveVersion(toolName, version)
	suite.Error(err)

	// 测试无效的版本格式
	err = suite.versionManager.ValidateVersion("invalid.version.format.too.many.parts")
	suite.Error(err)

	// 测试获取不存在工具的版本
	versions, err := suite.versionManager.ListVersions("nonexistent-tool")
	suite.NoError(err) // 应该返回空列表而不是错误
	suite.Empty(versions)
}

// TestVersionComparison 测试版本比较
func (suite *E2EWorkflowTestSuite) TestVersionComparison() {
	toolName := "version-tool"
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"}

	// 注册多个版本
	for _, ver := range versions {
		binary := suite.createTestBinary(toolName, ver, "#!/bin/bash\necho '"+toolName+" "+ver+"'")
		err := suite.versionManager.RegisterVersion(toolName, ver, binary)
		suite.NoError(err)
	}

	// 获取最新版本
	latest, err := suite.versionManager.GetLatestVersion(toolName)
	suite.NoError(err)
	suite.Equal("2.1.0", latest)

	// 验证版本比较逻辑
	err = suite.versionManager.ValidateVersion("1.0.0")
	suite.NoError(err)

	err = suite.versionManager.ValidateVersion("v1.0.0")
	suite.NoError(err)

	err = suite.versionManager.ValidateVersion("1.0.0-alpha")
	suite.NoError(err)
}

// 辅助方法

// createTestBinary 创建测试二进制文件
func (suite *E2EWorkflowTestSuite) createTestBinary(toolName, version, content string) string {
	binaryPath := filepath.Join(suite.tmpDir, "test-binaries", toolName+"-"+version)
	err := os.MkdirAll(filepath.Dir(binaryPath), 0755)
	require.NoError(suite.T(), err)

	err = os.WriteFile(binaryPath, []byte(content), 0755)
	require.NoError(suite.T(), err)

	return binaryPath
}

// TestE2EWorkflowTestSuite 运行端到端工作流测试套件
func TestE2EWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(E2EWorkflowTestSuite))
}
