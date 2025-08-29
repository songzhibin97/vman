# Changelog

所有重要的项目变更都将记录在此文件中。

本项目的版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/) 规范。

## [Unreleased]

### Added
- 新增功能将在此列出

### Changed
- 变更的功能将在此列出

### Deprecated
- 即将废弃的功能将在此列出

### Removed
- 已移除的功能将在此列出

### Fixed
- Bug 修复将在此列出

### Security
- 安全相关的变更将在此列出

## [0.1.0] - 2024-01-15

### Added
- 🎉 初始版本发布
- ✨ 核心版本管理功能
  - 工具版本安装和卸载
  - 全局和项目级版本切换
  - 多工具源支持（GitHub Release、直接下载）
- 🛠️ CLI 命令行界面
  - `vman init` - 初始化配置
  - `vman add` - 添加工具源
  - `vman install` - 安装工具版本
  - `vman global/local` - 设置版本
  - `vman list` - 查看已安装版本
  - `vman current` - 查看当前版本
- ⚙️ 配置管理系统
  - YAML 格式的全局配置
  - 项目级 `.vmanrc` 配置文件
  - 环境变量覆盖支持
- 🔗 命令代理机制
  - 透明的命令拦截和路由
  - Shell 集成（Bash、Zsh、Fish、PowerShell）
  - 智能符号链接管理
- 📦 下载管理
  - 并发下载支持
  - 下载缓存机制
  - 文件完整性校验（SHA256）
  - 断点续传支持
- 🌐 跨平台支持
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- 🔧 开发工具
  - 完整的测试套件
  - CI/CD 流水线（GitHub Actions）
  - 代码质量检查（golangci-lint）
  - 自动化构建和发布

### Changed
- N/A (初始版本)

### Deprecated
- N/A (初始版本)

### Removed
- N/A (初始版本)

### Fixed
- N/A (初始版本)

### Security
- 🔒 强制 HTTPS 下载
- 🔐 文件完整性校验
- 🛡️ 安全的文件权限设置

---

## 版本说明

### 版本号规范
- **MAJOR.MINOR.PATCH** (例如: 1.2.3)
- **MAJOR**: 包含破坏性变更的版本
- **MINOR**: 添加新功能但保持向后兼容的版本
- **PATCH**: Bug 修复和小的改进，保持向后兼容

### 变更类型说明
- **Added**: 新增功能
- **Changed**: 现有功能的变更
- **Deprecated**: 即将在未来版本中移除的功能
- **Removed**: 在此版本中移除的功能
- **Fixed**: Bug 修复
- **Security**: 安全相关的修复和改进

### 发布周期
- **Major 版本**: 根据需要发布，通常包含重大架构变更
- **Minor 版本**: 每月发布，包含新功能和改进
- **Patch 版本**: 根据需要发布，主要用于 Bug 修复

### 破坏性变更政策
- 所有破坏性变更都将在 MAJOR 版本中引入
- 破坏性变更将在发布前至少一个 MINOR 版本中被标记为 Deprecated
- 详细的迁移指南将在破坏性变更发布时提供

### 长期支持 (LTS)
- 每年发布一个 LTS 版本
- LTS 版本将获得 18 个月的安全更新和重要 Bug 修复
- 下一个 LTS 版本计划为 v1.0.0

---

## 链接说明

- [Unreleased]: https://github.com/songzhibin97/vman/compare/v0.1.0...HEAD
- [0.1.0]: https://github.com/songzhibin97/vman/releases/tag/v0.1.0