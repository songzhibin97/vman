# vman 发布流程

本文档描述了 vman 项目的版本发布流程，包括版本规划、构建、测试、发布和发布后的流程。

## 📋 目录

- [🎯 发布原则](#-发布原则)
- [📅 发布计划](#-发布计划)
- [🔢 版本管理](#-版本管理)
- [🚀 发布流程](#-发布流程)
- [🧪 测试策略](#-测试策略)
- [📦 构建和打包](#-构建和打包)
- [📢 发布公告](#-发布公告)
- [🔄 发布后流程](#-发布后流程)

## 🎯 发布原则

### 质量优先
- 🔍 每个发布版本都必须通过完整的测试套件
- 🛡️ 不能引入已知的安全漏洞
- 📊 性能不能显著倒退
- 🔧 必须保持向后兼容性（除非是 MAJOR 版本）

### 透明度
- 📝 所有变更都在 CHANGELOG.md 中记录
- 🔄 发布说明详细描述新功能和破坏性变更
- 📅 发布时间表公开透明
- 🗣️ 社区参与发布计划的讨论

### 稳定性
- ⚡ 使用稳定的发布分支
- 🧪 充分的测试时间
- 🔒 代码冻结期间只允许关键 Bug 修复
- 📋 清晰的发布标准和检查清单

## 📅 发布计划

### 发布周期

| 版本类型 | 发布频率 | 内容 |
|----------|----------|------|
| Patch (x.y.Z) | 按需发布 | Bug 修复、安全更新 |
| Minor (x.Y.0) | 每月一次 | 新功能、改进 |
| Major (X.0.0) | 按需发布 | 破坏性变更、重大重构 |
| LTS | 每年一次 | 长期支持版本 |

### 发布时间表

#### 月度发布（Minor 版本）
- **第1周**: 功能开发和合并
- **第2周**: 功能完成，开始测试
- **第3周**: Bug 修复，候选版本构建
- **第4周**: 最终测试，正式发布

#### LTS 发布（年度）
- **Q1**: 功能规划和开发
- **Q2**: 功能开发继续，Alpha 版本
- **Q3**: 功能冻结，Beta 版本，广泛测试
- **Q4**: 候选版本，正式发布

## 🔢 版本管理

### 语义化版本控制

遵循 [Semantic Versioning](https://semver.org/) 规范：

```
MAJOR.MINOR.PATCH
```

- **MAJOR**: 包含破坏性变更时递增
- **MINOR**: 添加向后兼容的新功能时递增  
- **PATCH**: 进行向后兼容的问题修复时递增

### 版本号示例

```bash
1.0.0    # 首个稳定版本
1.1.0    # 添加新功能
1.1.1    # Bug 修复
1.2.0    # 更多新功能
2.0.0    # 破坏性变更
```

### 预发布版本

```bash
1.1.0-alpha.1    # Alpha 版本
1.1.0-beta.1     # Beta 版本
1.1.0-rc.1       # 候选版本
```

### 分支策略

```
main                 # 主分支，稳定代码
├── develop         # 开发分支
├── release/v1.1.0  # 发布分支
├── hotfix/v1.0.1   # 热修复分支
└── feature/xxx     # 功能分支
```

## 🚀 发布流程

### 1. 准备阶段

#### 检查发布条件
```bash
# 确保所有测试通过
make test-all

# 检查代码质量
make lint

# 验证文档完整性
make docs-check

# 检查依赖安全性
make security-check
```

#### 更新版本信息
```bash
# 更新版本号
vim internal/version/info.go

# 更新 CHANGELOG.md
vim CHANGELOG.md

# 更新文档中的版本引用
grep -r "v0\.1\.0" docs/ # 查找需要更新的版本号
```

### 2. 创建发布分支

```bash
# 从 develop 分支创建发布分支
git checkout develop
git pull origin develop
git checkout -b release/v1.1.0

# 提交版本更新
git add .
git commit -m "chore: bump version to v1.1.0"
git push origin release/v1.1.0
```

### 3. 候选版本测试

```bash
# 构建候选版本
make build-all

# 创建候选版本标签
git tag v1.1.0-rc.1
git push origin v1.1.0-rc.1

# 自动触发构建和测试
# GitHub Actions 会自动:
# - 运行完整测试套件
# - 构建多平台二进制文件
# - 运行安全扫描
# - 生成候选发布版本
```

### 4. 发布测试

#### 自动化测试
- ✅ 单元测试覆盖率 > 80%
- ✅ 集成测试全部通过
- ✅ 端到端测试通过
- ✅ 性能基准测试
- ✅ 安全扫描无高危漏洞

#### 手动测试
```bash
# 在多个平台上测试候选版本
# Linux
wget https://github.com/songzhibin97/vman/releases/download/v1.1.0-rc.1/vman-linux-amd64.tar.gz
./test-scenarios.sh

# macOS  
wget https://github.com/songzhibin97/vman/releases/download/v1.1.0-rc.1/vman-darwin-amd64.tar.gz
./test-scenarios.sh

# Windows
# 下载并测试 Windows 版本
```

#### 测试场景脚本
```bash
#!/bin/bash
# test-scenarios.sh

set -e

echo "🧪 开始发布前测试..."

# 基础功能测试
./vman --version
./vman init
./vman add kubectl
./vman install kubectl 1.28.0
./vman global kubectl 1.28.0
./vman current kubectl

# 项目级配置测试
mkdir test-project
cd test-project
echo "kubectl 1.29.0" > .vman-version
../vman current kubectl
cd ..

# 清理测试
./vman cleanup
rm -rf test-project

echo "✅ 发布前测试完成！"
```

### 5. 正式发布

```bash
# 合并发布分支到 main
git checkout main
git merge release/v1.1.0 --no-ff
git push origin main

# 创建正式版本标签
git tag -a v1.1.0 -m "Release v1.1.0

## 新功能
- 添加了 xxx 功能
- 改进了 yyy 性能

## Bug 修复  
- 修复了 zzz 问题

## 破坏性变更
- 无

完整更新日志: https://github.com/songzhibin97/vman/blob/main/CHANGELOG.md#v110"

git push origin v1.1.0

# 合并回 develop 分支
git checkout develop
git merge main
git push origin develop

# 删除发布分支
git branch -d release/v1.1.0
git push origin --delete release/v1.1.0
```

## 🧪 测试策略

### 测试层级

```
                🔺
               /   \
              /     \
             /  E2E   \     端到端测试
            /         \
           /___________\
          /             \
         /  Integration  \   集成测试
        /                 \
       /___________________\
      /                     \
     /       Unit Tests      \  单元测试
    /_________________________\
```

### 测试自动化

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: [1.21, 1.22]
    
    runs-on: ${{ matrix.os }}
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go }}
    
    - name: Run tests
      run: |
        make test
        make test-integration
        make test-e2e
    
    - name: Security scan
      run: make security-scan
    
    - name: Performance benchmark
      run: make benchmark
```

### 性能基准测试

```bash
# 运行性能基准测试
make benchmark

# 比较性能变化
go test -bench=. -benchmem -count=5 ./... > new.bench
benchcmp old.bench new.bench
```

### 安全测试

```bash
# 依赖漏洞扫描
make security-scan

# 静态代码分析
gosec ./...

# 二进制文件安全扫描
./scripts/binary-security-scan.sh
```

## 📦 构建和打包

### 构建配置

```makefile
# Makefile 中的发布构建目标
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD)

LDFLAGS := -ldflags="-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH) -s -w"

# 跨平台构建
.PHONY: build-release
build-release:
	@echo "构建发布版本..."
	@mkdir -p dist
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/vman-linux-amd64 ./cmd/vman
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/vman-linux-arm64 ./cmd/vman
	
	# macOS  
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/vman-darwin-amd64 ./cmd/vman
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/vman-darwin-arm64 ./cmd/vman
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/vman-windows-amd64.exe ./cmd/vman
	
	# 创建压缩包
	@for binary in dist/vman-*; do \
		if [[ "$$binary" == *.exe ]]; then \
			zip "$${binary%.exe}.zip" "$$binary"; \
		else \
			tar czf "$$binary.tar.gz" "$$binary"; \
		fi; \
	done
```

### 打包脚本

```bash
#!/bin/bash
# scripts/package.sh

set -e

VERSION=${1:-$(git describe --tags --always)}
DIST_DIR="dist"

echo "📦 打包 vman v$VERSION..."

# 创建分发目录
mkdir -p "$DIST_DIR"

# 复制必要文件
cp README.md "$DIST_DIR/"
cp LICENSE "$DIST_DIR/"
cp CHANGELOG.md "$DIST_DIR/"

# 为每个平台创建完整包
for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do
    package_dir="$DIST_DIR/vman-$VERSION-$platform"
    mkdir -p "$package_dir"
    
    # 复制二进制文件
    if [[ "$platform" == *"windows"* ]]; then
        cp "$DIST_DIR/vman-$platform.exe" "$package_dir/vman.exe"
    else
        cp "$DIST_DIR/vman-$platform" "$package_dir/vman"
        chmod +x "$package_dir/vman"
    fi
    
    # 复制文档
    cp README.md LICENSE CHANGELOG.md "$package_dir/"
    
    # 创建安装脚本
    cat > "$package_dir/install.sh" << 'EOF'
#!/bin/bash
set -e
echo "安装 vman..."
sudo cp vman /usr/local/bin/
sudo chmod +x /usr/local/bin/vman
echo "vman 安装完成！运行 'vman --version' 验证安装。"
EOF
    chmod +x "$package_dir/install.sh"
    
    # 创建压缩包
    if [[ "$platform" == *"windows"* ]]; then
        (cd "$DIST_DIR" && zip -r "vman-$VERSION-$platform.zip" "vman-$VERSION-$platform/")
    else
        (cd "$DIST_DIR" && tar czf "vman-$VERSION-$platform.tar.gz" "vman-$VERSION-$platform/")
    fi
    
    # 清理临时目录
    rm -rf "$package_dir"
done

echo "✅ 打包完成！"
ls -la "$DIST_DIR"/*.{tar.gz,zip} 2>/dev/null || true
```

### 校验和生成

```bash
#!/bin/bash
# scripts/generate-checksums.sh

cd dist

echo "📋 生成校验和..."

# 生成 SHA256 校验和
sha256sum *.tar.gz *.zip > checksums.txt

# 生成 GPG 签名
gpg --detach-sign --armor checksums.txt

echo "✅ 校验和生成完成！"
cat checksums.txt
```

## 📢 发布公告

### GitHub Release

自动创建 GitHub Release，包含：

```markdown
## vman v1.1.0 发布 🎉

这是 vman 的第 X 个版本，包含了多项新功能和改进。

### ✨ 新功能
- 新增 xxx 功能，支持 yyy
- 改进 zzz 性能，提升 20%
- 添加 aaa 命令，简化 bbb 操作

### 🐛 Bug 修复
- 修复了在 Windows 上的路径问题
- 解决了配置文件解析错误
- 修正了版本切换的竞态条件

### 💔 破坏性变更
- 无

### 📥 下载

选择适合你平台的二进制文件：

| 平台 | 架构 | 下载链接 |
|------|------|----------|
| Linux | amd64 | [vman-linux-amd64.tar.gz](link) |
| Linux | arm64 | [vman-linux-arm64.tar.gz](link) |
| macOS | amd64 | [vman-darwin-amd64.tar.gz](link) |
| macOS | arm64 | [vman-darwin-arm64.tar.gz](link) |
| Windows | amd64 | [vman-windows-amd64.zip](link) |

### 🚀 快速安装

```bash
# 使用安装脚本
curl -fsSL https://get.vman.dev | bash

# 或手动安装
wget https://github.com/songzhibin97/vman/releases/download/v1.1.0/vman-linux-amd64.tar.gz
tar -xzf vman-linux-amd64.tar.gz
sudo mv vman /usr/local/bin/
```

### 📊 完整更新日志

查看 [CHANGELOG.md](CHANGELOG.md#v110) 了解所有变更。

### 🙏 致谢

感谢所有贡献者和社区成员的支持！

**完整的变更集**: https://github.com/songzhibin97/vman/compare/v1.0.0...v1.1.0
```

### 社区公告

#### Twitter/X
```
🎉 vman v1.1.0 发布！

✨ 新功能: xxx
🐛 Bug 修复: yyy  
⚡ 性能提升: 20%

立即下载: https://github.com/songzhibin97/vman/releases/tag/v1.1.0

#vman #devtools #opensource
```

#### Reddit
在相关的 subreddits 发布：
- r/golang
- r/devops  
- r/kubernetes

#### 技术博客
发布详细的版本说明博客文章，包括：
- 新功能演示
- 升级指南
- 最佳实践更新

## 🔄 发布后流程

### 1. 验证发布

```bash
# 测试发布的二进制文件
curl -L https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz | tar xz
./vman --version
./vman doctor
```

### 2. 更新包管理器

#### Homebrew
```bash
# 更新 Homebrew formula
git clone https://github.com/songzhibin97/homebrew-vman
cd homebrew-vman
# 更新 vman.rb 中的版本和校验和
git commit -am "chore: bump vman to v1.1.0"
git push
```

#### Scoop  
```bash
# 更新 Scoop manifest
git clone https://github.com/songzhibin97/scoop-vman
cd scoop-vman
# 更新 vman.json
git commit -am "chore: bump vman to v1.1.0"
git push
```

### 3. 更新文档网站

```bash
# 更新文档网站的版本
cd docs-website
npm run update-version v1.1.0
npm run build
npm run deploy
```

### 4. 监控发布

#### 下载统计
- 监控 GitHub Release 下载量
- 跟踪包管理器安装量
- 分析用户地理分布

#### 错误监控
- 查看 GitHub Issues 中的新问题
- 监控社区反馈
- 跟踪错误报告

#### 性能监控
- 监控 CDN 响应时间
- 检查下载成功率
- 分析用户体验指标

### 5. 发布报告

在发布后一周内生成发布报告：

```markdown
# vman v1.1.0 发布报告

## 📊 统计数据
- 发布时间: 2024-02-01
- 下载次数: 1,234 (第一周)
- GitHub Stars 增长: +45
- 社区反馈: 98% 积极

## 🎯 目标达成
- ✅ 按时发布
- ✅ 零关键 Bug
- ✅ 性能提升 20%
- ✅ 用户满意度 > 95%

## 🐛 发现的问题
- 无关键问题
- 2 个小的文档错误已修复

## 📈 下次改进
- 增加更多自动化测试
- 改进发布文档的清晰度
- 优化构建时间
```

---

## 📚 相关文档

- [贡献指南](../CONTRIBUTING.md)
- [版本历史](../CHANGELOG.md)
- [安全政策](../SECURITY.md)
- [支持指南](support.md)

发布流程如有疑问，请联系维护团队或在 GitHub 上创建 Issue。