# vman 贡献指南

欢迎为 vman 项目做出贡献！本指南将帮助你了解如何参与 vman 的开发。

## 📋 目录

- [🎯 贡献方式](#-贡献方式)
- [🛠️ 开发环境搭建](#️-开发环境搭建)
- [📝 代码规范](#-代码规范)
- [🧪 测试指南](#-测试指南)
- [📖 文档规范](#-文档规范)
- [🔄 提交流程](#-提交流程)
- [🏗️ 项目结构](#️-项目结构)
- [🔍 调试指南](#-调试指南)
- [📦 发布流程](#-发布流程)

## 🎯 贡献方式

### 报告问题
- 🐛 [报告 Bug](https://github.com/songzhibin97/vman/issues/new?template=bug_report.md)
- 💡 [功能请求](https://github.com/songzhibin97/vman/issues/new?template=feature_request.md)
- 📚 [文档改进](https://github.com/songzhibin97/vman/issues/new?template=documentation.md)

### 贡献代码
- 🔧 修复 Bug
- ✨ 添加新功能
- 📈 性能优化
- 🧹 代码重构

### 贡献文档
- 📝 改进现有文档
- 🌍 翻译文档
- 📹 创建教程视频

## 🛠️ 开发环境搭建

### 系统要求

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 编程语言 |
| Git | 2.0+ | 版本控制 |
| Make | 4.0+ | 构建工具 |
| golangci-lint | 1.54+ | 代码检查 |

### 环境搭建步骤

#### 1. 克隆仓库

```bash
# 克隆主仓库
git clone https://github.com/songzhibin97/vman.git
cd vman

# 添加你的 fork 作为远程仓库
git remote add fork https://github.com/YOUR_USERNAME/vman.git
```

#### 2. 安装依赖

```bash
# 安装 Go 依赖
go mod download

# 安装开发工具
make dev-setup

# 验证安装
go version
golangci-lint version
```

#### 3. 验证环境

```bash
# 运行测试
make test

# 检查代码质量
make lint

# 构建项目
make build

# 运行程序
./build/vman --version
```

### IDE 配置

#### VS Code 推荐配置

创建 `.vscode/settings.json`：

```json
{
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast"
    ],
    "go.testFlags": [
        "-v",
        "-race"
    ],
    "go.buildFlags": [
        "-v"
    ],
    "editor.formatOnSave": true,
    "go.formatTool": "goimports"
}
```

推荐扩展：
- `golang.go` - Go 语言支持
- `ms-vscode.vscode-json` - JSON 支持
- `redhat.vscode-yaml` - YAML 支持

#### GoLand/IntelliJ 配置

1. 导入项目
2. 启用 `gofmt` 和 `goimports`
3. 配置 `golangci-lint` 作为外部工具
4. 设置代码模板和风格

## 📝 代码规范

### Go 代码规范

#### 1. 命名规范

```go
// ✅ 好的命名
type ConfigManager interface {}
type HTTPClient struct {}
func NewConfigManager() ConfigManager {}
func (c *ConfigManager) LoadGlobal() error {}

// ❌ 不好的命名
type configmgr interface {}
type httpclient struct {}
func newConfigMgr() configmgr {}
func (c *configmgr) loadGlobal() error {}
```

#### 2. 注释规范

```go
// ✅ 好的注释
// ConfigManager 管理 vman 的配置文件，包括全局配置和项目配置。
// 它提供了加载、保存、验证配置的功能。
type ConfigManager interface {
    // LoadGlobal 从 ~/.vman/config.yaml 加载全局配置。
    // 如果文件不存在，返回默认配置。
    LoadGlobal() (*GlobalConfig, error)
}

// ❌ 不好的注释
// config manager
type ConfigManager interface {
    // load global config
    LoadGlobal() (*GlobalConfig, error)
}
```

#### 3. 错误处理

```go
// ✅ 好的错误处理
func (c *ConfigManager) LoadGlobal() (*GlobalConfig, error) {
    data, err := os.ReadFile(c.globalConfigPath)
    if err != nil {
        if os.IsNotExist(err) {
            // 文件不存在时返回默认配置
            return GetDefaultGlobalConfig(), nil
        }
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    var config GlobalConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    return &config, nil
}

// ❌ 不好的错误处理
func (c *ConfigManager) LoadGlobal() (*GlobalConfig, error) {
    data, _ := os.ReadFile(c.globalConfigPath)
    var config GlobalConfig
    yaml.Unmarshal(data, &config)
    return &config, nil
}
```

#### 4. 包结构

```go
// ✅ 好的包结构
package config

import (
    "fmt"
    "os"
    
    "gopkg.in/yaml.v3"
    
    "github.com/songzhibin97/vman/pkg/types"
)

// ❌ 不好的包结构
package config

import (
    "fmt"
    "gopkg.in/yaml.v3"
    "github.com/songzhibin97/vman/pkg/types"
    "os"
)
```

### 代码质量检查

#### 运行检查工具

```bash
# 格式化代码
make fmt

# 运行 linter
make lint

# 运行测试
make test

# 生成测试覆盖率
make coverage
```

#### 预提交钩子

设置 git 预提交钩子：

```bash
# 创建预提交钩子
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

# 运行格式化
make fmt

# 检查是否有未提交的格式化变更
if ! git diff --quiet; then
    echo "Code formatting changed files. Please add them and commit again."
    exit 1
fi

# 运行 linter
make lint

# 运行测试
make test
EOF

chmod +x .git/hooks/pre-commit
```

## 🧪 测试指南

### 测试类型

#### 1. 单元测试

```go
// internal/config/manager_test.go
func TestConfigManager_LoadGlobal(t *testing.T) {
    tests := []struct {
        name           string
        configContent  string
        expectedConfig *GlobalConfig
        expectError    bool
    }{
        {
            name: "valid config",
            configContent: `
version: "1.0"
settings:
  download:
    timeout: 300s
`,
            expectedConfig: &GlobalConfig{
                Version: "1.0",
                Settings: Settings{
                    Download: DownloadSettings{
                        Timeout: 300 * time.Second,
                    },
                },
            },
            expectError: false,
        },
        {
            name:          "invalid yaml",
            configContent: "invalid: yaml: content:",
            expectError:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 创建临时配置文件
            tmpFile := createTempConfig(t, tt.configContent)
            defer os.Remove(tmpFile)

            // 创建配置管理器
            manager := NewConfigManager(tmpFile)

            // 加载配置
            config, err := manager.LoadGlobal()

            // 验证结果
            if tt.expectError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedConfig, config)
            }
        })
    }
}
```

#### 2. 集成测试

```go
// test/integration/e2e_workflow_test.go
func TestE2EWorkflow(t *testing.T) {
    // 创建临时目录
    tmpDir := t.TempDir()
    
    // 设置测试环境
    vman := setupTestVman(t, tmpDir)
    
    // 测试完整工作流
    t.Run("install and use tool", func(t *testing.T) {
        // 添加工具
        err := vman.AddTool("kubectl")
        require.NoError(t, err)
        
        // 安装版本
        err = vman.InstallVersion("kubectl", "1.28.0")
        require.NoError(t, err)
        
        // 设置全局版本
        err = vman.SetGlobalVersion("kubectl", "1.28.0")
        require.NoError(t, err)
        
        // 验证版本
        version, err := vman.GetCurrentVersion("kubectl", "")
        require.NoError(t, err)
        assert.Equal(t, "1.28.0", version)
    })
}
```

#### 3. 性能测试

```go
// internal/download/manager_benchmark_test.go
func BenchmarkDownloadManager_Download(b *testing.B) {
    manager := setupTestDownloadManager(b)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := manager.Download("test-tool", "1.0.0")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Mock 和测试工具

```go
// test/testutils/mock_services.go
//go:generate mockgen -source=../../internal/config/manager.go -destination=mock_config_manager.go

// 使用 mock
func TestVersionManager_InstallVersion(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockConfig := NewMockConfigManager(ctrl)
    mockDownload := NewMockDownloadManager(ctrl)

    // 设置期望
    mockConfig.EXPECT().
        LoadGlobal().
        Return(&GlobalConfig{}, nil)

    // 创建版本管理器
    manager := NewVersionManager(mockConfig, mockDownload)

    // 执行测试
    err := manager.InstallVersion("kubectl", "1.28.0")
    assert.NoError(t, err)
}
```

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test ./internal/config/...

# 运行特定测试
go test -run TestConfigManager_LoadGlobal ./internal/config/

# 运行基准测试
go test -bench=. ./internal/download/

# 生成测试覆盖率
make coverage
```

## 📖 文档规范

### Markdown 规范

#### 1. 文档结构

```markdown
# 标题

简短的介绍段落。

## 📋 目录

- [🎯 主要内容](#-主要内容)
- [🛠️ 其他内容](#️-其他内容)

## 🎯 主要内容

### 子标题

内容段落。

#### 更小的标题

详细内容。

---

## 📚 相关文档

- [链接1](link1.md)
- [链接2](link2.md)
```

#### 2. 代码示例

````markdown
# 代码块应该包含语言标识
```bash
# 这是一个 bash 命令
vman install kubectl 1.28.0
```

```go
// 这是 Go 代码
func main() {
    fmt.Println("Hello, World!")
}
```

# 行内代码使用单引号
使用 `vman --version` 检查版本。
````

#### 3. 表格格式

```markdown
| 列1 | 列2 | 列3 |
|-----|-----|-----|
| 值1 | 值2 | 值3 |
| 值4 | 值5 | 值6 |
```

### API 文档规范

使用 godoc 格式编写 API 文档：

```go
// Package config 提供 vman 的配置管理功能。
//
// 该包实现了全局配置和项目配置的加载、保存、合并等功能。
// 支持 YAML 格式的配置文件，并提供配置验证和迁移功能。
//
// 基本用法：
//
//     manager := config.NewManager()
//     globalConfig, err := manager.LoadGlobal()
//     if err != nil {
//         log.Fatal(err)
//     }
//
package config

// Manager 定义了配置管理器的接口。
//
// 配置管理器负责处理 vman 的所有配置相关操作，包括：
//   - 加载和保存全局配置
//   - 加载和保存项目配置
//   - 配置合并和验证
//   - 配置迁移
type Manager interface {
    // LoadGlobal 加载全局配置文件。
    //
    // 全局配置文件通常位于 ~/.vman/config.yaml。
    // 如果配置文件不存在，将返回默认配置。
    //
    // 返回值：
    //   - *GlobalConfig: 全局配置对象
    //   - error: 加载过程中的错误
    LoadGlobal() (*GlobalConfig, error)
}
```

## 🔄 提交流程

### Git 工作流

1. **创建功能分支**

```bash
# 确保在最新的 main 分支
git checkout main
git pull origin main

# 创建新的功能分支
git checkout -b feature/new-awesome-feature
```

2. **开发和测试**

```bash
# 进行开发...
# 编写测试...

# 运行测试
make test

# 检查代码质量
make lint
```

3. **提交代码**

```bash
# 添加变更
git add .

# 提交（遵循提交消息规范）
git commit -m "feat: add awesome new feature

- Add new configuration option
- Implement feature X
- Update documentation

Closes #123"
```

4. **推送和创建 PR**

```bash
# 推送到你的 fork
git push fork feature/new-awesome-feature

# 在 GitHub 上创建 Pull Request
```

### 提交消息规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<类型>[可选的作用域]: <描述>

[可选的正文]

[可选的脚注]
```

#### 类型

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建或工具相关

#### 示例

```bash
# 新功能
git commit -m "feat(config): add support for environment variables"

# Bug 修复
git commit -m "fix(download): handle network timeout properly"

# 文档更新
git commit -m "docs: update installation guide"

# 破坏性变更
git commit -m "feat!: change config file format

BREAKING CHANGE: config format changed from TOML to YAML"
```

### Pull Request 规范

#### PR 标题
- 使用清晰描述性的标题
- 包含相关的 issue 编号

#### PR 描述模板

```markdown
## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 破坏性变更
- [ ] 文档更新

## 描述
简要描述这个 PR 的变更内容。

## 测试
- [ ] 我已经测试了这些变更
- [ ] 我已经添加了适当的测试
- [ ] 所有测试都通过

## 相关 Issue
Closes #123

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 我已经进行了自我代码审查
- [ ] 我已经添加了必要的文档
- [ ] 我的变更没有产生新的警告
```

## 🏗️ 项目结构

```
vman/
├── cmd/vman/           # 主程序入口
│   └── main.go
├── internal/           # 内部包
│   ├── cli/           # CLI 命令实现
│   ├── config/        # 配置管理
│   ├── version/       # 版本管理
│   ├── download/      # 下载管理
│   ├── proxy/         # 命令代理
│   └── storage/       # 存储管理
├── pkg/               # 公共包
│   ├── types/         # 类型定义
│   └── utils/         # 工具函数
├── test/              # 测试
│   ├── functional/    # 功能测试
│   ├── integration/   # 集成测试
│   └── testutils/     # 测试工具
├── docs/              # 文档
├── configs/           # 配置文件模板
├── scripts/           # 脚本文件
├── Makefile           # 构建脚本
├── go.mod             # Go 模块文件
└── README.md          # 项目说明
```

### 代码组织原则

- `internal/` - 内部实现，不对外暴露
- `pkg/` - 可被外部引用的公共包
- `cmd/` - 可执行程序的入口点
- `test/` - 测试相关代码

## 🔍 调试指南

### 本地调试

#### 1. 使用调试构建

```bash
# 构建调试版本
go build -gcflags="all=-N -l" -o build/vman-debug ./cmd/vman

# 使用 delve 调试
dlv exec ./build/vman-debug -- --help
```

#### 2. 启用详细日志

```bash
# 设置环境变量
export VMAN_LOG_LEVEL=debug
export VMAN_LOG_FILE=/tmp/vman-debug.log

# 运行程序
./build/vman install kubectl 1.28.0

# 查看日志
tail -f /tmp/vman-debug.log
```

#### 3. 使用测试模式

```bash
# 在测试目录中运行
export VMAN_CONFIG_DIR=/tmp/vman-test
./build/vman init
./build/vman add kubectl
```

### 远程调试

```bash
# 在服务器上启动 delve 服务器
dlv exec ./vman --headless --listen=:2345 --api-version=2

# 在本地连接
dlv connect :2345
```

## 📦 发布流程

### 版本号规范

使用 [Semantic Versioning](https://semver.org/)：

- `MAJOR.MINOR.PATCH` (例如: 1.2.3)
- `MAJOR`: 破坏性变更
- `MINOR`: 新功能（向后兼容）
- `PATCH`: Bug 修复（向后兼容）

### 发布步骤

1. **准备发布**

```bash
# 更新版本号
vim internal/version/info.go

# 更新 CHANGELOG
vim CHANGELOG.md

# 运行完整测试
make test-all
```

2. **创建发布标签**

```bash
# 提交版本更新
git add .
git commit -m "chore: bump version to v1.2.3"

# 创建标签
git tag -a v1.2.3 -m "Release v1.2.3"

# 推送标签
git push origin v1.2.3
```

3. **自动构建发布**

GitHub Actions 会自动：
- 构建多平台二进制文件
- 运行测试
- 创建 GitHub Release
- 上传构建产物

---

## 🤝 社区准则

### 行为准则

- 🤝 友善和尊重
- 💬 建设性的讨论
- 🎯 关注技术问题
- 📚 分享知识和经验

### 获取帮助

- 📖 查看 [文档](https://github.com/songzhibin97/vman/docs)
- 💬 参与 [Discussions](https://github.com/songzhibin97/vman/discussions)
- 🐛 报告 [Issues](https://github.com/songzhibin97/vman/issues)
- 📧 联系维护者：songzhibin97@gmail.com

感谢你的贡献！🎉