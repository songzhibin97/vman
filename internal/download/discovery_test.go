package download

import (
	"context"
	"testing"

	"github.com/songzhibin97/vman/pkg/types"
	"github.com/stretchr/testify/assert"
)

// TestVersionParser_ParseVersion 测试版本解析
func TestVersionParser_ParseVersion(t *testing.T) {
	parser := NewVersionParser()

	tests := []struct {
		name          string
		version       string
		expectError   bool
		expectedType  string
		expectedMajor int
		expectedMinor int
		expectedPatch int
	}{
		{
			name:          "semver version",
			version:       "v1.2.3",
			expectError:   false,
			expectedType:  "semver",
			expectedMajor: 1,
			expectedMinor: 2,
			expectedPatch: 3,
		},
		{
			name:          "semver without v prefix",
			version:       "2.0.1",
			expectError:   false,
			expectedType:  "semver",
			expectedMajor: 2,
			expectedMinor: 0,
			expectedPatch: 1,
		},
		{
			name:          "semver with prerelease",
			version:       "1.0.0-alpha.1",
			expectError:   false,
			expectedType:  "semver",
			expectedMajor: 1,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:          "simple version",
			version:       "1.2",
			expectError:   false,
			expectedType:  "simple",
			expectedMajor: 1,
			expectedMinor: 2,
			expectedPatch: 0,
		},
		{
			name:          "date version",
			version:       "2023.12.25",
			expectError:   false,
			expectedType:  "semver",
			expectedMajor: 2023,
			expectedMinor: 12,
			expectedPatch: 25,
		},
		{
			name:        "invalid version",
			version:     "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.ParseVersion(tt.version)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, parsed)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, parsed)
				assert.Equal(t, tt.expectedType, parsed.Type)

				if tt.expectedType == "semver" || tt.expectedType == "simple" {
					assert.Equal(t, tt.expectedMajor, parsed.Major)
					assert.Equal(t, tt.expectedMinor, parsed.Minor)
					assert.Equal(t, tt.expectedPatch, parsed.Patch)
				}
			}
		})
	}
}

// TestParsedVersion_IsNewer 测试版本比较
func TestParsedVersion_IsNewer(t *testing.T) {
	parser := NewVersionParser()

	tests := []struct {
		name     string
		version1 string
		version2 string
		expected bool
	}{
		{
			name:     "newer major version",
			version1: "2.0.0",
			version2: "1.0.0",
			expected: true,
		},
		{
			name:     "newer minor version",
			version1: "1.2.0",
			version2: "1.1.0",
			expected: true,
		},
		{
			name:     "newer patch version",
			version1: "1.0.2",
			version2: "1.0.1",
			expected: true,
		},
		{
			name:     "same version",
			version1: "1.0.0",
			version2: "1.0.0",
			expected: false,
		},
		{
			name:     "older version",
			version1: "1.0.0",
			version2: "2.0.0",
			expected: false,
		},
		{
			name:     "prerelease vs release",
			version1: "1.0.0",
			version2: "1.0.0-alpha",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := parser.ParseVersion(tt.version1)
			assert.NoError(t, err)

			v2, err := parser.ParseVersion(tt.version2)
			assert.NoError(t, err)

			result := v1.IsNewer(v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCompareVersions 测试版本字符串比较
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		version1 string
		version2 string
		expected int
	}{
		{
			name:     "v1 > v2",
			version1: "2.0.0",
			version2: "1.0.0",
			expected: 1,
		},
		{
			name:     "v1 < v2",
			version1: "1.0.0",
			version2: "2.0.0",
			expected: -1,
		},
		{
			name:     "v1 == v2",
			version1: "1.0.0",
			version2: "1.0.0",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.version1, tt.version2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultPlatformMatcher_Match 测试平台匹配
func TestDefaultPlatformMatcher_Match(t *testing.T) {
	matcher := NewDefaultPlatformMatcher()

	tests := []struct {
		name     string
		platform *types.PlatformInfo
		expected bool
	}{
		{
			name: "valid linux amd64",
			platform: &types.PlatformInfo{
				OS:   "linux",
				Arch: "amd64",
			},
			expected: true,
		},
		{
			name: "valid darwin arm64",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			expected: true,
		},
		{
			name: "valid windows 386",
			platform: &types.PlatformInfo{
				OS:   "windows",
				Arch: "386",
			},
			expected: true,
		},
		{
			name: "invalid os",
			platform: &types.PlatformInfo{
				OS:   "invalid",
				Arch: "amd64",
			},
			expected: false,
		},
		{
			name: "invalid arch",
			platform: &types.PlatformInfo{
				OS:   "linux",
				Arch: "invalid",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultPlatformMatcher_IsArchiveSupported 测试归档文件支持
func TestDefaultPlatformMatcher_IsArchiveSupported(t *testing.T) {
	matcher := NewDefaultPlatformMatcher()

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "tar.gz file",
			filename: "kubectl.tar.gz",
			expected: true,
		},
		{
			name:     "tgz file",
			filename: "kubectl.tgz",
			expected: true,
		},
		{
			name:     "zip file",
			filename: "kubectl.zip",
			expected: true,
		},
		{
			name:     "tar.bz2 file",
			filename: "kubectl.tar.bz2",
			expected: true,
		},
		{
			name:     "tar.xz file",
			filename: "kubectl.tar.xz",
			expected: true,
		},
		{
			name:     "exe file",
			filename: "kubectl.exe",
			expected: false,
		},
		{
			name:     "plain binary",
			filename: "kubectl",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.IsArchiveSupported(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPlatformResolver_MatchesPlatform 测试平台匹配
func TestPlatformResolver_MatchesPlatform(t *testing.T) {
	resolver := NewPlatformResolver()

	tests := []struct {
		name     string
		filename string
		platform *types.PlatformInfo
		expected bool
	}{
		{
			name:     "linux amd64 match",
			filename: "kubectl-linux-amd64.tar.gz",
			platform: &types.PlatformInfo{OS: "linux", Arch: "amd64"},
			expected: true,
		},
		{
			name:     "darwin arm64 match",
			filename: "kubectl-darwin-arm64.zip",
			platform: &types.PlatformInfo{OS: "darwin", Arch: "arm64"},
			expected: true,
		},
		{
			name:     "windows x86_64 match",
			filename: "kubectl-windows-x86_64.exe",
			platform: &types.PlatformInfo{OS: "windows", Arch: "amd64"},
			expected: true,
		},
		{
			name:     "no platform match",
			filename: "kubectl-source.tar.gz",
			platform: &types.PlatformInfo{OS: "linux", Arch: "amd64"},
			expected: false,
		},
		{
			name:     "os match but no arch",
			filename: "kubectl-linux.tar.gz",
			platform: &types.PlatformInfo{OS: "linux", Arch: "amd64"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.MatchesPlatform(tt.filename, tt.platform)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultVersionDiscovery_SortVersions 测试版本排序
func TestDefaultVersionDiscovery_SortVersions(t *testing.T) {
	// 创建模拟策略
	mockStrategy := &MockStrategy{}
	discovery := NewVersionDiscovery(mockStrategy)

	// 测试数据
	versions := []*types.VersionInfo{
		{Version: "1.0.0"},
		{Version: "2.1.0"},
		{Version: "1.2.0"},
		{Version: "2.0.0"},
		{Version: "1.1.0"},
	}

	// 执行排序
	sorted := discovery.SortVersions(versions)

	// 验证结果（应该按降序排列）
	expected := []string{"2.1.0", "2.0.0", "1.2.0", "1.1.0", "1.0.0"}

	assert.Len(t, sorted, 5)
	for i, version := range sorted {
		assert.Equal(t, expected[i], version.Version, "Version at index %d should match", i)
	}
}

// TestDefaultVersionDiscovery_FilterByPlatform 测试平台过滤
func TestDefaultVersionDiscovery_FilterByPlatform(t *testing.T) {
	// 创建模拟策略
	mockStrategy := &MockStrategy{}
	discovery := NewVersionDiscovery(mockStrategy)

	// 测试平台
	platform := &types.PlatformInfo{
		OS:   "linux",
		Arch: "amd64",
	}

	// 测试数据
	versions := []*types.VersionInfo{
		{
			Version: "1.0.0",
			Downloads: map[string]types.DownloadInfo{
				"linux_amd64": {URL: "http://example.com/linux"},
			},
		},
		{
			Version: "1.1.0",
			Downloads: map[string]types.DownloadInfo{
				"darwin_arm64": {URL: "http://example.com/darwin"},
			},
		},
		{
			Version: "1.2.0",
			Downloads: map[string]types.DownloadInfo{
				"linux_amd64":   {URL: "http://example.com/linux2"},
				"windows_amd64": {URL: "http://example.com/windows"},
			},
		},
	}

	// 执行过滤
	filtered := discovery.FilterByPlatform(versions, platform)

	// 验证结果
	assert.Len(t, filtered, 2)
	assert.Equal(t, "1.0.0", filtered[0].Version)
	assert.Equal(t, "1.2.0", filtered[1].Version)
}

// MockStrategy 模拟下载策略
type MockStrategy struct{}

func (m *MockStrategy) GetDownloadInfo(ctx context.Context, version string) (*types.DownloadInfo, error) {
	return nil, nil
}

func (m *MockStrategy) GetDownloadURL(ctx context.Context, version string) (string, error) {
	return "", nil
}

func (m *MockStrategy) Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	return nil
}

func (m *MockStrategy) DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error {
	return nil
}

func (m *MockStrategy) ExtractArchive(archivePath, targetPath string) error {
	return nil
}

func (m *MockStrategy) GetLatestVersion(ctx context.Context) (string, error) {
	return "", nil
}

func (m *MockStrategy) ListVersions(ctx context.Context) ([]*types.VersionInfo, error) {
	return nil, nil
}

func (m *MockStrategy) ValidateVersion(ctx context.Context, version string) error {
	return nil
}

func (m *MockStrategy) GetChecksum(ctx context.Context, version string) (string, error) {
	return "", nil
}

func (m *MockStrategy) SupportsResume() bool {
	return false
}

func (m *MockStrategy) GetToolMetadata() *types.ToolMetadata {
	return nil
}
