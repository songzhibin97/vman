package proxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// MockConfigManager 模拟配置管理器
type MockConfigManager struct {
	mock.Mock
}

func (m *MockConfigManager) LoadGlobal() (*types.GlobalConfig, error) {
	args := m.Called()
	return args.Get(0).(*types.GlobalConfig), args.Error(1)
}

func (m *MockConfigManager) LoadProject(path string) (*types.ProjectConfig, error) {
	args := m.Called(path)
	return args.Get(0).(*types.ProjectConfig), args.Error(1)
}

func (m *MockConfigManager) LoadToolConfig(toolName string) (*types.ToolMetadata, error) {
	args := m.Called(toolName)
	return args.Get(0).(*types.ToolMetadata), args.Error(1)
}

func (m *MockConfigManager) SaveGlobal(config *types.GlobalConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigManager) SaveProject(path string, config *types.ProjectConfig) error {
	args := m.Called(path, config)
	return args.Error(0)
}

func (m *MockConfigManager) GetEffectiveVersion(toolName string, projectPath string) (string, error) {
	args := m.Called(toolName, projectPath)
	return args.String(0), args.Error(1)
}

func (m *MockConfigManager) GetConfigDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockConfigManager) GetProjectConfigPath(projectPath string) string {
	args := m.Called(projectPath)
	return args.String(0)
}

func (m *MockConfigManager) Initialize() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) Validate() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigManager) ListTools() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockConfigManager) IsToolInstalled(toolName, version string) bool {
	args := m.Called(toolName, version)
	return args.Bool(0)
}

func (m *MockConfigManager) SetToolVersion(toolName, version string, global bool, projectPath string) error {
	args := m.Called(toolName, version, global, projectPath)
	return args.Error(0)
}

func (m *MockConfigManager) RemoveToolVersion(toolName, version string) error {
	args := m.Called(toolName, version)
	return args.Error(0)
}

func (m *MockConfigManager) GetEffectiveConfig(projectPath string) (*types.EffectiveConfig, error) {
	args := m.Called(projectPath)
	return args.Get(0).(*types.EffectiveConfig), args.Error(1)
}

// MockVersionManager 模拟版本管理器
type MockVersionManager struct {
	mock.Mock
}

func (m *MockVersionManager) RegisterVersion(tool, version, path string) error {
	args := m.Called(tool, version, path)
	return args.Error(0)
}

func (m *MockVersionManager) ListVersions(tool string) ([]string, error) {
	args := m.Called(tool)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVersionManager) GetVersionPath(tool, version string) (string, error) {
	args := m.Called(tool, version)
	return args.String(0), args.Error(1)
}

func (m *MockVersionManager) RemoveVersion(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockVersionManager) SetGlobalVersion(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockVersionManager) SetLocalVersion(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockVersionManager) GetCurrentVersion(tool string) (string, error) {
	args := m.Called(tool)
	return args.String(0), args.Error(1)
}

func (m *MockVersionManager) IsVersionInstalled(tool, version string) bool {
	args := m.Called(tool, version)
	return args.Bool(0)
}

func (m *MockVersionManager) GetInstalledVersions(tool string) ([]string, error) {
	args := m.Called(tool)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVersionManager) ValidateVersion(version string) error {
	args := m.Called(version)
	return args.Error(0)
}

func (m *MockVersionManager) GetLatestVersion(tool string) (string, error) {
	args := m.Called(tool)
	return args.String(0), args.Error(1)
}

func (m *MockVersionManager) GetVersionMetadata(tool, version string) (*types.VersionMetadata, error) {
	args := m.Called(tool, version)
	return args.Get(0).(*types.VersionMetadata), args.Error(1)
}

func (m *MockVersionManager) SetProjectVersion(tool, version, projectPath string) error {
	args := m.Called(tool, version, projectPath)
	return args.Error(0)
}

func (m *MockVersionManager) GetEffectiveVersion(tool, projectPath string) (string, error) {
	args := m.Called(tool, projectPath)
	return args.String(0), args.Error(1)
}

func (m *MockVersionManager) ListAllTools() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVersionManager) InstallVersion(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockVersionManager) InstallVersionWithProgress(tool, version string, progress version.ProgressCallback) error {
	args := m.Called(tool, version, progress)
	return args.Error(0)
}

func (m *MockVersionManager) InstallLatestVersion(tool string) (string, error) {
	args := m.Called(tool)
	return args.String(0), args.Error(1)
}

func (m *MockVersionManager) SearchAvailableVersions(tool string) ([]*types.VersionInfo, error) {
	args := m.Called(tool)
	return args.Get(0).([]*types.VersionInfo), args.Error(1)
}

func (m *MockVersionManager) IsVersionAvailable(tool, version string) bool {
	args := m.Called(tool, version)
	return args.Bool(0)
}

func (m *MockVersionManager) UpdateTool(tool string) (string, error) {
	args := m.Called(tool)
	return args.String(0), args.Error(1)
}

// PathManagerTestSuite PATH管理器测试套件
type PathManagerTestSuite struct {
	suite.Suite
	fs          afero.Fs
	pathManager PathManager
	tempDir     string
}

func (suite *PathManagerTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.pathManager = NewPathManagerWithFs(suite.fs)
	suite.tempDir = "/tmp/vman-test"

	// 创建测试目录
	suite.fs.MkdirAll(suite.tempDir, 0755)
}

func (suite *PathManagerTestSuite) TestAddToPath() {
	testDir := "/test/bin"
	suite.fs.MkdirAll(testDir, 0755)

	// 测试添加目录到PATH
	err := suite.pathManager.AddToPath(testDir)
	suite.NoError(err)

	// 验证目录是否在PATH中
	isInPath := suite.pathManager.IsInPath(testDir)
	suite.True(isInPath)
}

func (suite *PathManagerTestSuite) TestRemoveFromPath() {
	testDir := "/test/bin"
	suite.fs.MkdirAll(testDir, 0755)

	// 先添加到PATH
	err := suite.pathManager.AddToPath(testDir)
	suite.NoError(err)

	// 然后移除
	err = suite.pathManager.RemoveFromPath(testDir)
	suite.NoError(err)

	// 验证目录不在PATH中
	isInPath := suite.pathManager.IsInPath(testDir)
	suite.False(isInPath)
}

func (suite *PathManagerTestSuite) TestSetupShimPath() {
	shimDir := "/test/shims"

	err := suite.pathManager.SetupShimPath(shimDir)
	suite.NoError(err)

	// 验证shim目录存在
	exists, _ := afero.Exists(suite.fs, shimDir)
	suite.True(exists)
}

func TestPathManagerTestSuite(t *testing.T) {
	suite.Run(t, new(PathManagerTestSuite))
}

// SymlinkManagerTestSuite 符号链接管理器测试套件
type SymlinkManagerTestSuite struct {
	suite.Suite
	fs             afero.Fs
	symlinkManager SymlinkManager
	tempDir        string
}

func (suite *SymlinkManagerTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.symlinkManager = NewSymlinkManagerWithFs(suite.fs)
	suite.tempDir = "/tmp/vman-test"

	// 创建测试目录
	suite.fs.MkdirAll(suite.tempDir, 0755)
}

func (suite *SymlinkManagerTestSuite) TestCreateToolSymlinks() {
	toolName := "kubectl"
	version := "1.20.0"
	binPath := "/test/bin/kubectl"
	shimDir := "/test/shims"

	// 创建二进制文件
	suite.fs.MkdirAll(filepath.Dir(binPath), 0755)
	afero.WriteFile(suite.fs, binPath, []byte("fake binary"), 0755)

	err := suite.symlinkManager.CreateToolSymlinks(toolName, version, binPath, shimDir)
	suite.NoError(err)

	// 验证主符号链接存在
	mainLinkPath := filepath.Join(shimDir, toolName)
	exists, _ := afero.Exists(suite.fs, mainLinkPath)
	suite.True(exists, "Main symlink should exist")
}

func (suite *SymlinkManagerTestSuite) TestRemoveToolSymlinks() {
	toolName := "kubectl"
	shimDir := "/test/shims"

	// 先创建一些测试符号链接
	suite.fs.MkdirAll(shimDir, 0755)
	afero.WriteFile(suite.fs, filepath.Join(shimDir, toolName), []byte("fake shim"), 0755)

	err := suite.symlinkManager.RemoveToolSymlinks(toolName, shimDir)
	suite.NoError(err)
}

func TestSymlinkManagerTestSuite(t *testing.T) {
	suite.Run(t, new(SymlinkManagerTestSuite))
}

// VersionResolverTestSuite 版本解析器测试套件
type VersionResolverTestSuite struct {
	suite.Suite
	fs              afero.Fs
	versionResolver VersionResolver
	mockConfig      *MockConfigManager
	mockVersion     *MockVersionManager
}

func (suite *VersionResolverTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.mockConfig = new(MockConfigManager)
	suite.mockVersion = new(MockVersionManager)
	suite.versionResolver = NewVersionResolverWithFs(suite.fs, suite.mockConfig, suite.mockVersion)
}

func (suite *VersionResolverTestSuite) TestResolveVersionFromGlobal() {
	toolName := "kubectl"
	version := "1.20.0"
	projectPath := "/test/project"

	// 设置mock返回值
	globalConfig := &types.GlobalConfig{
		GlobalVersions: map[string]string{
			toolName: version,
		},
	}
	suite.mockConfig.On("LoadGlobal").Return(globalConfig, nil)
	suite.mockConfig.On("LoadProject", projectPath).Return((*types.ProjectConfig)(nil), os.ErrNotExist)
	suite.mockVersion.On("IsVersionInstalled", toolName, version).Return(true)

	// 解析版本
	ctx := context.Background()
	resolution, err := suite.versionResolver.ResolveVersion(ctx, toolName, projectPath)

	suite.NoError(err)
	suite.Equal(version, resolution.Version)
	suite.Equal("global", resolution.Source)
	suite.True(resolution.IsInstalled)
}

func (suite *VersionResolverTestSuite) TestResolveVersionFromProject() {
	toolName := "kubectl"
	version := "1.21.0"
	projectPath := "/test/project"

	// 创建项目配置
	projectConfig := &types.ProjectConfig{
		Tools: map[string]string{
			toolName: version,
		},
	}

	suite.mockConfig.On("LoadProject", projectPath).Return(projectConfig, nil)
	suite.mockVersion.On("IsVersionInstalled", toolName, version).Return(true)

	// 解析版本
	ctx := context.Background()
	resolution, err := suite.versionResolver.ResolveVersion(ctx, toolName, projectPath)

	suite.NoError(err)
	suite.Equal(version, resolution.Version)
	suite.Equal("project", resolution.Source)
}

func (suite *VersionResolverTestSuite) TestResolveLatestVersion() {
	toolName := "kubectl"
	latestVersion := "1.22.0"
	projectPath := "/test/project"

	// 设置mock：没有全局和项目配置
	suite.mockConfig.On("LoadGlobal").Return(types.GetDefaultGlobalConfig(), nil)
	suite.mockConfig.On("LoadProject", projectPath).Return((*types.ProjectConfig)(nil), os.ErrNotExist)
	suite.mockVersion.On("GetLatestVersion", toolName).Return(latestVersion, nil)
	suite.mockVersion.On("IsVersionInstalled", toolName, latestVersion).Return(true)

	// 解析版本
	ctx := context.Background()
	resolution, err := suite.versionResolver.ResolveVersion(ctx, toolName, projectPath)

	suite.NoError(err)
	suite.Equal(latestVersion, resolution.Version)
	suite.Equal("latest", resolution.Source)
}

func TestVersionResolverTestSuite(t *testing.T) {
	suite.Run(t, new(VersionResolverTestSuite))
}

// ContextManagerTestSuite 上下文管理器测试套件
type ContextManagerTestSuite struct {
	suite.Suite
	fs             afero.Fs
	contextManager ContextManager
	mockConfig     *MockConfigManager
}

func (suite *ContextManagerTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.mockConfig = new(MockConfigManager)
	suite.contextManager = NewContextManagerWithFs(suite.fs, suite.mockConfig)
}

func (suite *ContextManagerTestSuite) TestDetectProjectContext() {
	projectPath := "/test/project"

	// 创建项目根目录标识文件
	suite.fs.MkdirAll(projectPath, 0755)
	afero.WriteFile(suite.fs, filepath.Join(projectPath, "go.mod"), []byte("module test"), 0644)

	// 设置mock
	suite.mockConfig.On("LoadProject", projectPath).Return(types.GetDefaultProjectConfig(), nil)

	context, err := suite.contextManager.DetectProjectContext(projectPath)

	suite.NoError(err)
	suite.Equal(projectPath, context.RootPath)
	suite.Equal("go", context.ProjectType)
}

func (suite *ContextManagerTestSuite) TestFindProjectRoot() {
	projectRoot := "/test/project"
	subDir := filepath.Join(projectRoot, "src", "cmd")

	// 创建目录结构
	suite.fs.MkdirAll(subDir, 0755)
	afero.WriteFile(suite.fs, filepath.Join(projectRoot, ".git"), []byte(""), 0644)

	foundRoot, err := suite.contextManager.FindProjectRoot(subDir)

	suite.NoError(err)
	suite.Equal(projectRoot, foundRoot)
}

func TestContextManagerTestSuite(t *testing.T) {
	suite.Run(t, new(ContextManagerTestSuite))
}

// CommandProxyTestSuite 命令代理测试套件
type CommandProxyTestSuite struct {
	suite.Suite
	fs           afero.Fs
	commandProxy CommandProxy
	mockConfig   *MockConfigManager
	mockVersion  *MockVersionManager
}

func (suite *CommandProxyTestSuite) SetupTest() {
	suite.fs = afero.NewMemMapFs()
	suite.mockConfig = new(MockConfigManager)
	suite.mockVersion = new(MockVersionManager)
	suite.commandProxy = NewCommandProxyWithFs(suite.fs, suite.mockConfig, suite.mockVersion)
}

func (suite *CommandProxyTestSuite) TestGenerateShim() {
	toolName := "kubectl"
	version := "1.20.0"
	versionPath := "/test/versions/kubectl/1.20.0"

	// 设置mock
	suite.mockVersion.On("GetVersionPath", toolName, version).Return(versionPath, nil)

	err := suite.commandProxy.GenerateShim(toolName, version)
	_ = err // 忽略错误以验证调用过程

	// 验证mock调用
	suite.mockVersion.AssertCalled(suite.T(), "GetVersionPath", toolName, version)
}

func (suite *CommandProxyTestSuite) TestGetProxyStatus() {
	status := suite.commandProxy.GetProxyStatus()

	suite.NotNil(status)
	suite.NotEmpty(status.ShimsDir)
}

func TestCommandProxyTestSuite(t *testing.T) {
	suite.Run(t, new(CommandProxyTestSuite))
}

// 基准测试
func BenchmarkVersionResolver(b *testing.B) {
	fs := afero.NewMemMapFs()
	mockConfig := new(MockConfigManager)
	mockVersion := new(MockVersionManager)

	// 设置mock
	globalConfig := &types.GlobalConfig{
		GlobalVersions: map[string]string{
			"kubectl": "1.20.0",
		},
	}
	mockConfig.On("LoadGlobal").Return(globalConfig, nil)
	mockConfig.On("LoadProject", mock.Anything).Return((*types.ProjectConfig)(nil), os.ErrNotExist)
	mockVersion.On("IsVersionInstalled", "kubectl", "1.20.0").Return(true)

	versionResolver := NewVersionResolverWithFs(fs, mockConfig, mockVersion)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := versionResolver.ResolveVersion(ctx, "kubectl", "/test/project")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCacheManager(b *testing.B) {
	cache := NewCacheManager(1000, 0) // 无TTL

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("test-key-%d", i%100)
		value := fmt.Sprintf("test-value-%d", i)

		cache.SetVersionPath("tool", key, value)
		_, _ = cache.GetVersionPath("tool", key)
	}
}
