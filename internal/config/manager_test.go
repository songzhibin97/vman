package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/vman/pkg/types"
)

func TestDefaultManager(t *testing.T) {
	// 创建内存文件系统用于测试
	fs := afero.NewMemMapFs()
	homeDir := "/home/test"

	// 创建测试用的管理器
	manager := &DefaultManager{
		fs:     fs,
		paths:  types.DefaultConfigPaths(homeDir),
		logger: testLogger(),
	}

	t.Run("Initialize", func(t *testing.T) {
		err := manager.Initialize()
		require.NoError(t, err)

		// 验证目录是否创建
		dirs := []string{
			manager.paths.ConfigDir,
			manager.paths.ToolsDir,
			manager.paths.BinDir,
			manager.paths.ShimsDir,
			manager.paths.VersionsDir,
			manager.paths.LogsDir,
			manager.paths.CacheDir,
			manager.paths.TempDir,
		}

		for _, dir := range dirs {
			exists, err := afero.DirExists(fs, dir)
			require.NoError(t, err)
			assert.True(t, exists, "Directory %s should exist", dir)
		}

		// 验证默认全局配置文件是否创建
		exists, err := afero.Exists(fs, manager.paths.GlobalConfigFile)
		require.NoError(t, err)
		assert.True(t, exists, "Global config file should exist")
	})

	t.Run("LoadGlobal", func(t *testing.T) {
		// 先初始化
		err := manager.Initialize()
		require.NoError(t, err)

		// 加载全局配置
		config, err := manager.LoadGlobal()
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "1.0", config.Version)
		assert.NotNil(t, config.Settings)
		assert.Equal(t, 300*time.Second, config.Settings.Download.Timeout)
		assert.Equal(t, 3, config.Settings.Download.Retries)
		assert.Equal(t, 2, config.Settings.Download.ConcurrentDownloads)
	})

	t.Run("SaveGlobal", func(t *testing.T) {
		// 先初始化
		err := manager.Initialize()
		require.NoError(t, err)

		// 创建测试配置
		config := types.GetDefaultGlobalConfig()
		config.GlobalVersions["test-tool"] = "1.0.0"

		// 保存配置
		err = manager.SaveGlobal(config)
		require.NoError(t, err)

		// 重新加载并验证
		loadedConfig, err := manager.LoadGlobal()
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", loadedConfig.GlobalVersions["test-tool"])
	})

	t.Run("LoadProject", func(t *testing.T) {
		projectPath := "/project/test"

		// 测试不存在项目配置文件的情况
		config, err := manager.LoadProject(projectPath)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "1.0", config.Version)
		assert.NotNil(t, config.Tools)

		// 创建项目配置文件
		projectConfig := types.GetDefaultProjectConfig()
		projectConfig.Tools["kubectl"] = "1.28.0"

		// 创建目录
		err = fs.MkdirAll(projectPath, 0755)
		require.NoError(t, err)

		// 保存项目配置
		err = manager.SaveProject(projectPath, projectConfig)
		require.NoError(t, err)

		// 重新加载并验证
		loadedConfig, err := manager.LoadProject(projectPath)
		require.NoError(t, err)
		assert.Equal(t, "1.28.0", loadedConfig.Tools["kubectl"])
	})

	t.Run("GetEffectiveVersion", func(t *testing.T) {
		// 先初始化
		err := manager.Initialize()
		require.NoError(t, err)

		projectPath := "/project/test-new"

		// 设置全局版本
		globalConfig, err := manager.LoadGlobal()
		require.NoError(t, err)
		globalConfig.GlobalVersions["kubectl"] = "1.29.0"
		err = manager.SaveGlobal(globalConfig)
		require.NoError(t, err)

		// 测试获取全局版本（项目配置不存在时）
		version, err := manager.GetEffectiveVersion("kubectl", projectPath)
		require.NoError(t, err)
		assert.Equal(t, "1.29.0", version)

		// 设置项目版本（应该覆盖全局版本）
		projectConfig := types.GetDefaultProjectConfig()
		projectConfig.Tools["kubectl"] = "1.28.0"
		err = manager.SaveProject(projectPath, projectConfig)
		require.NoError(t, err)

		// 测试获取项目版本（项目版本应该覆盖全局版本）
		version, err = manager.GetEffectiveVersion("kubectl", projectPath)
		require.NoError(t, err)
		assert.Equal(t, "1.28.0", version)
	})

	t.Run("SetToolVersion", func(t *testing.T) {
		// 先初始化
		err := manager.Initialize()
		require.NoError(t, err)

		projectPath := "/project/test"

		// 设置全局工具版本
		err = manager.SetToolVersion("terraform", "1.6.0", true, "")
		require.NoError(t, err)

		// 验证全局版本设置
		globalConfig, err := manager.LoadGlobal()
		require.NoError(t, err)
		assert.Equal(t, "1.6.0", globalConfig.GlobalVersions["terraform"])
		assert.Equal(t, "1.6.0", globalConfig.Tools["terraform"].CurrentVersion)

		// 设置项目工具版本
		err = manager.SetToolVersion("terraform", "1.5.0", false, projectPath)
		require.NoError(t, err)

		// 验证项目版本设置
		projectConfig, err := manager.LoadProject(projectPath)
		require.NoError(t, err)
		assert.Equal(t, "1.5.0", projectConfig.Tools["terraform"])
	})

	t.Run("ListTools", func(t *testing.T) {
		// 先初始化
		err := manager.Initialize()
		require.NoError(t, err)

		// 创建一些工具配置文件
		toolConfigs := []string{"kubectl.toml", "terraform.toml", "sqlc.toml"}
		for _, toolConfig := range toolConfigs {
			toolPath := filepath.Join(manager.paths.ToolsDir, toolConfig)
			err = afero.WriteFile(fs, toolPath, []byte("name = \"test\""), 0644)
			require.NoError(t, err)
		}

		// 列出工具
		tools, err := manager.ListTools()
		require.NoError(t, err)
		assert.Len(t, tools, 3)
		assert.Contains(t, tools, "kubectl")
		assert.Contains(t, tools, "terraform")
		assert.Contains(t, tools, "sqlc")
	})
}

func TestValidator(t *testing.T) {
	validator := NewValidator()

	t.Run("ValidateGlobalConfig", func(t *testing.T) {
		// 测试有效配置
		validConfig := types.GetDefaultGlobalConfig()
		err := validator.ValidateGlobalConfig(validConfig)
		assert.NoError(t, err)

		// 测试nil配置
		err = validator.ValidateGlobalConfig(nil)
		assert.Error(t, err)

		// 测试无效版本
		invalidConfig := types.GetDefaultGlobalConfig()
		invalidConfig.Version = ""
		err = validator.ValidateGlobalConfig(invalidConfig)
		assert.Error(t, err)

		// 测试无效重试次数
		invalidConfig = types.GetDefaultGlobalConfig()
		invalidConfig.Settings.Download.Retries = -1
		err = validator.ValidateGlobalConfig(invalidConfig)
		assert.Error(t, err)
	})

	t.Run("ValidateProjectConfig", func(t *testing.T) {
		// 测试有效配置
		validConfig := types.GetDefaultProjectConfig()
		err := validator.ValidateProjectConfig(validConfig)
		assert.NoError(t, err)

		// 测试nil配置
		err = validator.ValidateProjectConfig(nil)
		assert.Error(t, err)

		// 测试无效版本
		invalidConfig := types.GetDefaultProjectConfig()
		invalidConfig.Version = ""
		err = validator.ValidateProjectConfig(invalidConfig)
		assert.Error(t, err)
	})

	t.Run("ValidateToolName", func(t *testing.T) {
		// 测试有效工具名称
		validNames := []string{"kubectl", "terraform", "test-tool", "test_tool", "tool123"}
		for _, name := range validNames {
			err := validator.ValidateToolName(name)
			assert.NoError(t, err, "Tool name %s should be valid", name)
		}

		// 测试无效工具名称
		invalidNames := []string{"", "tool with spaces", "tool@invalid", repeat("a", 60)}
		for _, name := range invalidNames {
			err := validator.ValidateToolName(name)
			assert.Error(t, err, "Tool name %s should be invalid", name)
		}
	})

	t.Run("ValidateVersion", func(t *testing.T) {
		// 测试有效版本
		validVersions := []string{"1.0.0", "v1.0.0", "1.0.0-alpha", "latest", "stable"}
		for _, version := range validVersions {
			err := validator.ValidateVersion(version)
			assert.NoError(t, err, "Version %s should be valid", version)
		}

		// 测试无效版本
		invalidVersions := []string{"", "invalid", "1.0.0.0.0"}
		for _, version := range invalidVersions {
			err := validator.ValidateVersion(version)
			assert.Error(t, err, "Version %s should be invalid", version)
		}
	})
}

func TestMerger(t *testing.T) {
	merger := NewMerger()

	t.Run("MergeConfigs", func(t *testing.T) {
		// 创建测试配置
		globalConfig := types.GetDefaultGlobalConfig()
		globalConfig.GlobalVersions["kubectl"] = "1.29.0"
		globalConfig.GlobalVersions["terraform"] = "1.6.0"

		projectConfig := types.GetDefaultProjectConfig()
		projectConfig.Tools["kubectl"] = "1.28.0" // 覆盖全局版本
		projectConfig.Tools["sqlc"] = "1.20.0"    // 新增工具

		// 测试覆盖策略
		effective, err := merger.MergeConfigs(globalConfig, projectConfig, types.OverrideStrategy)
		require.NoError(t, err)
		assert.NotNil(t, effective)

		// 验证合并结果
		assert.Equal(t, "1.28.0", effective.ResolvedVersions["kubectl"])  // 项目版本覆盖全局版本
		assert.Equal(t, "1.6.0", effective.ResolvedVersions["terraform"]) // 全局版本
		assert.Equal(t, "1.20.0", effective.ResolvedVersions["sqlc"])     // 项目版本

		// 验证版本来源
		assert.Equal(t, "project", effective.ConfigSource["kubectl"])
		assert.Equal(t, "global", effective.ConfigSource["terraform"])
		assert.Equal(t, "project", effective.ConfigSource["sqlc"])
	})

	t.Run("GetVersionSource", func(t *testing.T) {
		globalConfig := types.GetDefaultGlobalConfig()
		globalConfig.GlobalVersions["kubectl"] = "1.29.0"

		projectConfig := types.GetDefaultProjectConfig()
		projectConfig.Tools["terraform"] = "1.6.0"

		// 测试项目版本来源
		version, source := merger.GetVersionSource("terraform", globalConfig, projectConfig)
		assert.Equal(t, "1.6.0", version)
		assert.Equal(t, "project", source)

		// 测试全局版本来源
		version, source = merger.GetVersionSource("kubectl", globalConfig, projectConfig)
		assert.Equal(t, "1.29.0", version)
		assert.Equal(t, "global", source)

		// 测试不存在的工具
		version, source = merger.GetVersionSource("nonexistent", globalConfig, projectConfig)
		assert.Equal(t, "", version)
		assert.Equal(t, "none", source)
	})
}

func TestAPI(t *testing.T) {
	// 由于API依赖文件系统，我们需要创建临时目录
	tmpDir, err := os.MkdirTemp("", "vman-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// 创建API实例
	api, err := NewAPI(tmpDir)
	require.NoError(t, err)

	t.Run("Init", func(t *testing.T) {
		ctx := context.Background()
		err := api.Init(ctx)
		require.NoError(t, err)

		// 验证配置目录是否创建
		paths, err := api.GetConfigPaths(ctx)
		require.NoError(t, err)
		_, err = os.Stat(paths.ConfigDir)
		assert.NoError(t, err)
	})

	t.Run("SetAndGetGlobalToolVersion", func(t *testing.T) {
		ctx := context.Background()
		// 先初始化
		err := api.Init(ctx)
		require.NoError(t, err)

		// 设置全局工具版本
		err = api.SetToolVersion(ctx, "kubectl", "1.29.0", true, "")
		require.NoError(t, err)

		// 获取版本
		version, err := api.GetEffectiveVersion(ctx, "kubectl", "")
		require.NoError(t, err)
		assert.Equal(t, "1.29.0", version)
	})

	t.Run("SetAndGetProjectToolVersion", func(t *testing.T) {
		ctx := context.Background()
		// 先初始化
		err := api.Init(ctx)
		require.NoError(t, err)

		projectPath := filepath.Join(tmpDir, "test-project")
		err = os.MkdirAll(projectPath, 0755)
		require.NoError(t, err)

		// 设置项目工具版本
		err = api.SetToolVersion(ctx, "terraform", "1.6.0", false, projectPath)
		require.NoError(t, err)

		// 获取版本
		version, err := api.GetEffectiveVersion(ctx, "terraform", projectPath)
		require.NoError(t, err)
		assert.Equal(t, "1.6.0", version)
	})

	t.Run("GetEffectiveConfig", func(t *testing.T) {
		ctx := context.Background()
		// 先初始化
		err := api.Init(ctx)
		require.NoError(t, err)

		// 设置一些配置
		err = api.SetToolVersion(ctx, "kubectl", "1.29.0", true, "")
		require.NoError(t, err)

		// 获取有效配置
		effective, err := api.GetEffectiveConfig(ctx, "")
		require.NoError(t, err)
		assert.NotNil(t, effective)
		assert.NotNil(t, effective.Global)
		assert.NotNil(t, effective.Project)
		assert.Equal(t, "1.29.0", effective.ResolvedVersions["kubectl"])
	})
}

// 辅助函数

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // 测试时只显示错误日志
	return logger
}

// String repeat helper for old Go versions
func repeat(s string, count int) string {
	if count <= 0 {
		return ""
	}
	result := make([]string, count)
	for i := range result {
		result[i] = s
	}
	return strings.Join(result, "")
}
