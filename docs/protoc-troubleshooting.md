# vman protoc 故障排除指南

本文档提供了使用 vman protoc 一键解决方案时可能遇到的常见问题及其解决方法。

## 🚨 常见问题

### 1. protoc 插件找不到

#### 症状
```bash
$ vman protoc make-api
protoc-gen-go: program not found or is not executable
```

#### 解决方案

**步骤1: 检查插件安装状态**
```bash
# 检查是否安装了所需版本
vman list protoc-gen-go
vman list protoc-gen-go-grpc
vman list protoc-gen-go-http

# 如果没有安装，执行安装
vman install protoc-gen-go 1.31.0
vman install protoc-gen-go-grpc 1.3.0
vman install protoc-gen-go-http 2.7.0
```

**步骤2: 检查当前使用版本**
```bash
# 检查当前使用的版本
vman current protoc-gen-go
vman current protoc-gen-go-grpc
vman current protoc-gen-go-http

# 如果版本不正确，切换到正确版本
vman use protoc-gen-go 1.31.0
vman use protoc-gen-go-grpc 1.3.0
vman use protoc-gen-go-http 2.7.0
```

**步骤3: 重新设置环境**
```bash
# 重新设置 protoc 环境
vman protoc setup

# 验证状态
vman protoc status
```

### 2. 版本冲突问题

#### 症状
```bash
$ protoc-gen-go --version
protoc-gen-go v1.28.0  # 期望是 v1.31.0
```

#### 解决方案

**检查版本来源**
```bash
# 查看详细版本信息
vman current protoc-gen-go --verbose

# 检查是否有系统安装的版本干扰
which protoc-gen-go
```

**清理并重新设置**
```bash
# 移除本地版本设置（如果存在）
vman local protoc-gen-go --unset

# 设置全局版本
vman global protoc-gen-go 1.31.0

# 或者设置项目版本
vman local protoc-gen-go 1.31.0

# 重新验证
vman protoc status
```

### 3. shim 冲突问题

#### 症状
```bash
$ vman protoc make-api
Error: protoc command conflicts with existing shim
```

#### 解决方案

**自动解决（推荐）**
```bash
# vman protoc 会自动处理 shim 冲突
vman protoc setup
```

**手动解决**
```bash
# 查看 shim 状态
ls -la ~/.vman/shims/protoc*

# 如果需要手动备份
mv ~/.vman/shims/protoc ~/.vman/shims/protoc.manual-backup

# 执行完毕后恢复
mv ~/.vman/shims/protoc.manual-backup ~/.vman/shims/protoc
```

### 4. 权限问题

#### 症状
```bash
$ vman protoc make-api
Permission denied: ~/.vman/shims/protoc-gen-go
```

#### 解决方案

**修复文件权限**
```bash
# 检查目录权限
ls -la ~/.vman/
ls -la ~/.vman/shims/

# 修复权限
chmod -R 755 ~/.vman/
chmod +x ~/.vman/shims/*

# 重新测试
vman protoc status
```

### 5. Makefile 问题

#### 症状
```bash
$ vman protoc make-api
Error: 当前目录不存在Makefile
```

#### 解决方案

**检查目录和文件**
```bash
# 确认当前目录
pwd

# 检查 Makefile 是否存在
ls -la Makefile

# 如果在子目录，指定正确的项目路径
vman protoc make-api --dir /path/to/project
```

### 6. 网络相关问题

#### 症状
```bash
$ vman install protoc-gen-go 1.31.0
Error: failed to download protoc-gen-go@1.31.0
```

#### 解决方案

**检查网络连接**
```bash
# 测试网络连接
curl -I https://github.com
ping github.com

# 检查代理设置
echo $https_proxy
echo $http_proxy
```

**使用代理（如果需要）**
```bash
# 设置代理
export https_proxy=http://proxy.company.com:8080
export http_proxy=http://proxy.company.com:8080

# 重新尝试安装
vman install protoc-gen-go 1.31.0
```

**清理缓存重试**
```bash
# 清理下载缓存
vman cleanup --cache

# 重新下载
vman install protoc-gen-go 1.31.0
```

## 🔍 诊断工具

### 系统诊断

```bash
# 运行完整诊断
vman doctor

# 检查特定工具
vman doctor protoc-gen-go

# 验证配置
vman config validate
```

### 详细状态检查

```bash
# 查看 protoc 环境详细状态
vman protoc status

# 查看所有已安装的 protoc 相关工具
vman list | grep protoc

# 查看当前工作目录的版本配置
vman current --all
```

### 调试模式

```bash
# 启用调试日志
export VMAN_LOG_LEVEL=debug

# 重新执行有问题的命令
vman protoc setup
vman protoc make-api

# 查看详细日志
tail -f ~/.vman/logs/vman.log
```

## 🛠️ 高级故障排除

### 环境变量检查

```bash
# 检查重要的环境变量
echo "PATH: $PATH"
echo "GOPATH: $GOPATH"
echo "GO111MODULE: $GO111MODULE"

# 检查 vman 相关环境变量
env | grep VMAN
```

### 手动验证工具

```bash
# 手动检查工具是否可执行
~/.vman/versions/protoc-gen-go/1.31.0/protoc-gen-go --version
~/.vman/versions/protoc-gen-go-grpc/1.3.0/protoc-gen-go-grpc --version

# 检查 PATH 中的工具
which protoc-gen-go
which protoc-gen-go-grpc
```

### 配置文件检查

```bash
# 检查全局配置
cat ~/.vman/config.yaml

# 检查项目配置（如果存在）
cat .vmanrc

# 验证配置语法
vman config validate
```

## 📊 性能优化

### 下载优化

```bash
# 启用并发下载
vman config set download.concurrent_downloads 3

# 启用下载缓存
vman config set cache.enable true
vman config set cache.ttl "24h"

# 启用压缩
vman config set download.compression true
```

### 磁盘空间管理

```bash
# 查看磁盘使用情况
du -sh ~/.vman/

# 清理旧版本
vman cleanup --old-versions

# 清理下载缓存
vman cleanup --cache

# 清理所有临时文件
vman cleanup --all
```

## 🔄 恢复和重置

### 重置 protoc 环境

```bash
# 恢复 protoc shim（如果被备份了）
vman protoc restore

# 重置 protoc 配置
rm -f ~/.vman/protoc-config.yaml

# 重新初始化
vman protoc setup
```

### 完全重置 vman

```bash
# 备份配置（可选）
cp ~/.vman/config.yaml ~/.vman/config.yaml.backup

# 重置配置
vman config reset

# 重新初始化
vman init

# 重新安装需要的工具
vman install protoc-gen-go 1.31.0
vman install protoc-gen-go-grpc 1.3.0
vman install protoc-gen-go-http 2.7.0
```

## 📞 获取帮助

### 内置帮助

```bash
# 查看 protoc 命令帮助
vman protoc --help
vman protoc setup --help
vman protoc make-api --help

# 查看完整帮助
vman --help
```

### 生成问题报告

```bash
# 生成诊断报告
vman bug-report

# 收集系统信息
vman doctor --export > vman-diagnosis.txt
```

### 社区支持

如果以上方法都无法解决问题，请：

1. 查看 [GitHub Issues](https://github.com/songzhibin97/vman/issues)
2. 搜索相关问题是否已被报告
3. 创建新的 issue，并包含：
   - 操作系统信息
   - vman 版本 (`vman --version`)
   - 详细的错误信息
   - 重现问题的步骤
   - `vman doctor` 的输出结果

## 💡 预防措施

### 最佳实践

1. **定期更新**: 定期运行 `vman update` 更新工具源
2. **版本固定**: 在项目中使用 `.vmanrc` 固定工具版本
3. **环境隔离**: 不同项目使用不同的本地版本配置
4. **定期清理**: 定期清理旧版本和缓存文件
5. **备份配置**: 重要的配置文件要备份

### 监控和维护

```bash
# 定期检查工具状态
vman protoc status

# 监控磁盘使用
du -sh ~/.vman/

# 检查配置有效性
vman config validate

# 更新工具源
vman update
```

---

如果本故障排除指南没有解决你的问题，请参考主文档或在 GitHub 上创建 issue。