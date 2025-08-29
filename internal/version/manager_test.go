package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/pkg/types"
)

func TestVersionManager(t *testing.T) {
	// 创建内存文件系统用于测试
	fs := afero.NewMemMapFs()
	homeDir := "/home/test"

	// 创建测试环境
	configPaths := types.DefaultConfigPaths(homeDir)

	// 创建配置管理器
	configManager, err := config.NewManager(homeDir)
	require.NoError(t, err)

	// 创建存储管理器
	storageManager := storage.NewFilesystemManagerWithFs(fs, configPaths)

	// 创建版本管理器
	versionManager := NewManagerWithFs(storageManager, configManager, fs)

	t.Run("ValidateVersion", func(t *testing.T) {
		testCases := []struct {
			version string
			valid   bool
		}{
			{"1.0.0", true},
			{"v1.2.3", true},
			{"1.2.3-alpha", true},
			{"2.1", true},
			{"invalid", false},
			{"", false},
			{"v", false},
		}

		for _, tc := range testCases {
			t.Run(tc.version, func(t *testing.T) {
				err := versionManager.ValidateVersion(tc.version)
				if tc.valid {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
				}
			})
		}
	})

	t.Run("RegisterVersion", func(t *testing.T) {
		// 确保目录存在
		err := storageManager.EnsureDirectories()
		require.NoError(t, err)

		// 创建测试二进制文件
		testTool := "test-tool"
		testVersion := "1.0.0"
		sourcePath := "/tmp/test-binary"

		// 创建源文件
		err = afero.WriteFile(fs, sourcePath, []byte("#!/bin/bash\necho 'test'"), 0755)
		require.NoError(t, err)

		// 注册版本
		err = versionManager.RegisterVersion(testTool, testVersion, sourcePath)
		require.NoError(t, err)

		// 验证版本已安装
		assert.True(t, versionManager.IsVersionInstalled(testTool, testVersion))

		// 验证二进制文件存在
		binaryPath := storageManager.GetBinaryPath(testTool, testVersion)
		exists, err := afero.Exists(fs, binaryPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// 验证元数据存在
		metadata, err := versionManager.GetVersionMetadata(testTool, testVersion)
		require.NoError(t, err)
		assert.Equal(t, testTool, metadata.ToolName)
		assert.Equal(t, testVersion, metadata.Version)
		assert.Equal(t, "manual", metadata.InstallType)
	})

	t.Run("ListVersions", func(t *testing.T) {
		// 注册多个版本
		testTool := "multi-version-tool"
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}

		for _, v := range versions {
			sourcePath := "/tmp/binary-" + v
			err := afero.WriteFile(fs, sourcePath, []byte("test"), 0755)
			require.NoError(t, err)

			err = versionManager.RegisterVersion(testTool, v, sourcePath)
			require.NoError(t, err)
		}

		// 获取版本列表
		installedVersions, err := versionManager.ListVersions(testTool)
		require.NoError(t, err)

		// 验证所有版本都被列出
		assert.Len(t, installedVersions, len(versions))
		for _, v := range versions {
			assert.Contains(t, installedVersions, v)
		}
	})

	t.Run("RemoveVersion", func(t *testing.T) {
		testTool := "removable-tool"
		testVersion := "1.0.0"
		sourcePath := "/tmp/removable-binary"

		// 创建并注册版本
		err := afero.WriteFile(fs, sourcePath, []byte("test"), 0755)
		require.NoError(t, err)

		err = versionManager.RegisterVersion(testTool, testVersion, sourcePath)
		require.NoError(t, err)

		// 验证版本存在
		assert.True(t, versionManager.IsVersionInstalled(testTool, testVersion))

		// 移除版本
		err = versionManager.RemoveVersion(testTool, testVersion)
		require.NoError(t, err)

		// 验证版本已被移除
		assert.False(t, versionManager.IsVersionInstalled(testTool, testVersion))
	})

	t.Run("GetLatestVersion", func(t *testing.T) {
		testTool := "latest-test-tool"
		versions := []string{"1.0.0", "1.2.0", "2.0.0", "1.1.0"}

		// 注册多个版本
		for _, v := range versions {
			sourcePath := "/tmp/binary-latest-" + v
			err := afero.WriteFile(fs, sourcePath, []byte("test"), 0755)
			require.NoError(t, err)

			err = versionManager.RegisterVersion(testTool, v, sourcePath)
			require.NoError(t, err)
		}

		// 获取最新版本
		latest, err := versionManager.GetLatestVersion(testTool)
		require.NoError(t, err)

		// 应该返回语义版本最高的版本
		assert.Equal(t, "2.0.0", latest)
	})

	t.Run("GetVersionPath", func(t *testing.T) {
		testTool := "path-tool"
		testVersion := "1.0.0"
		sourcePath := "/tmp/path-binary"

		// 注册版本
		err := afero.WriteFile(fs, sourcePath, []byte("test"), 0755)
		require.NoError(t, err)

		err = versionManager.RegisterVersion(testTool, testVersion, sourcePath)
		require.NoError(t, err)

		// 获取版本路径
		versionPath, err := versionManager.GetVersionPath(testTool, testVersion)
		require.NoError(t, err)

		expectedPath := storageManager.GetToolVersionPath(testTool, testVersion)
		assert.Equal(t, expectedPath, versionPath)

		// 测试不存在的版本
		_, err = versionManager.GetVersionPath(testTool, "999.0.0")
		assert.Error(t, err)
	})

	t.Run("ListAllTools", func(t *testing.T) {
		// 这个测试应该列出之前测试中注册的所有工具
		tools, err := versionManager.ListAllTools()
		require.NoError(t, err)

		// 验证至少包含我们注册的工具
		assert.GreaterOrEqual(t, len(tools), 1)
	})
}

func TestVersionManagerIntegration(t *testing.T) {
	// 创建临时目录进行集成测试
	tmpDir, err := os.MkdirTemp("", "vman-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建真实的管理器
	configPaths := types.DefaultConfigPaths(tmpDir)

	// 创建配置管理器
	configManager, err := config.NewManager(tmpDir)
	require.NoError(t, err)

	// 创建存储管理器
	storageManager := storage.NewFilesystemManager(configPaths)

	// 创建版本管理器
	versionManager := NewManager(storageManager, configManager)

	// 初始化
	err = storageManager.EnsureDirectories()
	require.NoError(t, err)

	err = configManager.Initialize()
	require.NoError(t, err)

	t.Run("FullWorkflow", func(t *testing.T) {
		testTool := "kubectl"
		testVersion := "1.29.0"

		// 创建测试二进制文件
		testBinary := filepath.Join(tmpDir, "kubectl-test")
		content := []byte("#!/bin/bash\necho 'kubectl version'")
		err := os.WriteFile(testBinary, content, 0755)
		require.NoError(t, err)

		// 1. 注册版本
		err = versionManager.RegisterVersion(testTool, testVersion, testBinary)
		require.NoError(t, err)

		// 2. 验证版本已安装
		assert.True(t, versionManager.IsVersionInstalled(testTool, testVersion))

		// 3. 设置全局版本
		err = versionManager.SetGlobalVersion(testTool, testVersion)
		require.NoError(t, err)

		// 4. 获取当前版本
		currentVersion, err := versionManager.GetCurrentVersion(testTool)
		require.NoError(t, err)
		assert.Equal(t, testVersion, currentVersion)

		// 5. 获取版本元数据
		metadata, err := versionManager.GetVersionMetadata(testTool, testVersion)
		require.NoError(t, err)
		assert.Equal(t, testTool, metadata.ToolName)
		assert.Equal(t, testVersion, metadata.Version)

		// 6. 列出版本
		versions, err := versionManager.ListVersions(testTool)
		require.NoError(t, err)
		assert.Contains(t, versions, testVersion)

		// 7. 测试项目级版本设置
		projectDir := filepath.Join(tmpDir, "test-project")
		err = os.MkdirAll(projectDir, 0755)
		require.NoError(t, err)

		// 注册另一个版本用于项目级测试
		testVersion2 := "1.28.0"
		testBinary2 := filepath.Join(tmpDir, "kubectl-test-2")
		err = os.WriteFile(testBinary2, content, 0755)
		require.NoError(t, err)

		err = versionManager.RegisterVersion(testTool, testVersion2, testBinary2)
		require.NoError(t, err)

		// 设置项目级版本
		err = versionManager.SetProjectVersion(testTool, testVersion2, projectDir)
		require.NoError(t, err)

		// 在项目目录中获取有效版本应该返回项目级版本
		effectiveVersion, err := versionManager.GetEffectiveVersion(testTool, projectDir)
		require.NoError(t, err)
		assert.Equal(t, testVersion2, effectiveVersion)

		// 8. 测试移除版本（不能移除当前使用的版本）
		err = versionManager.RemoveVersion(testTool, testVersion)
		assert.Error(t, err, "should not be able to remove currently active version")

		// 移除非当前版本应该成功
		err = versionManager.RemoveVersion(testTool, testVersion2)
		require.NoError(t, err)

		// 验证版本已被移除
		assert.False(t, versionManager.IsVersionInstalled(testTool, testVersion2))
	})
}

// 辅助函数创建测试logger
func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}
