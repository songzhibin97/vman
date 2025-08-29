package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/pkg/types"
)

// mockConfigManager 模拟配置管理器，简化测试
type mockConfigManager struct {
	globalConfig *types.GlobalConfig
	homeDir      string
}

func newMockConfigManager(homeDir string) *mockConfigManager {
	return &mockConfigManager{
		globalConfig: types.GetDefaultGlobalConfig(),
		homeDir:      homeDir,
	}
}

func (m *mockConfigManager) LoadGlobal() (*types.GlobalConfig, error) {
	return m.globalConfig, nil
}

func (m *mockConfigManager) LoadProject(path string) (*types.ProjectConfig, error) {
	return types.GetDefaultProjectConfig(), nil
}

func (m *mockConfigManager) LoadToolConfig(toolName string) (*types.ToolMetadata, error) {
	return &types.ToolMetadata{Name: toolName}, nil
}

func (m *mockConfigManager) SaveGlobal(config *types.GlobalConfig) error {
	m.globalConfig = config
	return nil
}

func (m *mockConfigManager) SaveProject(path string, config *types.ProjectConfig) error {
	return nil
}

func (m *mockConfigManager) GetEffectiveVersion(toolName, projectPath string) (string, error) {
	if toolInfo, exists := m.globalConfig.Tools[toolName]; exists {
		return toolInfo.CurrentVersion, nil
	}
	if version, exists := m.globalConfig.GlobalVersions[toolName]; exists {
		return version, nil
	}
	return "", nil
}

func (m *mockConfigManager) GetConfigDir() string {
	return filepath.Join(m.homeDir, ".vman")
}

func (m *mockConfigManager) GetProjectConfigPath(projectPath string) string {
	return filepath.Join(projectPath, ".vman.yaml")
}

func (m *mockConfigManager) Initialize() error {
	return nil
}

func (m *mockConfigManager) Validate() error {
	return nil
}

func (m *mockConfigManager) ListTools() ([]string, error) {
	var tools []string
	for tool := range m.globalConfig.Tools {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (m *mockConfigManager) IsToolInstalled(toolName, version string) bool {
	return true
}

func (m *mockConfigManager) SetToolVersion(toolName, version string, global bool, projectPath string) error {
	if global {
		if m.globalConfig.GlobalVersions == nil {
			m.globalConfig.GlobalVersions = make(map[string]string)
		}
		m.globalConfig.GlobalVersions[toolName] = version

		if m.globalConfig.Tools == nil {
			m.globalConfig.Tools = make(map[string]types.ToolInfo)
		}
		toolInfo := m.globalConfig.Tools[toolName]
		toolInfo.CurrentVersion = version
		m.globalConfig.Tools[toolName] = toolInfo
	}
	return nil
}

func (m *mockConfigManager) RemoveToolVersion(toolName, version string) error {
	return nil
}

func (m *mockConfigManager) GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error) {
	return &types.EffectiveConfig{
		Global:           m.globalConfig,
		Project:          types.GetDefaultProjectConfig(),
		ResolvedVersions: make(map[string]string),
		ConfigSource:     make(map[string]string),
	}, nil
}

func TestVersionManagerSimplified(t *testing.T) {
	// 创建内存文件系统用于测试
	fs := afero.NewMemMapFs()
	homeDir := "/home/test"

	// 创建测试环境
	configPaths := types.DefaultConfigPaths(homeDir)

	// 创建模拟配置管理器
	configManager := newMockConfigManager(homeDir)

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

	t.Run("SetAndGetGlobalVersion", func(t *testing.T) {
		testTool := "version-tool"
		testVersion := "1.0.0"
		sourcePath := "/tmp/version-binary"

		// 创建并注册版本
		err := afero.WriteFile(fs, sourcePath, []byte("test"), 0755)
		require.NoError(t, err)

		err = versionManager.RegisterVersion(testTool, testVersion, sourcePath)
		require.NoError(t, err)

		// 设置全局版本
		err = versionManager.SetGlobalVersion(testTool, testVersion)
		require.NoError(t, err)

		// 获取有效版本
		effectiveVersion, err := versionManager.GetEffectiveVersion(testTool, "")
		require.NoError(t, err)
		assert.Equal(t, testVersion, effectiveVersion)
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
}

func TestVersionManagerIntegrationReal(t *testing.T) {
	// 创建临时目录进行真实集成测试
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

	t.Run("BasicWorkflow", func(t *testing.T) {
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
	})
}
