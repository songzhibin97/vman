package proxy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/pkg/types"
)

// CommandRouterTestSuite 命令路由器测试套件
type CommandRouterTestSuite struct {
	suite.Suite
	fs            afero.Fs
	router        *DefaultCommandRouter
	mockVersion   *MockVersionManager
	mockConfig    *MockConfigManager
	tempDir       string
}

func (suite *CommandRouterTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.tempDir = "/tmp/router-test"
	suite.fs.MkdirAll(suite.tempDir, 0755)
	
	suite.mockVersion = new(MockVersionManager)
	suite.mockConfig = new(MockConfigManager)
	
	suite.router = &DefaultCommandRouter{
		fs:             suite.fs,
		versionManager: suite.mockVersion,
		configManager:  suite.mockConfig,
	}
}

func (suite *CommandRouterTestSuite) TestRouteCommand() {
	toolName := "kubectl"
	version := "1.29.0"
	args := []string{"get", "pods"}
	workingDir := "/test/project"
	
	// 设置mock期望
	suite.mockVersion.On("GetEffectiveVersion", toolName, workingDir).Return(version, nil)
	suite.mockVersion.On("GetVersionPath", toolName, version).Return("/versions/kubectl/1.29.0", nil)
	
	// 创建二进制文件
	binaryPath := "/versions/kubectl/1.29.0/bin/kubectl"
	suite.fs.MkdirAll(filepath.Dir(binaryPath), 0755)
	err := afero.WriteFile(suite.fs, binaryPath, []byte("#!/bin/bash\necho 'kubectl'"), 0755)
	suite.NoError(err)
	
	// 执行路由
	ctx := context.Background()
	result, err := suite.router.RouteCommand(ctx, toolName, args, workingDir)
	
	suite.NoError(err)
	suite.Equal(binaryPath, result.BinaryPath)
	suite.Equal(version, result.Version)
	suite.Equal(args, result.Args)
	suite.Equal(workingDir, result.WorkingDir)
	
	suite.mockVersion.AssertExpectations(suite.T())
}

func (suite *CommandRouterTestSuite) TestRouteCommandVersionNotInstalled() {
	toolName := "terraform"
	version := "1.6.0"
	workingDir := "/test/project"
	
	// 设置mock期望
	suite.mockVersion.On("GetEffectiveVersion", toolName, workingDir).Return(version, nil)
	suite.mockVersion.On("GetVersionPath", toolName, version).Return("", fmt.Errorf("version not installed"))
	
	// 执行路由
	ctx := context.Background()
	_, err := suite.router.RouteCommand(ctx, toolName, []string{}, workingDir)
	
	suite.Error(err)
	suite.Contains(err.Error(), "version not installed")
}

func (suite *CommandRouterTestSuite) TestRouteCommandBinaryNotFound() {
	toolName := "helm"
	version := "3.14.0"
	workingDir := "/test/project"
	versionPath := "/versions/helm/3.14.0"
	
	// 设置mock期望
	suite.mockVersion.On("GetEffectiveVersion", toolName, workingDir).Return(version, nil)
	suite.mockVersion.On("GetVersionPath", toolName, version).Return(versionPath, nil)
	
	// 不创建二进制文件
	
	// 执行路由
	ctx := context.Background()
	_, err := suite.router.RouteCommand(ctx, toolName, []string{}, workingDir)
	
	suite.Error(err)
	suite.Contains(err.Error(), "二进制文件不存在")
}

func (suite *CommandRouterTestSuite) TestGetCommandInfo() {
	toolName := "sqlc"
	workingDir := "/test/project"
	version := "1.25.0"
	
	// 设置mock期望
	suite.mockVersion.On("GetEffectiveVersion", toolName, workingDir).Return(version, nil)
	
	metadata := &types.VersionMetadata{
		Version:     version,
		ToolName:    toolName,
		InstallPath: "/versions/sqlc/1.25.0",
		Size:        1024000,
		InstallType: "download",
	}
	suite.mockVersion.On("GetVersionMetadata", toolName, version).Return(metadata, nil)
	
	// 执行获取命令信息
	ctx := context.Background()
	info, err := suite.router.GetCommandInfo(ctx, toolName, workingDir)
	
	suite.NoError(err)
	suite.Equal(toolName, info.Tool)
	suite.Equal(version, info.Version)
	suite.Equal(metadata.InstallPath, info.InstallPath)
	suite.Equal(metadata.Size, info.Size)
	suite.Equal("download", info.InstallType)
	
	suite.mockVersion.AssertExpectations(suite.T())
}

func TestCommandRouterTestSuite(t *testing.T) {
	suite.Run(t, new(CommandRouterTestSuite))
}

// InterceptorTestSuite 拦截器测试套件
type InterceptorTestSuite struct {
	suite.Suite
	fs          afero.Fs
	interceptor *DefaultInterceptor
	mockRouter  *MockCommandRouter
	tempDir     string
}

// MockCommandRouter 模拟命令路由器
type MockCommandRouter struct {
	mock.Mock
}

func (m *MockCommandRouter) RouteCommand(ctx context.Context, tool string, args []string, workingDir string) (*CommandRoute, error) {
	arguments := m.Called(ctx, tool, args, workingDir)
	return arguments.Get(0).(*CommandRoute), arguments.Error(1)
}

func (m *MockCommandRouter) GetCommandInfo(ctx context.Context, tool string, workingDir string) (*CommandInfo, error) {
	arguments := m.Called(ctx, tool, workingDir)
	return arguments.Get(0).(*CommandInfo), arguments.Error(1)
}

func (suite *InterceptorTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.tempDir = "/tmp/interceptor-test"
	suite.fs.MkdirAll(suite.tempDir, 0755)
	
	suite.mockRouter = new(MockCommandRouter)
	suite.interceptor = &DefaultInterceptor{
		fs:     suite.fs,
		router: suite.mockRouter,
	}
}

func (suite *InterceptorTestSuite) TestInterceptCommand() {
	toolName := "kubectl"
	args := []string{"get", "pods"}
	workingDir := "/test/project"
	
	// 设置mock期望
	route := &CommandRoute{
		Tool:       toolName,
		Version:    "1.29.0",
		BinaryPath: "/versions/kubectl/1.29.0/bin/kubectl",
		Args:       args,
		WorkingDir: workingDir,
		Env:        []string{"PATH=/usr/bin"},
	}
	suite.mockRouter.On("RouteCommand", mock.Anything, toolName, args, workingDir).Return(route, nil)
	
	// 执行拦截
	ctx := context.Background()
	result, err := suite.interceptor.InterceptCommand(ctx, toolName, args, workingDir)
	
	suite.NoError(err)
	suite.Equal(route, result)
	
	suite.mockRouter.AssertExpectations(suite.T())
}

func (suite *InterceptorTestSuite) TestInterceptCommandWithEnvVars() {
	toolName := "terraform"
	args := []string{"plan"}
	workingDir := "/test/project"
	
	// 设置环境变量
	os.Setenv("TF_VAR_test", "value")
	defer os.Unsetenv("TF_VAR_test")
	
	// 设置mock期望
	route := &CommandRoute{
		Tool:       toolName,
		Version:    "1.6.0",
		BinaryPath: "/versions/terraform/1.6.0/bin/terraform",
		Args:       args,
		WorkingDir: workingDir,
		Env:        []string{"TF_VAR_test=value"},
	}
	suite.mockRouter.On("RouteCommand", mock.Anything, toolName, args, workingDir).Return(route, nil)
	
	// 执行拦截
	ctx := context.Background()
	result, err := suite.interceptor.InterceptCommand(ctx, toolName, args, workingDir)
	
	suite.NoError(err)
	suite.Equal(route, result)
}

func TestInterceptorTestSuite(t *testing.T) {
	suite.Run(t, new(InterceptorTestSuite))
}

// ShellIntegratorTestSuite Shell集成器测试套件
type ShellIntegratorTestSuite struct {
	suite.Suite
	fs         afero.Fs
	integrator *DefaultShellIntegrator
	tempDir    string
}

func (suite *ShellIntegratorTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.tempDir = "/tmp/shell-test"
	suite.fs.MkdirAll(suite.tempDir, 0755)
	
	suite.integrator = &DefaultShellIntegrator{
		fs: suite.fs,
	}
}

func (suite *ShellIntegratorTestSuite) TestGenerateBashIntegration() {
	shimDir := "/test/shims"
	
	script, err := suite.integrator.GenerateBashIntegration(shimDir)
	suite.NoError(err)
	
	suite.Contains(script, "export PATH")
	suite.Contains(script, shimDir)
	suite.Contains(script, "vman_proxy")
}

func (suite *ShellIntegratorTestSuite) TestGenerateZshIntegration() {
	shimDir := "/test/shims"
	
	script, err := suite.integrator.GenerateZshIntegration(shimDir)
	suite.NoError(err)
	
	suite.Contains(script, "export PATH")
	suite.Contains(script, shimDir)
	suite.Contains(script, "vman_proxy")
}

func (suite *ShellIntegratorTestSuite) TestGenerateFishIntegration() {
	shimDir := "/test/shims"
	
	script, err := suite.integrator.GenerateFishIntegration(shimDir)
	suite.NoError(err)
	
	suite.Contains(script, "set -gx PATH")
	suite.Contains(script, shimDir)
}

func (suite *ShellIntegratorTestSuite) TestWriteIntegrationScript() {
	shimDir := "/test/shims"
	scriptPath := filepath.Join(suite.tempDir, "integration.sh")
	
	err := suite.integrator.WriteIntegrationScript("bash", shimDir, scriptPath)
	suite.NoError(err)
	
	// 验证文件存在
	exists, err := afero.Exists(suite.fs, scriptPath)
	suite.NoError(err)
	suite.True(exists)
	
	// 验证文件内容
	content, err := afero.ReadFile(suite.fs, scriptPath)
	suite.NoError(err)
	suite.Contains(string(content), shimDir)
}

func (suite *ShellIntegratorTestSuite) TestDetectShell() {
	// 保存原始环境变量
	originalShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", originalShell)
	
	// 测试bash检测
	os.Setenv("SHELL", "/bin/bash")
	shell := suite.integrator.DetectShell()
	suite.Equal("bash", shell)
	
	// 测试zsh检测
	os.Setenv("SHELL", "/usr/bin/zsh")
	shell = suite.integrator.DetectShell()
	suite.Equal("zsh", shell)
	
	// 测试fish检测
	os.Setenv("SHELL", "/usr/local/bin/fish")
	shell = suite.integrator.DetectShell()
	suite.Equal("fish", shell)
	
	// 测试未知shell
	os.Setenv("SHELL", "/bin/unknown")
	shell = suite.integrator.DetectShell()
	suite.Equal("bash", shell) // 默认返回bash
}

func TestShellIntegratorTestSuite(t *testing.T) {
	suite.Run(t, new(ShellIntegratorTestSuite))
}

// PerformanceTestSuite 性能监控器测试套件
type PerformanceTestSuite struct {
	suite.Suite
	monitor *DefaultPerformanceMonitor
}

func (suite *PerformanceTestSuite) SetupTest() {
	suite.monitor = &DefaultPerformanceMonitor{}
}

func (suite *PerformanceTestSuite) TestTrackCommandExecution() {
	ctx := context.Background()
	
	// 开始跟踪
	session := suite.monitor.StartTracking(ctx, "kubectl", "1.29.0")
	suite.NotNil(session)
	suite.Equal("kubectl", session.Tool)
	suite.Equal("1.29.0", session.Version)
	
	// 结束跟踪
	suite.monitor.EndTracking(session)
	suite.True(session.Duration > 0)
}

func (suite *PerformanceTestSuite) TestGetStatistics() {
	ctx := context.Background()
	
	// 模拟一些命令执行
	for i := 0; i < 5; i++ {
		session := suite.monitor.StartTracking(ctx, "kubectl", "1.29.0")
		suite.monitor.EndTracking(session)
	}
	
	for i := 0; i < 3; i++ {
		session := suite.monitor.StartTracking(ctx, "terraform", "1.6.0")
		suite.monitor.EndTracking(session)
	}
	
	// 获取统计信息
	stats := suite.monitor.GetStatistics()
	suite.NotNil(stats)
	
	// 验证统计信息
	kubectlStats, exists := stats.ToolStats["kubectl"]
	suite.True(exists)
	suite.Equal(5, kubectlStats.ExecutionCount)
	
	terraformStats, exists := stats.ToolStats["terraform"]
	suite.True(exists)
	suite.Equal(3, terraformStats.ExecutionCount)
}

func (suite *PerformanceTestSuite) TestClearStatistics() {
	ctx := context.Background()
	
	// 模拟一些命令执行
	session := suite.monitor.StartTracking(ctx, "kubectl", "1.29.0")
	suite.monitor.EndTracking(session)
	
	// 清除统计信息
	suite.monitor.ClearStatistics()
	
	// 验证统计信息被清除
	stats := suite.monitor.GetStatistics()
	suite.Empty(stats.ToolStats)
}

func TestPerformanceTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceTestSuite))
}

// CacheManagerTestSuite 缓存管理器测试套件
type CacheManagerTestSuite struct {
	suite.Suite
	cache *DefaultCacheManager
}

func (suite *CacheManagerTestSuite) SetupTest() {
	suite.cache = NewCacheManager(100, 0) // 最大100个条目，无TTL
}

func (suite *CacheManagerTestSuite) TestVersionPathCache() {
	tool := "kubectl"
	version := "1.29.0"
	path := "/versions/kubectl/1.29.0"
	
	// 设置缓存
	suite.cache.SetVersionPath(tool, version, path)
	
	// 获取缓存
	cachedPath, found := suite.cache.GetVersionPath(tool, version)
	suite.True(found)
	suite.Equal(path, cachedPath)
	
	// 测试不存在的缓存
	_, found = suite.cache.GetVersionPath("nonexistent", "1.0.0")
	suite.False(found)
}

func (suite *CacheManagerTestSuite) TestCommandInfoCache() {
	tool := "terraform"
	workingDir := "/test/project"
	info := &CommandInfo{
		Tool:        tool,
		Version:     "1.6.0",
		InstallPath: "/versions/terraform/1.6.0",
		Size:        1024000,
	}
	
	// 设置缓存
	suite.cache.SetCommandInfo(tool, workingDir, info)
	
	// 获取缓存
	cachedInfo, found := suite.cache.GetCommandInfo(tool, workingDir)
	suite.True(found)
	suite.Equal(info, cachedInfo)
}

func (suite *CacheManagerTestSuite) TestCacheEviction() {
	// 创建小容量缓存
	smallCache := NewCacheManager(2, 0)
	
	// 添加项目直到超过容量
	smallCache.SetVersionPath("tool1", "1.0.0", "/path1")
	smallCache.SetVersionPath("tool2", "1.0.0", "/path2")
	smallCache.SetVersionPath("tool3", "1.0.0", "/path3") // 应该触发淘汰
	
	// 验证最旧的项被淘汰
	_, found := smallCache.GetVersionPath("tool1", "1.0.0")
	suite.False(found)
	
	// 验证新项存在
	_, found = smallCache.GetVersionPath("tool3", "1.0.0")
	suite.True(found)
}

func (suite *CacheManagerTestSuite) TestClearCache() {
	// 添加一些缓存项
	suite.cache.SetVersionPath("kubectl", "1.29.0", "/path1")
	suite.cache.SetVersionPath("terraform", "1.6.0", "/path2")
	
	// 清除缓存
	suite.cache.Clear()
	
	// 验证缓存被清除
	_, found := suite.cache.GetVersionPath("kubectl", "1.29.0")
	suite.False(found)
	_, found = suite.cache.GetVersionPath("terraform", "1.6.0")
	suite.False(found)
}

func TestCacheManagerTestSuite(t *testing.T) {
	suite.Run(t, new(CacheManagerTestSuite))
}

// 集成测试

// TestProxyIntegration 代理集成测试
func TestProxyIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过Windows平台的集成测试")
	}
	
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "vman-proxy-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// 创建模拟的kubectl二进制文件
	kubectlDir := filepath.Join(tempDir, "versions", "kubectl", "1.29.0", "bin")
	err = os.MkdirAll(kubectlDir, 0755)
	require.NoError(t, err)
	
	kubectlPath := filepath.Join(kubectlDir, "kubectl")
	kubectlScript := `#!/bin/bash
echo "kubectl version v1.29.0"
echo "args: $@"
`
	err = os.WriteFile(kubectlPath, []byte(kubectlScript), 0755)
	require.NoError(t, err)
	
	// 创建shim目录
	shimDir := filepath.Join(tempDir, "shims")
	err = os.MkdirAll(shimDir, 0755)
	require.NoError(t, err)
	
	// 创建kubectl shim
	shimPath := filepath.Join(shimDir, "kubectl")
	shimScript := fmt.Sprintf(`#!/bin/bash
exec "%s" "$@"
`, kubectlPath)
	err = os.WriteFile(shimPath, []byte(shimScript), 0755)
	require.NoError(t, err)
	
	// 测试通过shim执行命令
	cmd := exec.Command(shimPath, "version")
	output, err := cmd.Output()
	require.NoError(t, err)
	
	outputStr := string(output)
	assert.Contains(t, outputStr, "kubectl version v1.29.0")
	assert.Contains(t, outputStr, "args: version")
}

// BenchmarkCommandRouting 命令路由性能基准测试
func BenchmarkCommandRouting(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockVersion := new(MockVersionManager)
	mockConfig := new(MockConfigManager)
	
	router := &DefaultCommandRouter{
		fs:             fs,
		versionManager: mockVersion,
		configManager:  mockConfig,
	}
	
	// 设置mock期望
	mockVersion.On("GetEffectiveVersion", "kubectl", "/test").Return("1.29.0", nil)
	mockVersion.On("GetVersionPath", "kubectl", "1.29.0").Return("/versions/kubectl/1.29.0", nil)
	
	// 创建二进制文件
	binaryPath := "/versions/kubectl/1.29.0/bin/kubectl"
	fs.MkdirAll(filepath.Dir(binaryPath), 0755)
	afero.WriteFile(fs, binaryPath, []byte("binary"), 0755)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := router.RouteCommand(ctx, "kubectl", []string{"get", "pods"}, "/test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCacheOperations 缓存操作性能基准测试
func BenchmarkCacheOperations(b *testing.B) {
	cache := NewCacheManager(1000, 0)
	
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.SetVersionPath(fmt.Sprintf("tool-%d", i%100), "1.0.0", "/path")
		}
	})
	
	b.Run("Get", func(b *testing.B) {
		// 预填充缓存
		for i := 0; i < 100; i++ {
			cache.SetVersionPath(fmt.Sprintf("tool-%d", i), "1.0.0", "/path")
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetVersionPath(fmt.Sprintf("tool-%d", i%100), "1.0.0")
		}
	})
}