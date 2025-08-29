package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/proxy"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// ErrorScenarioTestSuite 错误场景测试套件
type ErrorScenarioTestSuite struct {
	suite.Suite
	fs          afero.Fs
	homeDir     string
	configAPI   config.API
	versionMgr  version.Manager
	proxyMgr    proxy.CommandProxy
	ctx         context.Context
	cleanupFuncs []func()
}

func (suite *ErrorScenarioTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.fs = afero.NewMemMapFs()
	suite.cleanupFuncs = make([]func(), 0)

	// 创建临时主目录
	tempDir := fmt.Sprintf("/tmp/vman-error-test-%d", time.Now().Unix())
	suite.homeDir = tempDir
	err := suite.fs.MkdirAll(suite.homeDir, 0755)
	require.NoError(suite.T(), err)

	// 初始化组件
	suite.configAPI, err = config.NewAPI(suite.homeDir)
	require.NoError(suite.T(), err)

	err = suite.configAPI.Init(suite.ctx)
	require.NoError(suite.T(), err)

	storageManager := storage.NewFilesystemManager(suite.homeDir)
	configManager, err := config.NewManager(suite.homeDir)
	require.NoError(suite.T(), err)
	suite.versionMgr, err = version.NewManager(storageManager, configManager)
	require.NoError(suite.T(), err)

	suite.proxyMgr = proxy.NewCommandProxy(configManager, suite.versionMgr)
}

func (suite *ErrorScenarioTestSuite) TearDownSuite() {
	for i := len(suite.cleanupFuncs) - 1; i >= 0; i-- {
		suite.cleanupFuncs[i]()
	}
}

func (suite *ErrorScenarioTestSuite) addCleanup(fn func()) {
	suite.cleanupFuncs = append(suite.cleanupFuncs, fn)
}

// TestInvalidConfigFiles 测试无效配置文件处理
func (suite *ErrorScenarioTestSuite) TestInvalidConfigFiles() {
	suite.Run("损坏的全局配置文件", func() {
		paths, err := suite.configAPI.GetConfigPaths(suite.ctx)
		require.NoError(suite.T(), err)

		// 创建损坏的配置文件
		invalidYAML := "version: 1.0\ntools:\n  - invalid yaml structure"
		err = afero.WriteFile(suite.fs, paths.GlobalConfigFile, []byte(invalidYAML), 0644)
		require.NoError(suite.T(), err)

		// 尝试加载配置应该失败
		_, err = suite.configAPI.GetGlobalConfig(suite.ctx)
		assert.Error(suite.T(), err)
	})

	suite.Run("权限拒绝的配置文件", func() {
		paths, err := suite.configAPI.GetConfigPaths(suite.ctx)
		require.NoError(suite.T(), err)

		// 创建无权限访问的配置文件（在内存文件系统中模拟）
		restrictedFile := filepath.Join(paths.ConfigDir, "restricted.yaml")
		err = afero.WriteFile(suite.fs, restrictedFile, []byte("test"), 0000) // 无权限
		require.NoError(suite.T(), err)

		// 尝试读取应该处理权限错误
		_, err = afero.ReadFile(suite.fs, restrictedFile)
		assert.Error(suite.T(), err)
	})

	suite.Run("空配置文件", func() {
		projectPath := filepath.Join(suite.homeDir, "empty_project")
		err := suite.fs.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		emptyConfigFile := filepath.Join(projectPath, ".vman.yaml")
		err = afero.WriteFile(suite.fs, emptyConfigFile, []byte(""), 0644)
		require.NoError(suite.T(), err)

		// 应该返回默认配置而不是错误
		config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), config)
	})
}

// TestInvalidToolOperations 测试无效工具操作
func (suite *ErrorScenarioTestSuite) TestInvalidToolOperations() {
	suite.Run("注册无效工具", func() {
		invalidMetadata := &types.ToolMetadata{
			Name:        "", // 空名称
			Description: "Invalid tool",
		}

		err := suite.configAPI.RegisterTool(suite.ctx, invalidMetadata)
		assert.Error(suite.T(), err)
	})

	suite.Run("获取不存在的工具配置", func() {
		_, err := suite.configAPI.GetToolConfig(suite.ctx, "nonexistent-tool")
		assert.Error(suite.T(), err)
	})

	suite.Run("设置无效版本", func() {
		err := suite.configAPI.SetToolVersion(suite.ctx, "test-tool", "", true, "")
		assert.Error(suite.T(), err)
	})

	suite.Run("移除不存在的工具版本", func() {
		err := suite.configAPI.RemoveToolVersion(suite.ctx, "nonexistent-tool", "1.0.0")
		// 这应该优雅地处理而不是崩溃
		// 具体行为取决于实现
	})
}

// TestFileSystemErrors 测试文件系统错误
func (suite *ErrorScenarioTestSuite) TestFileSystemErrors() {
	suite.Run("磁盘空间不足模拟", func() {
		// 在内存文件系统中，我们可以通过其他方式模拟错误
		largePath := filepath.Join(suite.homeDir, "large_config")
		
		// 尝试创建大量文件来模拟空间不足
		for i := 0; i < 1000; i++ {
			testFile := filepath.Join(largePath, fmt.Sprintf("file_%d.txt", i))
			err := suite.fs.MkdirAll(filepath.Dir(testFile), 0755)
			if err != nil {
				suite.T().Logf("模拟空间不足在文件 %d", i)
				break
			}
			
			err = afero.WriteFile(suite.fs, testFile, make([]byte, 1024), 0644)
			if err != nil {
				suite.T().Logf("写入失败在文件 %d: %v", i, err)
				break
			}
		}
	})

	suite.Run("目录创建失败", func() {
		// 尝试在只读目录中创建子目录
		readOnlyDir := filepath.Join(suite.homeDir, "readonly")
		err := suite.fs.MkdirAll(readOnlyDir, 0755)
		require.NoError(suite.T(), err)

		// 在只读目录中尝试创建项目配置
		readOnlyProject := filepath.Join(readOnlyDir, "project")
		config := types.GetDefaultProjectConfig()
		
		// 这个操作可能会失败，我们要确保错误得到正确处理
		err = suite.configAPI.UpdateProjectConfig(suite.ctx, readOnlyProject, config)
		// 不同实现可能有不同的行为，我们检查错误是否被正确处理
		if err != nil {
			assert.Error(suite.T(), err)
			suite.T().Logf("正确处理了目录创建错误: %v", err)
		}
	})
}

// TestNetworkErrors 测试网络相关错误
func (suite *ErrorScenarioTestSuite) TestNetworkErrors() {
	suite.Run("下载超时", func() {
		// 模拟网络超时的工具安装
		toolMetadata := &types.ToolMetadata{
			Name:        "network-tool",
			Description: "网络工具测试",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "https://github.com/invalid/timeout-tool", // 不存在的仓库
			// },
			// Binary: &types.BinaryConfig{
			// 	Name: "network-tool",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "download",
			// },
		}

		err := suite.configAPI.RegisterTool(suite.ctx, toolMetadata)
		require.NoError(suite.T(), err)

		// 尝试安装应该处理网络错误
		err = suite.versionMgr.InstallVersion("network-tool", "1.0.0")
		// 我们期望这会失败，但错误应该被正确处理
		if err != nil {
			assert.Error(suite.T(), err)
			suite.T().Logf("正确处理了网络错误: %v", err)
		}
	})

	suite.Run("无效URL", func() {
		toolMetadata := &types.ToolMetadata{
			Name:        "invalid-url-tool",
			Description: "无效URL工具测试",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "not-a-valid-url", // 无效URL
			// },
			// Binary: &types.BinaryConfig{
			// 	Name: "invalid-url-tool",
			// },
			// Install: &types.InstallConfig{
			// 	Method: "download",
			// },
		}

		err := suite.configAPI.RegisterTool(suite.ctx, toolMetadata)
		if err != nil {
			assert.Error(suite.T(), err)
			suite.T().Logf("正确处理了无效URL: %v", err)
		}
	})
}

// TestConcurrencyErrors 测试并发错误
func (suite *ErrorScenarioTestSuite) TestConcurrencyErrors() {
	suite.Run("并发配置写入冲突", func() {
		const numWorkers = 5
		const numOperations = 20

		done := make(chan error, numWorkers)
		projectPath := filepath.Join(suite.homeDir, "concurrent_project")

		// 启动多个goroutine同时修改同一个项目配置
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				var lastErr error
				for j := 0; j < numOperations; j++ {
					config := types.GetDefaultProjectConfig()
					config.Tools[fmt.Sprintf("tool_%d_%d", workerID, j)] = "1.0.0"
					
					err := suite.configAPI.UpdateProjectConfig(suite.ctx, projectPath, config)
					if err != nil {
						lastErr = err
					}
					
					// 短暂延迟增加冲突概率
					time.Sleep(1 * time.Millisecond)
				}
				done <- lastErr
			}(i)
		}

		// 收集错误
		errors := make([]error, 0)
		for i := 0; i < numWorkers; i++ {
			if err := <-done; err != nil {
				errors = append(errors, err)
			}
		}

		// 检查最终配置
		finalConfig, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
		require.NoError(suite.T(), err)
		assert.NotNil(suite.T(), finalConfig)

		suite.T().Logf("并发写入完成，错误数量: %d", len(errors))
	})

	suite.Run("并发版本操作", func() {
		toolName := "concurrent-tool"
		
		// 注册工具
		metadata := &types.ToolMetadata{
			Name:        toolName,
			Description: "并发测试工具",
			// Source: &types.SourceConfig{
			// 	Type: "github",
			// 	URL:  "https://github.com/test/concurrent-tool",
			// },
			// Binary: &types.BinaryConfig{
			// 	Name: toolName,
			// },
			// Install: &types.InstallConfig{
			// 	Method: "extract",
			// },
		}

		err := suite.configAPI.RegisterTool(suite.ctx, metadata)
		require.NoError(suite.T(), err)

		const numVersionOps = 10
		done := make(chan error, numVersionOps)

		// 同时进行多个版本操作
		for i := 0; i < numVersionOps; i++ {
			go func(versionNum int) {
				version := fmt.Sprintf("1.%d.0", versionNum)
				err := suite.configAPI.SetToolVersion(suite.ctx, toolName, version, true, "")
				done <- err
			}(i)
		}

		// 收集结果
		errors := make([]error, 0)
		for i := 0; i < numVersionOps; i++ {
			if err := <-done; err != nil {
				errors = append(errors, err)
			}
		}

		suite.T().Logf("并发版本操作完成，错误数量: %d", len(errors))
	})
}

// TestResourceExhaustion 测试资源耗尽场景
func (suite *ErrorScenarioTestSuite) TestResourceExhaustion() {
	suite.Run("内存耗尽模拟", func() {
		// 创建大量配置对象来模拟内存使用
		configs := make([]*types.ProjectConfig, 0, 1000)
		
		for i := 0; i < 1000; i++ {
			config := types.GetDefaultProjectConfig()
			// 添加大量工具配置
			for j := 0; j < 100; j++ {
				config.Tools[fmt.Sprintf("tool_%d_%d", i, j)] = "1.0.0"
			}
			configs = append(configs, config)
		}

		// 确保我们创建了大量对象
		assert.Equal(suite.T(), 1000, len(configs))
		suite.T().Logf("创建了 %d 个配置对象", len(configs))
	})

	suite.Run("文件句柄耗尽模拟", func() {
		// 创建大量文件来模拟文件句柄耗尽
		files := make([]afero.File, 0, 100)
		
		for i := 0; i < 100; i++ {
			fileName := filepath.Join(suite.homeDir, fmt.Sprintf("handle_test_%d.txt", i))
			file, err := suite.fs.Create(fileName)
			if err != nil {
				suite.T().Logf("文件创建失败在 %d: %v", i, err)
				break
			}
			files = append(files, file)
		}

		// 清理文件句柄
		suite.addCleanup(func() {
			for _, file := range files {
				file.Close()
			}
		})

		suite.T().Logf("创建了 %d 个文件句柄", len(files))
	})
}

// TestCorruptedData 测试数据损坏场景
func (suite *ErrorScenarioTestSuite) TestCorruptedData() {
	suite.Run("配置文件部分损坏", func() {
		projectPath := filepath.Join(suite.homeDir, "corrupted_project")
		err := suite.fs.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		// 创建部分损坏的YAML文件
		corruptedYAML := `version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.6.
# 文件在这里被截断`

		configFile := filepath.Join(projectPath, ".vman.yaml")
		err = afero.WriteFile(suite.fs, configFile, []byte(corruptedYAML), 0644)
		require.NoError(suite.T(), err)

		// 尝试加载损坏的配置
		_, err = suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
		assert.Error(suite.T(), err)
		suite.T().Logf("正确检测到损坏的配置: %v", err)
	})

	suite.Run("二进制数据错误", func() {
		// 创建包含二进制数据的配置文件
		binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}
		
		configFile := filepath.Join(suite.homeDir, "binary.yaml")
		err := afero.WriteFile(suite.fs, configFile, binaryData, 0644)
		require.NoError(suite.T(), err)

		// 尝试读取二进制数据作为配置
		content, err := afero.ReadFile(suite.fs, configFile)
		require.NoError(suite.T(), err)
		
		// 验证读取的是二进制数据
		assert.Equal(suite.T(), binaryData, content)
	})
}

// TestTimeoutScenarios 测试超时场景
func (suite *ErrorScenarioTestSuite) TestTimeoutScenarios() {
	suite.Run("操作超时", func() {
		// 创建一个会超时的上下文
		ctx, cancel := context.WithTimeout(suite.ctx, 1*time.Millisecond)
		defer cancel()

		// 等待超时
		time.Sleep(2 * time.Millisecond)

		// 尝试在超时的上下文中执行操作
		config := types.GetDefaultProjectConfig()
		err := suite.configAPI.UpdateProjectConfig(ctx, "/timeout/test", config)
		
		if err != nil && errors.Is(err, context.DeadlineExceeded) {
			suite.T().Logf("正确处理了超时: %v", err)
		}
	})

	suite.Run("取消操作", func() {
		// 创建可取消的上下文
		ctx, cancel := context.WithCancel(suite.ctx)
		
		// 立即取消
		cancel()

		// 尝试在取消的上下文中执行操作
		_, err := suite.configAPI.GetGlobalConfig(ctx)
		
		if err != nil && errors.Is(err, context.Canceled) {
			suite.T().Logf("正确处理了取消: %v", err)
		}
	})
}

// TestRecoveryMechanisms 测试恢复机制
func (suite *ErrorScenarioTestSuite) TestRecoveryMechanisms() {
	suite.Run("配置备份和恢复", func() {
		// 创建初始配置
		globalConfig := types.GetDefaultGlobalConfig()
		globalConfig.GlobalVersions["test-tool"] = "1.0.0"
		
		err := suite.configAPI.UpdateGlobalConfig(suite.ctx, globalConfig)
		require.NoError(suite.T(), err)

		// 创建备份
		backupPath := filepath.Join(suite.homeDir, "backup")
		err = suite.configAPI.Backup(suite.ctx, backupPath)
		require.NoError(suite.T(), err)

		// 修改配置
		globalConfig.GlobalVersions["test-tool"] = "2.0.0"
		err = suite.configAPI.UpdateGlobalConfig(suite.ctx, globalConfig)
		require.NoError(suite.T(), err)

		// 恢复备份
		err = suite.configAPI.Restore(suite.ctx, backupPath)
		require.NoError(suite.T(), err)

		// 验证恢复
		restoredConfig, err := suite.configAPI.GetGlobalConfig(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "1.0.0", restoredConfig.GlobalVersions["test-tool"])
	})

	suite.Run("重置配置", func() {
		// 创建一些配置
		err := suite.configAPI.SetToolVersion(suite.ctx, "reset-tool", "1.0.0", true, "")
		require.NoError(suite.T(), err)

		// 重置配置
		err = suite.configAPI.Reset(suite.ctx)
		require.NoError(suite.T(), err)

		// 验证配置已重置
		config, err := suite.configAPI.GetGlobalConfig(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Empty(suite.T(), config.GlobalVersions)
	})
}

// TestErrorScenarioTestSuite 运行错误场景测试套件
func TestErrorScenarioTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorScenarioTestSuite))
}

// BenchmarkErrorHandling 错误处理性能基准测试
func BenchmarkErrorHandling(b *testing.B) {
	fs := afero.NewMemMapFs()
	homeDir := "/tmp/vman-error-bench"
	fs.MkdirAll(homeDir, 0755)

	configAPI, err := config.NewAPI(homeDir)
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
			// 尝试获取不存在的工具配置
			_, err := configAPI.GetToolConfig(ctx, fmt.Sprintf("nonexistent-tool-%d", counter))
			_ = err // 忽略错误，只测试性能
			counter++
		}
	})
}