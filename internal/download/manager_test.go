package download

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/songzhibin97/vman/pkg/types"
)

// MockStorageManager 存储管理器模拟
type MockStorageManager struct {
	mock.Mock
}

func (m *MockStorageManager) GetToolsDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetConfigDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetShimsDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetCacheDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetSourcesDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) EnsureDirectories() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorageManager) CleanupOrphaned() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorageManager) GetToolVersionPath(tool, version string) string {
	args := m.Called(tool, version)
	return args.String(0)
}

func (m *MockStorageManager) GetToolVersions(tool string) ([]string, error) {
	args := m.Called(tool)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorageManager) GetVersionsDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetBinDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetLogsDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) GetTempDir() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageManager) CreateVersionDir(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockStorageManager) RemoveVersionDir(tool, version string) error {
	args := m.Called(tool, version)
	return args.Error(0)
}

func (m *MockStorageManager) IsVersionInstalled(tool, version string) bool {
	args := m.Called(tool, version)
	return args.Bool(0)
}

func (m *MockStorageManager) GetVersionMetadataPath(tool, version string) string {
	args := m.Called(tool, version)
	return args.String(0)
}

func (m *MockStorageManager) SaveVersionMetadata(tool, version string, metadata *types.VersionMetadata) error {
	args := m.Called(tool, version, metadata)
	return args.Error(0)
}

func (m *MockStorageManager) LoadVersionMetadata(tool, version string) (*types.VersionMetadata, error) {
	args := m.Called(tool, version)
	return args.Get(0).(*types.VersionMetadata), args.Error(1)
}

func (m *MockStorageManager) GetBinaryPath(tool, version string) string {
	args := m.Called(tool, version)
	return args.String(0)
}

// MockConfigManager 配置管理器模拟
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

func (m *MockConfigManager) GetEffectiveVersion(toolName, projectPath string) (string, error) {
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

// TestDefaultManager_AddSource 测试添加下载源
func TestDefaultManager_AddSource(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	mockStorage.On("GetSourcesDir").Return("/tmp/sources")

	// 创建内存文件系统
	fs := afero.NewMemMapFs()

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 测试数据
	toolMetadata := &types.ToolMetadata{
		Name:        "kubectl",
		Description: "Kubernetes command-line tool",
		DownloadConfig: types.DownloadConfig{
			Type:       "github",
			Repository: "kubernetes/kubernetes",
		},
	}

	// 执行测试
	err := manager.AddSource("kubectl", toolMetadata)

	// 验证结果
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

// TestDefaultManager_ListSources 测试列出下载源
func TestDefaultManager_ListSources(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	mockStorage.On("GetSourcesDir").Return("/tmp/sources")

	// 创建内存文件系统并添加测试文件
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/tmp/sources", 0755)
	afero.WriteFile(fs, "/tmp/sources/kubectl.toml", []byte("test"), 0644)
	afero.WriteFile(fs, "/tmp/sources/terraform.toml", []byte("test"), 0644)

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 执行测试
	sources, err := manager.ListSources()

	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, sources, 2)
	assert.Contains(t, sources, "kubectl")
	assert.Contains(t, sources, "terraform")

	mockStorage.AssertExpectations(t)
}

// TestDefaultManager_RemoveSource 测试移除下载源
func TestDefaultManager_RemoveSource(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	mockStorage.On("GetSourcesDir").Return("/tmp/sources")

	// 创建内存文件系统并添加测试文件
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/tmp/sources", 0755)
	afero.WriteFile(fs, "/tmp/sources/kubectl.toml", []byte("test"), 0644)

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 执行测试
	err := manager.RemoveSource("kubectl")

	// 验证结果
	assert.NoError(t, err)

	// 验证文件已删除
	exists, _ := afero.Exists(fs, "/tmp/sources/kubectl.toml")
	assert.False(t, exists)

	mockStorage.AssertExpectations(t)
}

// TestDefaultManager_GetDownloadStrategy 测试获取下载策略
func TestDefaultManager_GetDownloadStrategy(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	toolMetadata := &types.ToolMetadata{
		Name: "kubectl",
		DownloadConfig: types.DownloadConfig{
			Type:       "github",
			Repository: "kubernetes/kubernetes",
		},
	}
	mockConfig.On("LoadToolConfig", "kubectl").Return(toolMetadata, nil)

	// 创建内存文件系统
	fs := afero.NewMemMapFs()

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 执行测试
	strategy, err := manager.GetDownloadStrategy("kubectl")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, strategy)
	assert.Equal(t, toolMetadata, strategy.GetToolMetadata())

	mockConfig.AssertExpectations(t)
}

// TestDefaultManager_ClearCache 测试清理缓存
func TestDefaultManager_ClearCache(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	mockStorage.On("GetCacheDir").Return("/tmp/cache")

	// 创建内存文件系统并添加测试文件
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/tmp/cache/kubectl", 0755)
	afero.WriteFile(fs, "/tmp/cache/kubectl/test.file", []byte("test"), 0644)

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 执行测试
	err := manager.ClearCache("kubectl")

	// 验证结果
	assert.NoError(t, err)

	// 验证缓存已清理
	exists, _ := afero.Exists(fs, "/tmp/cache/kubectl")
	assert.False(t, exists)

	mockStorage.AssertExpectations(t)
}

// TestDefaultManager_GetCacheSize 测试获取缓存大小
func TestDefaultManager_GetCacheSize(t *testing.T) {
	// 创建模拟对象
	mockStorage := new(MockStorageManager)
	mockConfig := new(MockConfigManager)

	// 设置期望
	mockStorage.On("GetCacheDir").Return("/tmp/cache")

	// 创建内存文件系统并添加测试文件
	fs := afero.NewMemMapFs()
	fs.MkdirAll("/tmp/cache/kubectl", 0755)
	afero.WriteFile(fs, "/tmp/cache/kubectl/test1.file", []byte("test1"), 0644)
	afero.WriteFile(fs, "/tmp/cache/kubectl/test2.file", []byte("test2"), 0644)

	// 创建管理器
	manager := NewManagerWithFs(mockStorage, mockConfig, fs)

	// 执行测试
	size, err := manager.GetCacheSize("kubectl")

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, int64(10), size) // "test1" + "test2" = 10 bytes

	mockStorage.AssertExpectations(t)
}
