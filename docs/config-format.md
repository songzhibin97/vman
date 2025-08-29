# vman 配置文件格式文档

## 概述

vman 使用分层的配置系统来管理工具版本和系统设置。配置系统包含三个层级：

1. **全局配置** - 系统级别的默认设置和工具版本
2. **项目配置** - 项目级别的工具版本覆盖
3. **工具定义** - 工具的下载和版本管理元数据

## 配置文件位置

### 全局配置
- **路径**: `~/.vman/config.yaml`
- **格式**: YAML
- **用途**: 存储全局设置、默认工具版本和已安装工具信息

### 项目配置
- **路径**: `<项目根目录>/.vman.yaml`
- **格式**: YAML
- **用途**: 为特定项目指定工具版本，覆盖全局设置

### 工具定义
- **路径**: `~/.vman/tools/<工具名>.toml`
- **格式**: TOML
- **用途**: 定义工具的下载源、版本约束和元数据

## 配置文件目录结构

```
~/.vman/
├── config.yaml          # 全局配置文件
├── tools/               # 工具定义目录
│   ├── kubectl.toml     # kubectl 工具定义
│   ├── terraform.toml   # terraform 工具定义
│   └── sqlc.toml        # sqlc 工具定义
├── bin/                 # 工具二进制文件
├── shims/              # 工具代理脚本
├── versions/           # 版本存储目录
├── logs/               # 日志文件
├── cache/              # 缓存目录
└── tmp/                # 临时文件
```

## 全局配置文件 (config.yaml)

### 完整示例

```yaml
# vman 全局配置文件
version: "1.0"

# 全局设置
settings:
  # 下载设置
  download:
    timeout: 300s        # 下载超时时间 (1s - 30m)
    retries: 3           # 重试次数 (0 - 10)
    concurrent_downloads: 2  # 并发下载数 (1 - 10)
  
  # 代理设置
  proxy:
    enabled: true        # 启用命令代理
    shims_in_path: true  # 将shims目录添加到PATH
  
  # 日志设置
  logging:
    level: "info"        # 日志级别: debug, info, warn, error
    file: "~/.vman/logs/vman.log"  # 日志文件路径

# 全局工具版本
global_versions:
  kubectl: "1.28.0"
  terraform: "1.6.0"
  sqlc: "1.20.0"

# 已注册的工具信息
tools:
  kubectl:
    current_version: "1.28.0"
    installed_versions:
      - "1.28.0"
      - "1.29.0"
  terraform:
    current_version: "1.6.0"
    installed_versions:
      - "1.5.0"
      - "1.6.0"
```

### 配置字段说明

#### version (必需)
- **类型**: string
- **描述**: 配置文件版本
- **支持的值**: "1.0"

#### settings
下载、代理和日志相关的系统设置。

##### settings.download
- **timeout**: 下载超时时间 (1秒 - 30分钟)
- **retries**: 下载重试次数 (0 - 10)
- **concurrent_downloads**: 并发下载数 (1 - 10)

##### settings.proxy
- **enabled**: 是否启用命令代理
- **shims_in_path**: 是否将shims目录添加到PATH环境变量

##### settings.logging
- **level**: 日志级别 (debug, info, warn, error)
- **file**: 日志文件路径

#### global_versions
全局工具版本映射，格式为 `工具名: 版本号`。

#### tools
已安装工具的详细信息，包括当前版本和所有已安装版本。

## 项目配置文件 (.vman.yaml)

### 完整示例

```yaml
# vman 项目配置文件
version: "1.0"

# 项目特定的工具版本
tools:
  kubectl: "1.29.0"      # 覆盖全局版本
  terraform: "1.5.0"     # 覆盖全局版本
  sqlc: "1.19.0"         # 项目特定版本
```

### 配置字段说明

#### version (必需)
- **类型**: string
- **描述**: 配置文件版本
- **支持的值**: "1.0"

#### tools
项目特定的工具版本映射，会覆盖全局配置中的相应设置。

## 工具定义文件 (工具名.toml)

### 完整示例 - kubectl.toml

```toml
[tool]
name = "kubectl"
description = "Kubernetes command-line tool"
homepage = "https://kubernetes.io/docs/reference/kubectl/"
repository = "https://github.com/kubernetes/kubernetes"

[download]
type = "direct"
url_template = "https://dl.k8s.io/release/v{version}/bin/{os}/{arch}/kubectl"
extract_binary = "kubectl"

[download.headers]
"User-Agent" = "vman/0.1.0"

[versions]
[versions.aliases]
latest = "1.29.0"
stable = "1.28.0"

[versions.constraints]
min_version = "1.20.0"
```

### 完整示例 - terraform.toml

```toml
[tool]
name = "terraform"
description = "Infrastructure as Code tool"
homepage = "https://www.terraform.io/"
repository = "https://github.com/hashicorp/terraform"

[download]
type = "archive"
url_template = "https://releases.hashicorp.com/terraform/{version}/terraform_{version}_{os}_{arch}.zip"
extract_binary = "terraform"

[versions]
[versions.aliases]
latest = "1.7.0"
stable = "1.6.6"

[versions.constraints]
min_version = "1.0.0"
max_version = "2.0.0"
```

### 配置字段说明

#### [tool] 部分
- **name**: 工具名称 (必需，字母、数字、连字符、下划线，最大50字符)
- **description**: 工具描述 (必需)
- **homepage**: 工具主页URL (必需，必须以http://或https://开头)
- **repository**: 源代码仓库URL (必需)

#### [download] 部分
- **type**: 下载类型 (必需)
  - `direct`: 直接下载二进制文件
  - `archive`: 下载并解压归档文件
  - `github`: 从GitHub Releases下载
- **url_template**: URL模板 (direct和archive类型必需)
- **repository**: GitHub仓库 (github类型必需)
- **asset_pattern**: 资产文件匹配模式 (github类型可选)
- **extract_binary**: 要提取的二进制文件名 (archive类型必需)
- **headers**: HTTP请求头 (可选)

#### [versions] 部分
- **aliases**: 版本别名映射
- **constraints**: 版本约束
  - **min_version**: 最小支持版本
  - **max_version**: 最大支持版本

## 版本格式

vman 支持以下版本格式：

### 标准语义版本
- `1.0.0` - 标准语义版本
- `v1.0.0` - 带v前缀的语义版本
- `1.0.0-alpha` - 预发布版本
- `1.0.0-alpha.1` - 带数字的预发布版本
- `1.0.0+build` - 带构建元数据的版本

### 简化版本
- `1.0` - 两位版本号
- `v1.0` - 带v前缀的两位版本号

### 特殊版本
- `latest` - 最新版本
- `stable` - 稳定版本
- `main` - 主分支版本
- `master` - 主分支版本

## 配置优先级

vman 使用以下优先级来解析工具版本：

1. **项目配置** - 项目中的 `.vman.yaml` 文件
2. **全局版本** - 全局配置中的 `global_versions` 部分
3. **工具当前版本** - 全局配置中工具信息的 `current_version`

## 配置验证

### 全局配置验证
- version 字段必须存在且为支持的版本
- 下载设置必须在有效范围内
- 日志级别必须为支持的值
- 工具名称和版本必须有效

### 项目配置验证
- version 字段必须存在且为支持的版本
- 工具名称和版本必须有效

### 工具定义验证
- 工具名称必须有效
- 描述不能为空
- URL必须以http://或https://开头
- 下载配置必须完整且有效
- 版本约束必须有效（最小版本不能大于最大版本）

## 环境变量替换

配置文件中支持以下环境变量替换：

- `~` - 用户主目录
- `${HOME}` - 用户主目录
- `${VMAN_CONFIG_DIR}` - vman配置目录
- `${VMAN_CACHE_DIR}` - vman缓存目录

## 配置迁移

### 从旧版本迁移
当配置文件版本不匹配时，vman 会自动进行迁移：

1. 备份现有配置
2. 应用默认值
3. 保留兼容的设置
4. 记录迁移日志

### 配置重置
可以使用以下命令重置配置：

```bash
# 重置全局配置
vman config reset --global

# 重置项目配置
vman config reset --project

# 重置所有配置
vman config reset --all
```

## 最佳实践

### 全局配置
1. 在全局配置中设置常用工具的默认版本
2. 配置合理的下载超时和重试次数
3. 根据网络环境调整并发下载数
4. 设置适当的日志级别

### 项目配置
1. 只在项目配置中覆盖必要的工具版本
2. 使用版本别名（如stable、latest）提高可维护性
3. 在团队项目中提交 `.vman.yaml` 文件到版本控制

### 工具定义
1. 为新工具创建完整的工具定义文件
2. 设置合理的版本约束
3. 提供准确的工具描述和链接
4. 测试下载配置的有效性

## 故障排除

### 常见问题

#### 配置文件格式错误
- 检查YAML/TOML语法
- 验证字段名和值的正确性
- 使用 `vman config validate` 命令验证配置

#### 版本解析失败
- 检查版本格式是否正确
- 验证版本约束设置
- 确认工具定义文件存在

#### 权限问题
- 确保vman配置目录有读写权限
- 检查项目目录的写入权限

#### 网络问题
- 检查网络连接
- 调整下载超时设置
- 配置代理设置（如需要）