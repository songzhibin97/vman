package download

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

func TestGitHubStrategy_matchAssetByPattern(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name           string
		assetPattern   string
		platform       *types.PlatformInfo
		assets         []GitHubAsset
		expectedAsset  string
		shouldMatch    bool
		description    string
	}{
		{
			name:         "darwin/arm64 protoc-gen-go pattern",
			assetPattern: "protoc-gen-go.v{version}.{os}.{arch}.tar.gz",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			assets: []GitHubAsset{
				{Name: "protoc-gen-go.v1.31.0.darwin.arm64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.linux.amd64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.windows.amd64.tar.gz"},
			},
			expectedAsset: "protoc-gen-go.v1.31.0.darwin.arm64.tar.gz",
			shouldMatch:   true,
			description:   "Should match darwin/arm64 asset for protoc-gen-go",
		},
		{
			name:         "darwin/amd64 protoc-gen-go pattern",
			assetPattern: "protoc-gen-go.v{version}.{os}.{arch}.tar.gz",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "amd64",
			},
			assets: []GitHubAsset{
				{Name: "protoc-gen-go.v1.31.0.darwin.amd64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.linux.amd64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.windows.amd64.tar.gz"},
			},
			expectedAsset: "protoc-gen-go.v1.31.0.darwin.amd64.tar.gz",
			shouldMatch:   true,
			description:   "Should match darwin/amd64 asset for protoc-gen-go",
		},
		{
			name:         "linux/amd64 protoc-gen-go pattern",
			assetPattern: "protoc-gen-go.v{version}.{os}.{arch}.tar.gz",
			platform: &types.PlatformInfo{
				OS:   "linux",
				Arch: "amd64",
			},
			assets: []GitHubAsset{
				{Name: "protoc-gen-go.v1.31.0.darwin.arm64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.linux.amd64.tar.gz"},
				{Name: "protoc-gen-go.v1.31.0.windows.amd64.tar.gz"},
			},
			expectedAsset: "protoc-gen-go.v1.31.0.linux.amd64.tar.gz",
			shouldMatch:   true,
			description:   "Should match linux/amd64 asset for protoc-gen-go",
		},
		{
			name:         "protoc-gen-go-http underscore pattern",
			assetPattern: "protoc-gen-go-http_{os}_{arch}.tar.gz",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			assets: []GitHubAsset{
				{Name: "protoc-gen-go-http_darwin_arm64.tar.gz"},
				{Name: "protoc-gen-go-http_linux_amd64.tar.gz"},
				{Name: "protoc-gen-go-http_windows_amd64.tar.gz"},
			},
			expectedAsset: "protoc-gen-go-http_darwin_arm64.tar.gz",
			shouldMatch:   true,
			description:   "Should match darwin/arm64 asset for protoc-gen-go-http with underscore pattern",
		},
		{
			name:         "no matching asset",
			assetPattern: "tool-{os}-{arch}.tar.gz",
			platform: &types.PlatformInfo{
				OS:   "freebsd",
				Arch: "riscv64",
			},
			assets: []GitHubAsset{
				{Name: "tool-darwin-arm64.tar.gz"},
				{Name: "tool-linux-amd64.tar.gz"},
				{Name: "tool-windows-amd64.tar.gz"},
			},
			shouldMatch: false,
			description: "Should not match when platform not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &types.ToolMetadata{
				Name: "test-tool",
				DownloadConfig: types.DownloadConfig{
					AssetPattern: tt.assetPattern,
				},
			}

			strategy := &GitHubStrategy{
				metadata: metadata,
				fs:       fs,
				logger:   logger,
			}

			asset, err := strategy.matchAssetByPattern(tt.assets, tt.platform)

			if tt.shouldMatch {
				if err != nil {
					t.Errorf("Expected to find matching asset, but got error: %v", err)
					return
				}

				if asset.Name != tt.expectedAsset {
					t.Errorf("Expected asset %s, got %s", tt.expectedAsset, asset.Name)
				}

				t.Logf("✓ %s - Found: %s", tt.description, asset.Name)
			} else {
				if err == nil {
					t.Errorf("Expected no matching asset, but found: %s", asset.Name)
				}
				t.Logf("✓ %s - No match as expected", tt.description)
			}
		})
	}
}

func TestGitHubStrategy_matchAssetByDefault(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logrus.New()

	strategy := &GitHubStrategy{
		metadata: &types.ToolMetadata{Name: "test-tool"},
		fs:       fs,
		logger:   logger,
	}

	tests := []struct {
		name          string
		platform      *types.PlatformInfo
		assets        []GitHubAsset
		expectedAsset string
		description   string
	}{
		{
			name: "darwin/arm64 with various naming conventions",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			assets: []GitHubAsset{
				{Name: "tool-linux-amd64.tar.gz"},
				{Name: "tool-darwin-arm64.tar.gz"},
				{Name: "tool-windows-amd64.exe"},
			},
			expectedAsset: "tool-darwin-arm64.tar.gz",
			description:   "Should match darwin arm64 asset",
		},
		{
			name: "darwin/arm64 with macos naming",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			assets: []GitHubAsset{
				{Name: "tool-linux-amd64.tar.gz"},
				{Name: "tool-macos-arm64.tar.gz"},
				{Name: "tool-windows-amd64.exe"},
			},
			expectedAsset: "tool-macos-arm64.tar.gz",
			description:   "Should match macos arm64 asset when darwin not available",
		},
		{
			name: "linux/amd64 with x86_64 naming",
			platform: &types.PlatformInfo{
				OS:   "linux",
				Arch: "amd64",
			},
			assets: []GitHubAsset{
				{Name: "tool-linux-x86_64.tar.gz"},
				{Name: "tool-darwin-arm64.tar.gz"},
				{Name: "tool-windows-amd64.exe"},
			},
			expectedAsset: "tool-linux-x86_64.tar.gz",
			description:   "Should match linux x86_64 asset",
		},
		{
			name: "darwin/arm64 with aarch64 naming",
			platform: &types.PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			assets: []GitHubAsset{
				{Name: "tool-linux-amd64.tar.gz"},
				{Name: "tool-darwin-aarch64.tar.gz"},
				{Name: "tool-windows-amd64.exe"},
			},
			expectedAsset: "tool-darwin-aarch64.tar.gz",
			description:   "Should match darwin aarch64 asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := strategy.matchAssetByDefault(tt.assets, tt.platform)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if asset.Name != tt.expectedAsset {
				t.Errorf("Expected asset %s, got %s", tt.expectedAsset, asset.Name)
			}

			t.Logf("✓ %s - Found: %s", tt.description, asset.Name)
		})
	}
}

func TestGitHubStrategy_mapOSName(t *testing.T) {
	strategy := &GitHubStrategy{}

	tests := []struct {
		input    string
		expected string
	}{
		{"darwin", "darwin"},
		{"linux", "linux"},
		{"windows", "windows"},
		{"freebsd", "freebsd"}, // 未映射的应该返回原值
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strategy.mapOSName(tt.input)
			if result != tt.expected {
				t.Errorf("mapOSName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGitHubStrategy_mapArchName(t *testing.T) {
	strategy := &GitHubStrategy{}

	tests := []struct {
		input    string
		expected string
	}{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"386", "386"},
		{"riscv64", "riscv64"}, // 未映射的应该返回原值
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := strategy.mapArchName(tt.input)
			if result != tt.expected {
				t.Errorf("mapArchName(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}