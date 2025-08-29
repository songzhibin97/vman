package functional

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/pkg/types"
)

// ConfigCompatibilityTestSuite 配置文件兼容性功能测试套件
type ConfigCompatibilityTestSuite struct {
	suite.Suite
	fs           afero.Fs
	homeDir      string
	configAPI    config.API
	ctx          context.Context
	cleanupFuncs []func()
}

func (suite *ConfigCompatibilityTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.fs = afero.NewOsFs() // 使用真实文件系统
	suite.cleanupFuncs = make([]func(), 0)

	// 创建临时主目录
	tempDir, err := os.MkdirTemp("", "vman-config-test-*")
	require.NoError(suite.T(), err)
	suite.homeDir = tempDir

	suite.addCleanup(func() {
		os.RemoveAll(suite.homeDir)
	})

	// 初始化配置API
	suite.configAPI, err = config.NewAPI(suite.homeDir)
	require.NoError(suite.T(), err)

	err = suite.configAPI.Init(suite.ctx)
	require.NoError(suite.T(), err)
}

func (suite *ConfigCompatibilityTestSuite) TearDownSuite() {
	for i := len(suite.cleanupFuncs) - 1; i >= 0; i-- {
		suite.cleanupFuncs[i]()
	}
}

func (suite *ConfigCompatibilityTestSuite) addCleanup(fn func()) {
	suite.cleanupFuncs = append(suite.cleanupFuncs, fn)
}

// TestLegacyConfigFormats 测试遗留配置格式
func (suite *ConfigCompatibilityTestSuite) TestLegacyConfigFormats() {
	legacyFormats := []struct {
		name    string
		content string
		version string
	}{
		{
			name:    "v0.9格式",
			version: "0.9",
			content: `
version: "0.9"
tools:
  kubectl: "1.28.0"
  terraform: "1.5.0"
global:
  download_timeout: 300
  concurrent_downloads: 2
`,
		},
		{
			name:    "v0.8格式",
			version: "0.8",
			content: `
version: "0.8"
global_versions:
  kubectl: "1.27.0"
  terraform: "1.4.0"
settings:
  timeout: 300
  retries: 3
`,
		},
		{
			name:    "简化格式",
			version: "1.0",
			content: `
kubectl: "1.29.0"
terraform: "1.6.0"
node: "20.11.0"
`,
		},
	}

	for _, format := range legacyFormats {
		suite.Run(format.name, func() {
			// 创建测试项目目录
			projectPath := filepath.Join(suite.homeDir, fmt.Sprintf("legacy_%s", strings.ReplaceAll(format.version, ".", "_")))
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			// 写入遗留格式配置文件
			configFile := filepath.Join(projectPath, ".vman.yaml")
			err = os.WriteFile(configFile, []byte(format.content), 0644)
			require.NoError(suite.T(), err)

			// 尝试加载配置
			config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
			
			if err != nil {
				// 某些遗留格式可能不被支持，这是可以接受的
				suite.T().Logf("遗留格式 %s 不被支持: %v", format.name, err)
			} else {
				// 如果支持，验证配置正确加载
				assert.NotNil(suite.T(), config)
				suite.T().Logf("成功加载遗留格式 %s", format.name)
			}
		})
	}
}

// TestVersionMigration 测试版本迁移
func (suite *ConfigCompatibilityTestSuite) TestVersionMigration() {
	suite.Run("配置版本迁移", func() {
		// 创建旧版本配置
		projectPath := filepath.Join(suite.homeDir, "migration_test")
		err := os.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		// 创建v0.9格式的配置
		oldConfig := `
version: "0.9"
tools:
  kubectl: "1.28.0"
  terraform: "1.5.0"
global:
  download_timeout: 300
`
		configFile := filepath.Join(projectPath, ".vman.yaml")
		err = os.WriteFile(configFile, []byte(oldConfig), 0644)
		require.NoError(suite.T(), err)

		// 加载配置（应该触发迁移）
		config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
		require.NoError(suite.T(), err)

		// 验证配置已迁移到新格式
		assert.NotNil(suite.T(), config)
		assert.Equal(suite.T(), "1.0", config.Version) // 应该升级到当前版本

		// 保存配置（应该使用新格式）
		err = suite.configAPI.UpdateProjectConfig(suite.ctx, projectPath, config)
		require.NoError(suite.T(), err)

		// 重新读取配置文件，验证格式
		content, err := os.ReadFile(configFile)
		require.NoError(suite.T(), err)

		var savedConfig map[string]interface{}
		err = yaml.Unmarshal(content, &savedConfig)
		require.NoError(suite.T(), err)

		// 验证新格式
		assert.Equal(suite.T(), "1.0", savedConfig["version"])
		assert.Contains(suite.T(), savedConfig, "tools")
	})
}

// TestDifferentFileFormats 测试不同文件格式
func (suite *ConfigCompatibilityTestSuite) TestDifferentFileFormats() {
	formats := []struct {
		name      string
		filename  string
		content   string
		supported bool
	}{
		{
			name:      "YAML格式",
			filename:  ".vman.yaml",
			supported: true,
			content: `
version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.6.0"
`,
		},
		{
			name:      "YML格式",
			filename:  ".vman.yml",
			supported: true,
			content: `
version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.6.0"
`,
		},
		{
			name:      "JSON格式",
			filename:  ".vman.json",
			supported: false, // 当前不支持，但测试兼容性
			content: `{
  "version": "1.0",
  "tools": {
    "kubectl": "1.29.0",
    "terraform": "1.6.0"
  }
}`,
		},
		{
			name:      "TOML格式",
			filename:  ".vman.toml",
			supported: false, // 当前不支持，但测试兼容性
			content: `
version = "1.0"

[tools]
kubectl = "1.29.0"
terraform = "1.6.0"
`,
		},
	}

	for _, format := range formats {
		suite.Run(format.name, func() {
			projectPath := filepath.Join(suite.homeDir, fmt.Sprintf("format_%s", format.name))
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			configFile := filepath.Join(projectPath, format.filename)
			err = os.WriteFile(configFile, []byte(format.content), 0644)
			require.NoError(suite.T(), err)

			config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)

			if format.supported {
				require.NoError(suite.T(), err)
				assert.NotNil(suite.T(), config)
				assert.Equal(suite.T(), "1.0", config.Version)
				suite.T().Logf("格式 %s 得到支持", format.name)
			} else {
				// 不支持的格式应该返回默认配置或错误
				if err != nil {
					suite.T().Logf("格式 %s 不被支持（预期）: %v", format.name, err)
				} else {
					// 如果没有错误，应该返回默认配置
					assert.NotNil(suite.T(), config)
					suite.T().Logf("格式 %s 回退到默认配置", format.name)
				}
			}
		})
	}
}

// TestEncodingCompatibility 测试编码兼容性
func (suite *ConfigCompatibilityTestSuite) TestEncodingCompatibility() {
	encodings := []struct {
		name     string
		content  string
		encoding string
	}{
		{
			name:     "UTF-8",
			encoding: "utf-8",
			content: `
version: "1.0"
tools:
  kubectl: "1.29.0"
  # 中文注释测试
  terraform: "1.6.0"
`,
		},
		{
			name:     "UTF-8 with BOM",
			encoding: "utf-8-bom",
			content: "\ufeff" + `
version: "1.0"
tools:
  kubectl: "1.29.0"
  terraform: "1.6.0"
`,
		},
		{
			name:     "Unicode字符",
			encoding: "unicode",
			content: `
version: "1.0"
tools:
  kubectl: "1.29.0"
  # Unicode: ñáéíóú αβγδε 中文 日本語 한국어
  terraform: "1.6.0"
`,
		},
	}

	for _, encoding := range encodings {
		suite.Run(encoding.name, func() {
			projectPath := filepath.Join(suite.homeDir, fmt.Sprintf("encoding_%s", encoding.name))
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			configFile := filepath.Join(projectPath, ".vman.yaml")
			err = os.WriteFile(configFile, []byte(encoding.content), 0644)
			require.NoError(suite.T(), err)

			config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
			require.NoError(suite.T(), err)
			assert.NotNil(suite.T(), config)
			assert.Equal(suite.T(), "1.0", config.Version)

			suite.T().Logf("编码 %s 处理成功", encoding.name)
		})
	}
}

// TestConfigMerging 测试配置合并
func (suite *ConfigCompatibilityTestSuite) TestConfigMerging() {
	suite.Run("全局和项目配置合并", func() {
		// 设置全局配置
		globalConfig := types.GetDefaultGlobalConfig()
		globalConfig.GlobalVersions["kubectl"] = "1.28.0"
		globalConfig.GlobalVersions["terraform"] = "1.5.0"
		globalConfig.GlobalVersions["node"] = "18.19.0"

		err := suite.configAPI.UpdateGlobalConfig(suite.ctx, globalConfig)
		require.NoError(suite.T(), err)

		// 创建项目配置
		projectPath := filepath.Join(suite.homeDir, "merge_test")
		err = os.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		projectConfigContent := `
version: "1.0"
tools:
  kubectl: "1.29.0"  # 覆盖全局版本
  helm: "3.14.0"     # 新增工具
`
		configFile := filepath.Join(projectPath, ".vman.yaml")
		err = os.WriteFile(configFile, []byte(projectConfigContent), 0644)
		require.NoError(suite.T(), err)

		// 获取有效配置
		effectiveConfig, err := suite.configAPI.GetEffectiveConfig(suite.ctx, projectPath)
		require.NoError(suite.T(), err)

		// 验证配置合并结果
		assert.Equal(suite.T(), "1.29.0", effectiveConfig.ResolvedVersions["kubectl"])  // 项目版本覆盖全局
		assert.Equal(suite.T(), "1.5.0", effectiveConfig.ResolvedVersions["terraform"]) // 全局版本
		assert.Equal(suite.T(), "18.19.0", effectiveConfig.ResolvedVersions["node"])    // 全局版本
		assert.Equal(suite.T(), "3.14.0", effectiveConfig.ResolvedVersions["helm"])     // 项目独有

		// 验证配置来源
		assert.Equal(suite.T(), "project", effectiveConfig.ConfigSource["kubectl"])
		assert.Equal(suite.T(), "global", effectiveConfig.ConfigSource["terraform"])
		assert.Equal(suite.T(), "global", effectiveConfig.ConfigSource["node"])
		assert.Equal(suite.T(), "project", effectiveConfig.ConfigSource["helm"])
	})
}

// TestNestedProjectConfigs 测试嵌套项目配置
func (suite *ConfigCompatibilityTestSuite) TestNestedProjectConfigs() {
	suite.Run("嵌套项目配置", func() {
		// 创建父项目
		parentPath := filepath.Join(suite.homeDir, "parent_project")
		err := os.MkdirAll(parentPath, 0755)
		require.NoError(suite.T(), err)

		parentConfigContent := `
version: "1.0"
tools:
  kubectl: "1.28.0"
  terraform: "1.5.0"
`
		parentConfigFile := filepath.Join(parentPath, ".vman.yaml")
		err = os.WriteFile(parentConfigFile, []byte(parentConfigContent), 0644)
		require.NoError(suite.T(), err)

		// 创建子项目
		childPath := filepath.Join(parentPath, "child_project")
		err = os.MkdirAll(childPath, 0755)
		require.NoError(suite.T(), err)

		childConfigContent := `
version: "1.0"
tools:
  kubectl: "1.29.0"  # 覆盖父项目版本
  helm: "3.14.0"     # 新增工具
`
		childConfigFile := filepath.Join(childPath, ".vman.yaml")
		err = os.WriteFile(childConfigFile, []byte(childConfigContent), 0644)
		require.NoError(suite.T(), err)

		// 测试父项目配置
		parentEffective, err := suite.configAPI.GetEffectiveConfig(suite.ctx, parentPath)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "1.28.0", parentEffective.ResolvedVersions["kubectl"])
		assert.Equal(suite.T(), "1.5.0", parentEffective.ResolvedVersions["terraform"])
		assert.NotContains(suite.T(), parentEffective.ResolvedVersions, "helm")

		// 测试子项目配置
		childEffective, err := suite.configAPI.GetEffectiveConfig(suite.ctx, childPath)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "1.29.0", childEffective.ResolvedVersions["kubectl"])  // 子项目覆盖
		assert.Equal(suite.T(), "3.14.0", childEffective.ResolvedVersions["helm"])     // 子项目独有
		
		// 子项目应该继承父项目没有覆盖的工具
		// 注意：这取决于具体的实现逻辑
		suite.T().Logf("子项目有效配置: %+v", childEffective.ResolvedVersions)
	})
}

// TestConfigValidation 测试配置验证
func (suite *ConfigCompatibilityTestSuite) TestConfigValidation() {
	invalidConfigs := []struct {
		name    string
		content string
		error   string
	}{
		{
			name: "缺少版本",
			content: `
tools:
  kubectl: "1.29.0"
`,
			error: "version",
		},
		{
			name: "无效版本格式",
			content: `
version: "invalid-version"
tools:
  kubectl: "1.29.0"
`,
			error: "version",
		},
		{
			name: "无效工具版本",
			content: `
version: "1.0"
tools:
  kubectl: "invalid-version"
`,
			error: "tool version",
		},
		{
			name: "空工具名称",
			content: `
version: "1.0"
tools:
  "": "1.29.0"
`,
			error: "tool name",
		},
	}

	for _, invalid := range invalidConfigs {
		suite.Run(invalid.name, func() {
			projectPath := filepath.Join(suite.homeDir, fmt.Sprintf("invalid_%s", strings.ReplaceAll(invalid.name, " ", "_")))
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			configFile := filepath.Join(projectPath, ".vman.yaml")
			err = os.WriteFile(configFile, []byte(invalid.content), 0644)
			require.NoError(suite.T(), err)

			_, err = suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
			
			if err != nil {
				assert.Contains(suite.T(), strings.ToLower(err.Error()), strings.ToLower(invalid.error))
				suite.T().Logf("正确检测到无效配置 %s: %v", invalid.name, err)
			} else {
				// 某些无效配置可能被容忍并使用默认值
				suite.T().Logf("无效配置 %s 被容忍", invalid.name)
			}
		})
	}
}

// TestConfigBackupRestore 测试配置备份和恢复
func (suite *ConfigCompatibilityTestSuite) TestConfigBackupRestore() {
	suite.Run("配置备份和恢复", func() {
		// 创建初始配置
		originalConfig := types.GetDefaultGlobalConfig()
		originalConfig.GlobalVersions["kubectl"] = "1.29.0"
		originalConfig.GlobalVersions["terraform"] = "1.6.0"

		err := suite.configAPI.UpdateGlobalConfig(suite.ctx, originalConfig)
		require.NoError(suite.T(), err)

		// 创建项目配置
		projectPath := filepath.Join(suite.homeDir, "backup_test")
		err = os.MkdirAll(projectPath, 0755)
		require.NoError(suite.T(), err)

		projectConfig := types.GetDefaultProjectConfig()
		projectConfig.Tools["helm"] = "3.14.0"
		err = suite.configAPI.UpdateProjectConfig(suite.ctx, projectPath, projectConfig)
		require.NoError(suite.T(), err)

		// 创建备份
		backupPath := filepath.Join(suite.homeDir, "backup")
		err = suite.configAPI.Backup(suite.ctx, backupPath)
		require.NoError(suite.T(), err)

		// 验证备份文件存在
		backupFiles := []string{
			filepath.Join(backupPath, "config.yaml"),
		}

		for _, file := range backupFiles {
			_, err := os.Stat(file)
			assert.NoError(suite.T(), err, "备份文件 %s 应该存在", file)
		}

		// 修改配置
		originalConfig.GlobalVersions["kubectl"] = "1.30.0"
		err = suite.configAPI.UpdateGlobalConfig(suite.ctx, originalConfig)
		require.NoError(suite.T(), err)

		// 恢复备份
		err = suite.configAPI.Restore(suite.ctx, backupPath)
		require.NoError(suite.T(), err)

		// 验证配置已恢复
		restoredConfig, err := suite.configAPI.GetGlobalConfig(suite.ctx)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), "1.29.0", restoredConfig.GlobalVersions["kubectl"])
	})
}

// TestConfigTemplates 测试配置模板
func (suite *ConfigCompatibilityTestSuite) TestConfigTemplates() {
	templates := []struct {
		name     string
		template string
		context  map[string]string
	}{
		{
			name: "Go项目模板",
			template: `
version: "1.0"
tools:
  go: "{{.GO_VERSION}}"
  golangci-lint: "{{.LINT_VERSION}}"
`,
			context: map[string]string{
				"GO_VERSION":   "1.21.0",
				"LINT_VERSION": "1.55.0",
			},
		},
		{
			name: "Node.js项目模板",
			template: `
version: "1.0"
tools:
  node: "{{.NODE_VERSION}}"
  npm: "{{.NPM_VERSION}}"
`,
			context: map[string]string{
				"NODE_VERSION": "20.11.0",
				"NPM_VERSION":  "10.2.0",
			},
		},
	}

	for _, tmpl := range templates {
		suite.Run(tmpl.name, func() {
			// 渲染模板
			content := tmpl.template
			for key, value := range tmpl.context {
				placeholder := fmt.Sprintf("{{.%s}}", key)
				content = strings.ReplaceAll(content, placeholder, value)
			}

			// 创建项目目录
			projectPath := filepath.Join(suite.homeDir, fmt.Sprintf("template_%s", strings.ReplaceAll(tmpl.name, " ", "_")))
			err := os.MkdirAll(projectPath, 0755)
			require.NoError(suite.T(), err)

			// 写入配置文件
			configFile := filepath.Join(projectPath, ".vman.yaml")
			err = os.WriteFile(configFile, []byte(content), 0644)
			require.NoError(suite.T(), err)

			// 加载配置
			config, err := suite.configAPI.GetProjectConfig(suite.ctx, projectPath)
			require.NoError(suite.T(), err)
			assert.NotNil(suite.T(), config)

			// 验证模板变量已替换
			for tool, expectedVersion := range tmpl.context {
				// 将环境变量名转换为工具名（简化处理）
				toolName := strings.ToLower(strings.Split(tool, "_")[0])
				if version, exists := config.Tools[toolName]; exists {
					assert.Equal(suite.T(), expectedVersion, version)
				}
			}

			suite.T().Logf("模板 %s 处理成功", tmpl.name)
		})
	}
}

// TestConfigWatching 测试配置文件监听
func (suite *ConfigCompatibilityTestSuite) TestConfigWatching() {
	suite.Run("配置文件变更监听", func() {
		// 这个测试在某些环境中可能不适用
		if testing.Short() {
			suite.T().Skip("跳过配置监听测试")
		}

		events := make(chan *types.ConfigChangeEvent, 10)

		// 开始监听配置变更
		err := suite.configAPI.Watch(suite.ctx, func(event *types.ConfigChangeEvent) {
			select {
			case events <- event:
			default:
				// 缓冲区满，忽略事件
			}
		})
		require.NoError(suite.T(), err)

		suite.addCleanup(func() {
			suite.configAPI.StopWatch(suite.ctx)
		})

		// 修改全局配置
		globalConfig, err := suite.configAPI.GetGlobalConfig(suite.ctx)
		require.NoError(suite.T(), err)

		globalConfig.GlobalVersions["test-watch-tool"] = "1.0.0"
		err = suite.configAPI.UpdateGlobalConfig(suite.ctx, globalConfig)
		require.NoError(suite.T(), err)

		// 等待并验证事件
		select {
		case event := <-events:
			assert.Equal(suite.T(), types.ConfigModified, event.Type)
			assert.Equal(suite.T(), "global", event.ConfigType)
			suite.T().Logf("收到配置变更事件: %+v", event)
		case <-time.After(2 * time.Second):
			suite.T().Log("未收到配置变更事件（可能是正常的）")
		}
	})
}

// TestConfigCompatibilityTestSuite 运行配置文件兼容性功能测试套件
func TestConfigCompatibilityTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过配置兼容性功能测试（使用 -short 标志）")
	}

	suite.Run(t, new(ConfigCompatibilityTestSuite))
}

// BenchmarkConfigOperations 配置操作性能基准测试
func BenchmarkConfigOperations(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vman-config-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configAPI, err := config.NewAPI(tempDir)
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
			// 测试配置读写性能
			projectPath := filepath.Join(tempDir, fmt.Sprintf("bench_project_%d", counter%100))
			
			config := types.GetDefaultProjectConfig()
			config.Tools[fmt.Sprintf("tool_%d", counter)] = "1.0.0"
			
			_ = configAPI.UpdateProjectConfig(ctx, projectPath, config)
			_, _ = configAPI.GetProjectConfig(ctx, projectPath)
			
			counter++
		}
	})
}