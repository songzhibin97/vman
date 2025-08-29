package types

import (
	"runtime"
	"testing"
)

func TestGetCurrentPlatform(t *testing.T) {
	platform := GetCurrentPlatform()

	// 验证平台信息不为空
	if platform == nil {
		t.Fatal("GetCurrentPlatform() returned nil")
	}

	// 验证操作系统信息
	if platform.OS == "" {
		t.Error("Platform OS should not be empty")
	}

	// 验证架构信息
	if platform.Arch == "" {
		t.Error("Platform Arch should not be empty")
	}

	// 验证返回的平台信息与runtime包一致
	if platform.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, platform.OS)
	}

	if platform.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, platform.Arch)
	}

	t.Logf("Detected platform: %s/%s", platform.OS, platform.Arch)
}

func TestPlatformInfo_GetPlatformKey(t *testing.T) {
	tests := []struct {
		name     string
		platform *PlatformInfo
		expected string
	}{
		{
			name: "darwin/arm64",
			platform: &PlatformInfo{
				OS:   "darwin",
				Arch: "arm64",
			},
			expected: "darwin_arm64",
		},
		{
			name: "linux/amd64",
			platform: &PlatformInfo{
				OS:   "linux",
				Arch: "amd64",
			},
			expected: "linux_amd64",
		},
		{
			name: "windows/amd64",
			platform: &PlatformInfo{
				OS:   "windows",
				Arch: "amd64",
			},
			expected: "windows_amd64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.platform.GetPlatformKey()
			if result != tt.expected {
				t.Errorf("GetPlatformKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPlatformSupport(t *testing.T) {
	platform := GetCurrentPlatform()

	// 测试支持的平台
	supportedOS := []string{"darwin", "linux", "windows"}
	supportedArch := []string{"amd64", "arm64", "386"}

	osSupported := false
	for _, os := range supportedOS {
		if platform.OS == os {
			osSupported = true
			break
		}
	}

	if !osSupported {
		t.Errorf("Unsupported OS: %s", platform.OS)
	}

	archSupported := false
	for _, arch := range supportedArch {
		if platform.Arch == arch {
			archSupported = true
			break
		}
	}

	if !archSupported {
		t.Errorf("Unsupported Arch: %s", platform.Arch)
	}
}

// 测试特定平台场景 - 仅在对应平台运行
func TestDarwinARM64Platform(t *testing.T) {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		t.Skip("Skipping darwin/arm64 specific test")
	}

	platform := GetCurrentPlatform()
	if platform.OS != "darwin" {
		t.Errorf("Expected OS 'darwin', got '%s'", platform.OS)
	}

	if platform.Arch != "arm64" {
		t.Errorf("Expected Arch 'arm64', got '%s'", platform.Arch)
	}

	key := platform.GetPlatformKey()
	if key != "darwin_arm64" {
		t.Errorf("Expected platform key 'darwin_arm64', got '%s'", key)
	}
}

func TestLinuxAMD64Platform(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Skipping linux/amd64 specific test")
	}

	platform := GetCurrentPlatform()
	if platform.OS != "linux" {
		t.Errorf("Expected OS 'linux', got '%s'", platform.OS)
	}

	if platform.Arch != "amd64" {
		t.Errorf("Expected Arch 'amd64', got '%s'", platform.Arch)
	}

	key := platform.GetPlatformKey()
	if key != "linux_amd64" {
		t.Errorf("Expected platform key 'linux_amd64', got '%s'", key)
	}
}