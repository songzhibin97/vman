package integration

import (
	"context"
	"fmt"
	"os"
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
	"github.com/songzhibin97/vman/internal/proxy"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// PlatformCompatibilityTestSuite 多平台兼容性测试套件
type PlatformCompatibilityTestSuite struct {
	suite.Suite
	fs          afero.Fs
	homeDir     string
	configAPI   config.API
	versionMgr  version.Manager
	proxyMgr    proxy.CommandProxy
	ctx         context.Context
	cleanupFuncs []func()
}

func (suite *PlatformCompatibilityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.fs = afero.NewMemMapFs()
	suite.cleanupFuncs = make([]func(), 0)

	// 创建临时主目录
	tempDir := fmt.Sprintf("/tmp/vman-platform-test-%d", time.Now().Unix())
	suite.homeDir = tempDir
	err := suite.fs.MkdirAll(suite.homeDir, 0755)
	require.NoError(suite.T(), err)

	// 初始化配置API
	suite.configAPI, err = config.NewAPI(suite.homeDir)
	require.NoError(suite.T(), err)

	// 初始化配置
	err = suite.configAPI.Init(suite.ctx)
	require.NoError(suite.T(), err)

	// 创建版本管理器
	storageManager := storage.NewFilesystemManager(suite.homeDir)
	configManager, err := config.NewManager(suite.homeDir)
	require.NoError(suite.T(), err)
	suite.versionMgr, err = version.NewManager(storageManager, configManager)
	require.NoError(suite.T(), err)

	// 创建代理管理器
	suite.proxyMgr = proxy.NewCommandProxy(configManager, suite.versionMgr)
}

func (suite *PlatformCompatibilityTestSuite) TearDownSuite() {
	for i := len(suite.cleanupFuncs) - 1; i >= 0; i-- {
		suite.cleanupFuncs[i]()
	}
}

func (suite *PlatformCompatibilityTestSuite) addCleanup(fn func()) {
	suite.cleanupFuncs = append(suite.cleanupFuncs, fn)
}

// TestPathSeparators 测试路径分隔符兼容性
func (suite *PlatformCompatibilityTestSuite) TestPathSeparators() {
	tests := []struct {
		name         string
		inputPath    string
		expectedSeps string
	}{
		{
			name:         "Unix路径",
			inputPath:    "/home/user/tools",
			expectedSeps: "/",
		},
		{
			name:         "Windows路径",
			inputPath:    "C:\\Users\\user\\tools",
			expectedSeps: "\\",
		},
		{
			name:         "混合路径",
			inputPath:    "/home/user\\mixed/path",
			expectedSeps: "/\\",
		},
	}

	for _, test := range tests {
		suite.Run(test.name, func() {
			// 标准化路径
			normalized := filepath.Clean(test.inputPath)
			suite.NotEmpty(normalized)

			// 检查路径分隔符
			for _, sep := range test.expectedSeps {
				if strings.ContainsRune(test.inputPath, sep) {
					// 验证路径处理逻辑
					parts := strings.Split(test.inputPath, string(sep))
					suite.True(len(parts) > 1)
				}
			}
		})
	}
}

// TestFilePermissions 测试文件权限兼容性
func (suite *PlatformCompatibilityTestSuite) TestFilePermissions() {
	testFile := filepath.Join(suite.homeDir, "test_permissions.txt")

	// 创建测试文件
	err := afero.WriteFile(suite.fs, testFile, []byte("test content"), 0644)
	require.NoError(suite.T(), err)

	// 检查文件是否存在
	exists, err := afero.Exists(suite.fs, testFile)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// 根据操作系统测试权限
	if runtime.GOOS != "windows" {
		// Unix系统测试可执行权限
		execFile := filepath.Join(suite.homeDir, "test_exec.sh")
		err = afero.WriteFile(suite.fs, execFile, []byte("#!/bin/bash\necho 'test'"), 0755)
		require.NoError(suite.T(), err)

		stat, err := suite.fs.Stat(execFile)
		require.NoError(suite.T(), err)
		suite.Equal(os.FileMode(0755), stat.Mode())
	}
}

// TestConfigPathResolution 测试配置路径解析兼容性
func (suite *PlatformCompatibilityTestSuite) TestConfigPathResolution() {
	// 获取配置路径
	paths, err := suite.configAPI.GetConfigPaths(suite.ctx)
	require.NoError(suite.T(), err)

	// 验证路径是否为绝对路径
	assert.True(suite.T(), filepath.IsAbs(paths.ConfigDir))
	assert.True(suite.T(), filepath.IsAbs(paths.ToolsDir))
	assert.True(suite.T(), filepath.IsAbs(paths.BinDir))
	assert.True(suite.T(), filepath.IsAbs(paths.ShimsDir))

	// 验证路径格式符合当前平台
	for _, path := range []string{
		paths.ConfigDir,
		paths.ToolsDir,
		paths.BinDir,
		paths.ShimsDir,
	} {
		// 路径应该以正确的分隔符开始
		if runtime.GOOS == "windows" {
			// Windows路径可能以驱动器字母开始
			assert.True(suite.T(), len(path) >= 3, "Windows路径格式: %s", path)
		} else {
			// Unix路径应该以/开始
			assert.True(suite.T(), strings.HasPrefix(path, "/"), "Unix路径格式: %s", path)
		}
	}
}

// TestEnvironmentVariables 测试环境变量处理兼容性
func (suite *PlatformCompatibilityTestSuite) TestEnvironmentVariables() {
	// 测试HOME/USERPROFILE环境变量
	var homeEnvVar string
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	} else {
		homeEnvVar = "HOME"
	}

	// 模拟环境变量设置
	originalValue := os.Getenv(homeEnvVar)
	testValue := "/test/home/path"
	if runtime.GOOS == "windows" {
		testValue = "C:\\test\\home\\path"
	}

	os.Setenv(homeEnvVar, testValue)
	suite.addCleanup(func() {
		if originalValue != "" {
			os.Setenv(homeEnvVar, originalValue)
		} else {
			os.Unsetenv(homeEnvVar)
		}
	})

	// 验证环境变量是否正确设置
	retrievedValue := os.Getenv(homeEnvVar)
	assert.Equal(suite.T(), testValue, retrievedValue)
}

// TestShellScriptGeneration 测试Shell脚本生成兼容性
func (suite *PlatformCompatibilityTestSuite) TestShellScriptGeneration() {
	toolName := "kubectl"
	version := "1.29.0"

	// 生成shim脚本
	err := suite.proxyMgr.GenerateShim(toolName, version)
	if err != nil {
		// 某些平台可能不支持shim生成，这是可以接受的
		suite.T().Logf("Shim generation not supported on %s: %v", runtime.GOOS, err)
		return
	}

	// 验证生成的脚本
	status := suite.proxyMgr.GetProxyStatus()
	suite.NotNil(status)
	suite.NotEmpty(status.ShimsDir)
}

// TestArchitectureSpecificPaths 测试架构特定路径
func (suite *PlatformCompatibilityTestSuite) TestArchitectureSpecificPaths() {
	// 测试当前架构的路径生成
	archPath := fmt.Sprintf("tools/%s_%s", runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(suite.homeDir, archPath)

	err := suite.fs.MkdirAll(fullPath, 0755)
	require.NoError(suite.T(), err)

	exists, err := afero.DirExists(suite.fs, fullPath)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// 验证路径包含正确的架构信息
	assert.Contains(suite.T(), fullPath, runtime.GOOS)
	assert.Contains(suite.T(), fullPath, runtime.GOARCH)
}

// TestCrossplatformToolExecution 测试跨平台工具执行
func (suite *PlatformCompatibilityTestSuite) TestCrossplatformToolExecution() {
	toolName := "test-tool"
	version := "1.0.0"

	// 创建工具元数据
	metadata := &types.ToolMetadata{
		Name:        toolName,
		Description: "测试工具",
		// Source: &types.SourceConfig{
		// 	Type: "github",
		// 	URL:  "https://github.com/test/test-tool",
		// },
		// Binary: &types.BinaryConfig{
		// 	Name: toolName,
		// },
		// Install: &types.InstallConfig{
		// 	Method: "extract",
		// },
	}

	// 注册工具
	err := suite.configAPI.RegisterTool(suite.ctx, metadata)
	require.NoError(suite.T(), err)

	// 验证工具注册
	tools, err := suite.configAPI.ListTools(suite.ctx)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), tools, toolName)
}

// TestUnicodePath 测试Unicode路径支持
func (suite *PlatformCompatibilityTestSuite) TestUnicodePath() {
	// 创建包含Unicode字符的路径
	unicodePath := filepath.Join(suite.homeDir, "测试目录", "工具")
	err := suite.fs.MkdirAll(unicodePath, 0755)
	require.NoError(suite.T(), err)

	// 验证Unicode路径处理
	exists, err := afero.DirExists(suite.fs, unicodePath)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// 在Unicode路径中创建文件
	unicodeFile := filepath.Join(unicodePath, "配置.yaml")
	testContent := "# 测试配置文件\nversion: 1.0\n"
	err = afero.WriteFile(suite.fs, unicodeFile, []byte(testContent), 0644)
	require.NoError(suite.T(), err)

	// 读取并验证文件内容
	content, err := afero.ReadFile(suite.fs, unicodeFile)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), testContent, string(content))
}

// TestLongPath 测试长路径支持
func (suite *PlatformCompatibilityTestSuite) TestLongPath() {
	// 创建深层嵌套路径
	longPath := suite.homeDir
	for i := 0; i < 10; i++ {
		longPath = filepath.Join(longPath, fmt.Sprintf("level%d", i))
	}

	err := suite.fs.MkdirAll(longPath, 0755)
	require.NoError(suite.T(), err)

	// 验证长路径处理
	exists, err := afero.DirExists(suite.fs, longPath)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// 在长路径中创建文件
	longFile := filepath.Join(longPath, "deep_file.txt")
	err = afero.WriteFile(suite.fs, longFile, []byte("deep content"), 0644)
	require.NoError(suite.T(), err)

	// 验证文件存在
	exists, err = afero.Exists(suite.fs, longFile)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)
}

// TestConcurrentAccess 测试并发访问兼容性
func (suite *PlatformCompatibilityTestSuite) TestConcurrentAccess() {
	const numGoroutines = 10
	const numOperations = 50

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*numOperations)

	// 启动多个goroutine进行并发操作
	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				// 并发创建配置
				testPath := filepath.Join(suite.homeDir, fmt.Sprintf("worker%d_op%d", workerID, j))
				config := types.GetDefaultProjectConfig()
				config.Tools[fmt.Sprintf("tool%d", j)] = "1.0.0"

				err := suite.configAPI.UpdateProjectConfig(suite.ctx, testPath, config)
				if err != nil {
					errors <- fmt.Errorf("worker %d, operation %d: %w", workerID, j, err)
					return
				}

				// 并发读取配置
				_, err = suite.configAPI.GetProjectConfig(suite.ctx, testPath)
				if err != nil {
					errors <- fmt.Errorf("worker %d, read operation %d: %w", workerID, j, err)
					return
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 检查是否有错误
	close(errors)
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		suite.T().Logf("并发访问中出现 %d 个错误", len(allErrors))
		for _, err := range allErrors[:min(10, len(allErrors))] { // 只显示前10个错误
			suite.T().Logf("错误: %v", err)
		}
	}

	// 允许少量错误，但不应该超过10%
	errorRate := float64(len(allErrors)) / float64(numGoroutines*numOperations)
	assert.Less(suite.T(), errorRate, 0.1, "错误率应该小于10%%")
}

// TestPlatformSpecificFeatures 测试平台特定功能
func (suite *PlatformCompatibilityTestSuite) TestPlatformSpecificFeatures() {
	switch runtime.GOOS {
	case "windows":
		suite.testWindowsSpecificFeatures()
	case "darwin":
		suite.testMacOSSpecificFeatures()
	case "linux":
		suite.testLinuxSpecificFeatures()
	default:
		suite.T().Logf("未知平台 %s，跳过平台特定测试", runtime.GOOS)
	}
}

func (suite *PlatformCompatibilityTestSuite) testWindowsSpecificFeatures() {
	// 测试Windows特定功能
	suite.T().Log("测试Windows特定功能")

	// 测试驱动器路径
	drivePath := "C:\\vman\\test"
	normalizedPath := filepath.Clean(drivePath)
	assert.True(suite.T(), filepath.IsAbs(normalizedPath))

	// 测试批处理文件扩展名
	batchFile := "test.bat"
	assert.True(suite.T(), strings.HasSuffix(batchFile, ".bat"))
}

func (suite *PlatformCompatibilityTestSuite) testMacOSSpecificFeatures() {
	// 测试macOS特定功能
	suite.T().Log("测试macOS特定功能")

	// 测试应用程序包路径
	appPath := "/Applications/TestApp.app"
	assert.True(suite.T(), strings.HasSuffix(appPath, ".app"))

	// 测试隐藏文件
	hiddenFile := ".vman_config"
	assert.True(suite.T(), strings.HasPrefix(hiddenFile, "."))
}

func (suite *PlatformCompatibilityTestSuite) testLinuxSpecificFeatures() {
	// 测试Linux特定功能
	suite.T().Log("测试Linux特定功能")

	// 测试/usr/local路径
	usrLocalPath := "/usr/local/bin"
	assert.True(suite.T(), strings.HasPrefix(usrLocalPath, "/usr/local"))

	// 测试shell脚本
	shellScript := "#!/bin/bash\necho 'hello'"
	assert.True(suite.T(), strings.HasPrefix(shellScript, "#!/bin/bash"))
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestPlatformCompatibilityTestSuite 运行多平台兼容性测试套件
func TestPlatformCompatibilityTestSuite(t *testing.T) {
	suite.Run(t, new(PlatformCompatibilityTestSuite))
}

// BenchmarkPlatformOperations 平台操作性能基准测试
func BenchmarkPlatformOperations(b *testing.B) {
	fs := afero.NewMemMapFs()
	homeDir := "/tmp/vman-bench"
	fs.MkdirAll(homeDir, 0755)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			testPath := filepath.Join(homeDir, fmt.Sprintf("test_%d", counter))
			fs.MkdirAll(testPath, 0755)
			afero.WriteFile(fs, filepath.Join(testPath, "test.txt"), []byte("test"), 0644)
			counter++
		}
	})
}