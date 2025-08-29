package config

import (
	"testing"
	"time"

	"github.com/songzhibin97/vman/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDefaultValidator_ValidateGlobalConfig(t *testing.T) {
	validator := NewValidator()

	t.Run("ValidConfig", func(t *testing.T) {
		config := types.GetDefaultGlobalConfig()
		err := validator.ValidateGlobalConfig(config)
		assert.NoError(t, err)
	})

	t.Run("NilConfig", func(t *testing.T) {
		err := validator.ValidateGlobalConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "global config cannot be nil")
	})

	t.Run("EmptyVersion", func(t *testing.T) {
		config := types.GetDefaultGlobalConfig()
		config.Version = ""
		err := validator.ValidateGlobalConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config version is required")
	})

	t.Run("InvalidLogLevel", func(t *testing.T) {
		config := types.GetDefaultGlobalConfig()
		config.Settings.Logging.Level = "invalid"
		err := validator.ValidateGlobalConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
	})

	t.Run("NegativeRetries", func(t *testing.T) {
		config := types.GetDefaultGlobalConfig()
		config.Settings.Download.Retries = -1
		err := validator.ValidateGlobalConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retries must be >= 0")
	})

	t.Run("InvalidConcurrentDownloads", func(t *testing.T) {
		config := types.GetDefaultGlobalConfig()
		config.Settings.Download.ConcurrentDownloads = 0
		err := validator.ValidateGlobalConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrent_downloads must be >= 1")
	})
}

func TestDefaultValidator_ValidateProjectConfig(t *testing.T) {
	validator := NewValidator()

	t.Run("ValidConfig", func(t *testing.T) {
		config := types.GetDefaultProjectConfig()
		config.Tools["kubectl"] = "1.28.0"
		err := validator.ValidateProjectConfig(config)
		assert.NoError(t, err)
	})

	t.Run("NilConfig", func(t *testing.T) {
		err := validator.ValidateProjectConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project config cannot be nil")
	})

	t.Run("EmptyVersion", func(t *testing.T) {
		config := types.GetDefaultProjectConfig()
		config.Version = ""
		err := validator.ValidateProjectConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config version is required")
	})

	t.Run("InvalidToolVersion", func(t *testing.T) {
		config := types.GetDefaultProjectConfig()
		config.Tools["invalid-tool@"] = "1.0.0"
		err := validator.ValidateProjectConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tool name")
	})
}

func TestDefaultValidator_ValidateToolMetadata(t *testing.T) {
	validator := NewValidator()

	t.Run("ValidMetadata", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "kubectl",
			Description: "Kubernetes command-line tool",
			Homepage:    "https://kubernetes.io/",
			Repository:  "https://github.com/kubernetes/kubernetes",
			DownloadConfig: types.DownloadConfig{
				Type:        "direct",
				URLTemplate: "https://dl.k8s.io/release/v{version}/bin/{os}/{arch}/kubectl",
			},
			VersionConfig: types.VersionConfig{
				Aliases: map[string]string{
					"latest": "1.29.0",
					"stable": "1.28.0",
				},
				Constraints: types.VersionConstraints{
					MinVersion: "1.20.0",
				},
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.NoError(t, err)
	})

	t.Run("NilMetadata", func(t *testing.T) {
		err := validator.ValidateToolMetadata(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool metadata cannot be nil")
	})

	t.Run("InvalidName", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "invalid@name",
			Description: "Test tool",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type:        "direct",
				URLTemplate: "https://example.com/{version}",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool name can only contain")
	})

	t.Run("EmptyDescription", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:       "testtool",
			Homepage:   "https://example.com",
			Repository: "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type:        "direct",
				URLTemplate: "https://example.com/{version}",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool description is required")
	})

	t.Run("InvalidHomepageURL", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "testtool",
			Description: "Test tool",
			Homepage:    "invalid-url",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type:        "direct",
				URLTemplate: "https://example.com/{version}",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "homepage URL must start with http")
	})

	t.Run("InvalidDownloadType", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "testtool",
			Description: "Test tool",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type: "invalid",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid download type")
	})

	t.Run("DirectDownloadMissingURLTemplate", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "testtool",
			Description: "Test tool",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type: "direct",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "url_template is required for direct download type")
	})

	t.Run("ArchiveDownloadMissingExtractBinary", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "testtool",
			Description: "Test tool",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type:        "archive",
				URLTemplate: "https://example.com/{version}.zip",
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "extract_binary is required for archive download type")
	})

	t.Run("InvalidVersionConstraints", func(t *testing.T) {
		metadata := &types.ToolMetadata{
			Name:        "testtool",
			Description: "Test tool",
			Homepage:    "https://example.com",
			Repository:  "https://github.com/example/tool",
			DownloadConfig: types.DownloadConfig{
				Type:        "direct",
				URLTemplate: "https://example.com/{version}",
			},
			VersionConfig: types.VersionConfig{
				Constraints: types.VersionConstraints{
					MinVersion: "2.0.0",
					MaxVersion: "1.0.0", // max < min
				},
			},
		}
		err := validator.ValidateToolMetadata(metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "min_version cannot be greater than max_version")
	})
}

func TestDefaultValidator_ValidateVersion(t *testing.T) {
	validator := NewValidator()

	validVersions := []string{
		"1.0.0",
		"v1.0.0",
		"1.0.0-alpha",
		"1.0.0-alpha.1",
		"1.0.0+build",
		"1.0",
		"v1.0",
		"latest",
		"stable",
		"main",
		"master",
	}

	for _, version := range validVersions {
		t.Run("Valid_"+version, func(t *testing.T) {
			err := validator.ValidateVersion(version)
			assert.NoError(t, err, "Version %s should be valid", version)
		})
	}

	invalidVersions := []string{
		"",
		"   ",
		"invalid-version",
		"1.0.0.0.0",
	}

	for _, version := range invalidVersions {
		t.Run("Invalid_"+version, func(t *testing.T) {
			err := validator.ValidateVersion(version)
			assert.Error(t, err, "Version %s should be invalid", version)
		})
	}
}

func TestDefaultValidator_ValidateToolName(t *testing.T) {
	validator := NewValidator()

	validNames := []string{
		"kubectl",
		"terraform",
		"test-tool",
		"test_tool",
		"tool123",
		"a",
		"a-b_c123",
	}

	for _, name := range validNames {
		t.Run("Valid_"+name, func(t *testing.T) {
			err := validator.ValidateToolName(name)
			assert.NoError(t, err, "Tool name %s should be valid", name)
		})
	}

	invalidNames := []string{
		"",
		"   ",
		"tool with spaces",
		"tool@invalid",
		"tool.invalid",
		"tool/invalid",
		"tool\\invalid",
		repeat("a", 51), // 超过50字符
	}

	for _, name := range invalidNames {
		t.Run("Invalid_"+name, func(t *testing.T) {
			err := validator.ValidateToolName(name)
			assert.Error(t, err, "Tool name %s should be invalid", name)
		})
	}
}

func TestDefaultValidator_ValidatePath(t *testing.T) {
	validator := NewValidator()

	validPaths := []string{
		"/absolute/path",
		"./relative/path",
		"../relative/path",
		"~/home/path",
		"relative/path",
	}

	for _, path := range validPaths {
		t.Run("Valid_"+path, func(t *testing.T) {
			err := validator.ValidatePath(path)
			assert.NoError(t, err, "Path %s should be valid", path)
		})
	}

	invalidPaths := []string{
		"",
		"   ",
	}

	for _, path := range invalidPaths {
		t.Run("Invalid_"+path, func(t *testing.T) {
			err := validator.ValidatePath(path)
			assert.Error(t, err, "Path %s should be invalid", path)
		})
	}
}

func TestDefaultValidator_ValidateDownloadSettings(t *testing.T) {
	validator := &DefaultValidator{}

	t.Run("ValidSettings", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             3,
			ConcurrentDownloads: 2,
		}
		err := validator.validateDownloadSettings(settings)
		assert.NoError(t, err)
	})

	t.Run("ZeroTimeout", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             0,
			Retries:             3,
			ConcurrentDownloads: 2,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout must be greater than 0")
	})

	t.Run("ExcessiveTimeout", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             31 * time.Minute,
			Retries:             3,
			ConcurrentDownloads: 2,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout cannot exceed 30 minutes")
	})

	t.Run("NegativeRetries", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             -1,
			ConcurrentDownloads: 2,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retries must be >= 0")
	})

	t.Run("ExcessiveRetries", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             11,
			ConcurrentDownloads: 2,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retries cannot exceed 10")
	})

	t.Run("ZeroConcurrentDownloads", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             3,
			ConcurrentDownloads: 0,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrent_downloads must be >= 1")
	})

	t.Run("ExcessiveConcurrentDownloads", func(t *testing.T) {
		settings := &types.DownloadSettings{
			Timeout:             300 * time.Second,
			Retries:             3,
			ConcurrentDownloads: 11,
		}
		err := validator.validateDownloadSettings(settings)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrent_downloads cannot exceed 10")
	})
}
