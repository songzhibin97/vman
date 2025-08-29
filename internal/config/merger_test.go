package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/vman/pkg/types"
)

func TestDefaultMerger_MergeConfigs(t *testing.T) {
	merger := NewMerger()

	globalConfig := &types.GlobalConfig{
		Version: "1.0",
		Settings: types.Settings{
			Download: types.DownloadSettings{
				Timeout:             300 * time.Second,
				Retries:             3,
				ConcurrentDownloads: 2,
			},
			Logging: types.LoggingSettings{
				Level: "info",
				File:  "~/.vman/logs/vman.log",
			},
		},
		GlobalVersions: map[string]string{
			"kubectl":   "1.28.0",
			"terraform": "1.5.0",
		},
		Tools: map[string]types.ToolInfo{
			"sqlc": {
				CurrentVersion:    "1.19.0",
				InstalledVersions: []string{"1.19.0"},
			},
		},
	}

	projectConfig := &types.ProjectConfig{
		Version: "1.0",
		Tools: map[string]string{
			"kubectl": "1.29.0", // 覆盖全局版本
			"helm":    "3.12.0", // 项目特有工具
		},
	}

	tests := []struct {
		name     string
		strategy types.ConfigMergeStrategy
		expected map[string]string
		sources  map[string]string
	}{
		{
			name:     "override strategy",
			strategy: types.OverrideStrategy,
			expected: map[string]string{
				"kubectl":   "1.29.0", // 项目配置覆盖
				"terraform": "1.5.0",  // 全局配置
				"sqlc":      "1.19.0", // 工具信息
				"helm":      "3.12.0", // 项目特有
			},
			sources: map[string]string{
				"kubectl":   "project",
				"terraform": "global",
				"sqlc":      "global_tool",
				"helm":      "project",
			},
		},
		{
			name:     "merge strategy",
			strategy: types.MergeStrategy,
			expected: map[string]string{
				"kubectl":   "1.29.0", // 项目配置优先
				"terraform": "1.5.0",  // 全局配置
				"sqlc":      "1.19.0", // 工具信息
				"helm":      "3.12.0", // 项目特有
			},
			sources: map[string]string{
				"kubectl":   "project",
				"terraform": "global",
				"sqlc":      "global_tool",
				"helm":      "project",
			},
		},
		{
			name:     "ignore strategy",
			strategy: types.IgnoreStrategy,
			expected: map[string]string{
				"kubectl":   "1.28.0", // 只使用全局配置
				"terraform": "1.5.0",  // 全局配置
				"sqlc":      "1.19.0", // 工具信息
			},
			sources: map[string]string{
				"kubectl":   "global",
				"terraform": "global",
				"sqlc":      "global_tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective, err := merger.MergeConfigs(globalConfig, projectConfig, tt.strategy)
			require.NoError(t, err)
			require.NotNil(t, effective)

			// 验证解析的版本
			for tool, expectedVersion := range tt.expected {
				actualVersion, exists := effective.ResolvedVersions[tool]
				assert.True(t, exists, "Tool %s should exist in resolved versions", tool)
				assert.Equal(t, expectedVersion, actualVersion, "Version mismatch for tool %s", tool)
			}

			// 验证配置来源
			for tool, expectedSource := range tt.sources {
				actualSource, exists := effective.ConfigSource[tool]
				assert.True(t, exists, "Tool %s should exist in config source", tool)
				assert.Equal(t, expectedSource, actualSource, "Source mismatch for tool %s", tool)
			}

			// 验证配置引用
			assert.Equal(t, globalConfig, effective.Global)
			assert.Equal(t, projectConfig, effective.Project)
		})
	}
}

func TestDefaultMerger_MergeConfigs_NilProject(t *testing.T) {
	merger := NewMerger()

	globalConfig := &types.GlobalConfig{
		Version: "1.0",
		Settings: types.Settings{
			Download: types.DownloadSettings{
				Timeout:             300 * time.Second,
				Retries:             3,
				ConcurrentDownloads: 2,
			},
			Logging: types.LoggingSettings{
				Level: "info",
				File:  "~/.vman/logs/vman.log",
			},
		},
		GlobalVersions: map[string]string{
			"kubectl": "1.28.0",
		},
		Tools: map[string]types.ToolInfo{},
	}

	// 测试项目配置为nil的情况
	effective, err := merger.MergeConfigs(globalConfig, nil, types.OverrideStrategy)
	require.NoError(t, err)
	require.NotNil(t, effective)

	// 应该只包含全局配置的工具
	assert.Equal(t, "1.28.0", effective.ResolvedVersions["kubectl"])
	assert.Equal(t, "global", effective.ConfigSource["kubectl"])

	// 项目配置应该是默认配置
	assert.NotNil(t, effective.Project)
	assert.Equal(t, "1.0", effective.Project.Version)
}

func TestDefaultMerger_MergeConfigs_NilGlobal(t *testing.T) {
	merger := NewMerger()

	// 测试全局配置为nil的情况（应该返回错误）
	effective, err := merger.MergeConfigs(nil, &types.ProjectConfig{}, types.OverrideStrategy)
	assert.Error(t, err)
	assert.Nil(t, effective)
}

func TestDefaultMerger_ResolveVersion(t *testing.T) {
	merger := NewMerger()

	metadata := &types.ToolMetadata{
		Name: "kubectl",
		VersionConfig: types.VersionConfig{
			Aliases: map[string]string{
				"latest": "1.29.0",
				"stable": "1.28.0",
			},
			Constraints: types.VersionConstraints{
				MinVersion: "1.20.0",
				MaxVersion: "1.30.0",
			},
		},
	}

	tests := []struct {
		name             string
		toolName         string
		requestedVersion string
		metadata         *types.ToolMetadata
		expectedResolved string
		expectedSource   string
		expectError      bool
	}{
		{
			name:             "direct version",
			toolName:         "kubectl",
			requestedVersion: "1.25.0",
			metadata:         metadata,
			expectedResolved: "1.25.0",
			expectedSource:   "direct",
			expectError:      false,
		},
		{
			name:             "alias resolution",
			toolName:         "kubectl",
			requestedVersion: "latest",
			metadata:         metadata,
			expectedResolved: "1.29.0",
			expectedSource:   "alias",
			expectError:      false,
		},
		{
			name:             "stable alias",
			toolName:         "kubectl",
			requestedVersion: "stable",
			metadata:         metadata,
			expectedResolved: "1.28.0",
			expectedSource:   "alias",
			expectError:      false,
		},
		{
			name:             "no metadata",
			toolName:         "kubectl",
			requestedVersion: "1.25.0",
			metadata:         nil,
			expectedResolved: "1.25.0",
			expectedSource:   "direct",
			expectError:      false,
		},
		{
			name:             "version below minimum",
			toolName:         "kubectl",
			requestedVersion: "1.19.0",
			metadata:         metadata,
			expectedResolved: "",
			expectedSource:   "",
			expectError:      true,
		},
		{
			name:             "version above maximum",
			toolName:         "kubectl",
			requestedVersion: "1.31.0",
			metadata:         metadata,
			expectedResolved: "",
			expectedSource:   "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := merger.ResolveVersion(tt.toolName, tt.requestedVersion, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resolution)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resolution)

				assert.Equal(t, tt.toolName, resolution.ToolName)
				assert.Equal(t, tt.requestedVersion, resolution.RequestedVersion)
				assert.Equal(t, tt.expectedResolved, resolution.ResolvedVersion)
				assert.Equal(t, tt.expectedSource, resolution.Source)
			}
		})
	}
}

func TestDefaultMerger_GetVersionSource(t *testing.T) {
	merger := NewMerger()

	globalConfig := &types.GlobalConfig{
		GlobalVersions: map[string]string{
			"kubectl":   "1.28.0",
			"terraform": "1.5.0",
		},
		Tools: map[string]types.ToolInfo{
			"sqlc": {
				CurrentVersion:    "1.19.0",
				InstalledVersions: []string{"1.19.0"},
			},
		},
	}

	projectConfig := &types.ProjectConfig{
		Tools: map[string]string{
			"kubectl": "1.29.0", // 覆盖全局版本
			"helm":    "3.12.0", // 项目特有
		},
	}

	tests := []struct {
		name            string
		toolName        string
		expectedVersion string
		expectedSource  string
	}{
		{
			name:            "project config override",
			toolName:        "kubectl",
			expectedVersion: "1.29.0",
			expectedSource:  "project",
		},
		{
			name:            "global version",
			toolName:        "terraform",
			expectedVersion: "1.5.0",
			expectedSource:  "global",
		},
		{
			name:            "tool info",
			toolName:        "sqlc",
			expectedVersion: "1.19.0",
			expectedSource:  "tool_info",
		},
		{
			name:            "project only",
			toolName:        "helm",
			expectedVersion: "3.12.0",
			expectedSource:  "project",
		},
		{
			name:            "not found",
			toolName:        "nonexistent",
			expectedVersion: "",
			expectedSource:  "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, source := merger.GetVersionSource(tt.toolName, globalConfig, projectConfig)
			assert.Equal(t, tt.expectedVersion, version)
			assert.Equal(t, tt.expectedSource, source)
		})
	}
}

func TestDefaultMerger_GetVersionSource_NilProject(t *testing.T) {
	merger := NewMerger()

	globalConfig := &types.GlobalConfig{
		GlobalVersions: map[string]string{
			"kubectl": "1.28.0",
		},
		Tools: map[string]types.ToolInfo{},
	}

	// 测试项目配置为nil的情况
	version, source := merger.GetVersionSource("kubectl", globalConfig, nil)
	assert.Equal(t, "1.28.0", version)
	assert.Equal(t, "global", source)
}

func TestAdvancedMerger_MergeConfigs(t *testing.T) {
	validator := NewValidator()
	merger := NewAdvancedMerger(validator)

	validGlobalConfig := &types.GlobalConfig{
		Version: "1.0",
		Settings: types.Settings{
			Download: types.DownloadSettings{
				Timeout:             300 * time.Second,
				Retries:             3,
				ConcurrentDownloads: 2,
			},
			Logging: types.LoggingSettings{
				Level: "info",
				File:  "~/.vman/logs/vman.log",
			},
		},
		GlobalVersions: map[string]string{
			"kubectl": "1.28.0",
		},
		Tools: map[string]types.ToolInfo{},
	}

	validProjectConfig := &types.ProjectConfig{
		Version: "1.0",
		Tools: map[string]string{
			"kubectl": "1.29.0",
		},
	}

	// 测试有效配置
	effective, err := merger.MergeConfigs(validGlobalConfig, validProjectConfig, types.OverrideStrategy)
	assert.NoError(t, err)
	assert.NotNil(t, effective)

	// 测试无效全局配置
	invalidGlobalConfig := &types.GlobalConfig{
		Version: "", // 无效版本
	}

	effective, err = merger.MergeConfigs(invalidGlobalConfig, validProjectConfig, types.OverrideStrategy)
	assert.Error(t, err)
	assert.Nil(t, effective)

	// 测试无效项目配置
	invalidProjectConfig := &types.ProjectConfig{
		Version: "", // 无效版本
	}

	effective, err = merger.MergeConfigs(validGlobalConfig, invalidProjectConfig, types.OverrideStrategy)
	assert.Error(t, err)
	assert.Nil(t, effective)
}
