# vman 故障排除指南

本指南提供了使用 vman 时可能遇到的常见问题的解决方案。如果你的问题没有在这里找到答案，请查看 [GitHub Issues](https://github.com/songzhibin97/vman/issues) 或创建新的问题报告。

## 📋 目录

- [🛠️ 诊断工具](#️-诊断工具)
- [🚨 安装问题](#-安装问题)
- [⚙️ 配置问题](#️-配置问题)
- [📥 下载问题](#-下载问题)
- [🔄 版本管理问题](#-版本管理问题)
- [🔗 命令代理问题](#-命令代理问题)
- [🐧 平台特定问题](#-平台特定问题)
- [🔒 权限问题](#-权限问题)
- [🌐 网络问题](#-网络问题)
- [❓ 常见问题 FAQ](#-常见问题-faq)

## 🛠️ 诊断工具

### 运行系统诊断

```bash
# 运行完整诊断
vman doctor

# 检查特定工具
vman doctor kubectl

# 详细诊断输出
vman doctor --verbose

# 生成诊断报告
vman doctor --report > vman-diagnosis.txt
```

### 检查配置

```bash
# 验证配置文件
vman config validate

# 显示当前配置
vman config show

# 检查配置来源
vman config sources

# 测试配置加载
vman config test
```

### 查看日志

```bash
# 查看最新日志
vman logs

# 实时查看日志
vman logs --follow

# 查看特定级别的日志
vman logs --level error

# 查看特定时间范围的日志
vman logs --since "1h ago"
```

### 获取系统信息

```bash
# 显示系统信息
vman info system

# 显示版本信息
vman --version

# 显示环境变量
vman env

# 显示路径信息
vman paths
```

## 🚨 安装问题

### 问题：vman 命令找不到

**症状：**
```bash
$ vman --version
bash: vman: command not found
```

**解决方案：**

1. **检查安装路径**
```bash
# 检查 vman 是否已安装
which vman
ls -la /usr/local/bin/vman

# 如果文件存在但命令找不到，检查 PATH
echo $PATH
```

2. **重新安装**
```bash
# 使用安装脚本重新安装
curl -fsSL https://get.vman.dev | bash

# 手动添加到 PATH
export PATH="/usr/local/bin:$PATH"
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

3. **权限问题**
```bash
# 检查文件权限
ls -la /usr/local/bin/vman

# 修复权限
sudo chmod +x /usr/local/bin/vman
```

### 问题：安装脚本失败

**症状：**
```bash
curl: (7) Failed to connect to get.vman.dev port 443: Connection refused
```

**解决方案：**

1. **网络连接问题**
```bash
# 测试网络连接
ping github.com
curl -I https://github.com

# 使用代理
export https_proxy=http://proxy.company.com:8080
curl -fsSL https://get.vman.dev | bash
```

2. **手动下载安装**
```bash
# 直接从 GitHub 下载
wget https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz
tar -xzf vman-linux-amd64.tar.gz
sudo mv vman /usr/local/bin/
sudo chmod +x /usr/local/bin/vman
```

## ⚙️ 配置问题

### 问题：配置文件格式错误

**症状：**
```bash
$ vman config validate
Error: invalid YAML format in config file
```

**解决方案：**

1. **检查 YAML 格式**
```bash
# 使用 YAML 验证工具
python -c "import yaml; yaml.safe_load(open('~/.vman/config.yaml'))"

# 或使用在线 YAML 验证器
```

2. **备份并重置配置**
```bash
# 备份当前配置
cp ~/.vman/config.yaml ~/.vman/config.yaml.backup

# 生成默认配置
vman config reset

# 手动合并需要的设置
```

3. **常见 YAML 错误**
```yaml
# ❌ 错误：缩进不一致
version: "1.0"
settings:
  download:
  timeout: 300s  # 缩进错误

# ✅ 正确：一致的缩进
version: "1.0"
settings:
  download:
    timeout: 300s
```

### 问题：配置不生效

**症状：**
配置修改后，vman 行为没有改变。

**解决方案：**

1. **检查配置优先级**
```bash
# 查看有效配置
vman config effective

# 检查配置来源
vman config sources

# 清除缓存
vman config clear-cache
```

2. **环境变量覆盖**
```bash
# 检查环境变量
env | grep VMAN

# 取消环境变量设置
unset VMAN_CONFIG_DIR
unset VMAN_LOG_LEVEL
```

## 📥 下载问题

### 问题：下载失败

**症状：**
```bash
$ vman install kubectl 1.28.0
Error: failed to download kubectl 1.28.0: connection timeout
```

**解决方案：**

1. **网络连接问题**
```bash
# 测试网络连接
curl -I https://github.com
curl -I https://dl.k8s.io

# 检查 DNS 解析
nslookup github.com
```

2. **代理设置**
```bash
# 设置代理
vman config set download.proxy.http "http://proxy.company.com:8080"
vman config set download.proxy.https "https://proxy.company.com:8080"

# 或使用环境变量
export https_proxy=http://proxy.company.com:8080
```

3. **增加超时时间**
```bash
# 增加下载超时时间
vman config set download.timeout 600s

# 增加重试次数
vman config set download.retries 5
```

4. **清理缓存重试**
```bash
# 清理下载缓存
vman cleanup --cache

# 重新尝试下载
vman install kubectl 1.28.0 --force
```

### 问题：校验失败

**症状：**
```bash
Error: checksum verification failed for kubectl 1.28.0
```

**解决方案：**

1. **重新下载**
```bash
# 清理缓存重新下载
vman cleanup --cache
vman install kubectl 1.28.0
```

2. **跳过校验（不推荐）**
```bash
# 临时跳过校验
vman install kubectl 1.28.0 --skip-checksum

# 或在配置中禁用
vman config set download.verify_checksum false
```

3. **手动验证**
```bash
# 手动下载并验证
wget https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl
wget https://dl.k8s.io/release/v1.28.0/bin/linux/amd64/kubectl.sha256
sha256sum -c kubectl.sha256
```

## 🔄 版本管理问题

### 问题：版本切换不生效

**症状：**
```bash
$ vman global kubectl 1.29.0
$ kubectl version --client
Client Version: v1.28.0  # 仍然是旧版本
```

**解决方案：**

1. **重建符号链接**
```bash
# 重建所有符号链接
vman reshim

# 重建特定工具的符号链接
vman reshim kubectl
```

2. **检查 PATH 顺序**
```bash
# 检查 PATH 中的顺序
echo $PATH

# 确保 vman shims 目录在前面
export PATH="$HOME/.vman/shims:$PATH"
```

3. **检查项目配置覆盖**
```bash
# 查看当前版本来源
vman current kubectl --verbose

# 检查项目配置
cat .vmanrc
cat .vman-version
```

4. **清理缓存**
```bash
# 清理版本缓存
vman cache clear

# 重新加载配置
vman config reload
```

### 问题：无法安装特定版本

**症状：**
```bash
$ vman install kubectl 1.25.0
Error: version 1.25.0 not found for kubectl
```

**解决方案：**

1. **检查可用版本**
```bash
# 查看所有可用版本
vman list-all kubectl

# 更新工具源信息
vman update kubectl
```

2. **检查版本格式**
```bash
# 不同工具的版本格式可能不同
vman list-all kubectl | grep 1.25  # 查看实际版本号

# 可能需要添加前缀
vman install kubectl v1.25.0
```

3. **手动注册版本**
```bash
# 如果已有本地安装，可以手动注册
vman register kubectl 1.25.0 /path/to/kubectl-1.25.0
```

## 🔗 命令代理问题

### 问题：命令没有被代理

**症状：**
```bash
$ which kubectl
/usr/bin/kubectl  # 不是 vman shim
```

**解决方案：**

1. **检查 shim 生成**
```bash
# 查看 shim 目录
ls -la ~/.vman/shims/

# 重新生成 shim
vman reshim kubectl
```

2. **检查 PATH 配置**
```bash
# 检查 PATH 中 shims 目录的位置
echo $PATH | tr ':' '\n' | grep -n vman

# 手动添加到 PATH 前面
export PATH="$HOME/.vman/shims:$PATH"
```

3. **重新初始化 shell 集成**
```bash
# 重新添加 shell 集成
echo 'eval "$(vman init bash)"' >> ~/.bashrc
source ~/.bashrc
```

### 问题：shim 脚本执行失败

**症状：**
```bash
$ kubectl version
/home/user/.vman/shims/kubectl: line 2: vman: command not found
```

**解决方案：**

1. **检查 vman 路径**
```bash
# 检查 vman 是否在 PATH 中
which vman

# 编辑 shim 脚本，使用绝对路径
vim ~/.vman/shims/kubectl
# 将 vman 改为 /usr/local/bin/vman
```

2. **重新生成 shim**
```bash
# 使用正确路径重新生成
vman reshim --force
```

## 🐧 平台特定问题

### Linux 问题

#### 问题：权限被拒绝

**解决方案：**
```bash
# 检查 SELinux 状态
sestatus

# 如果 SELinux 启用，设置上下文
sudo setsebool -P use_nfs_home_dirs 1
sudo restorecon -R ~/.vman
```

#### 问题：依赖库缺失

**解决方案：**
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install libc6-dev

# CentOS/RHEL
sudo yum install glibc-devel

# Arch Linux
sudo pacman -S glibc
```

### macOS 问题

#### 问题：Gatekeeper 阻止执行

**解决方案：**
```bash
# 移除隔离属性
sudo xattr -rd com.apple.quarantine ~/.vman

# 或允许特定文件
sudo xattr -d com.apple.quarantine ~/.vman/tools/kubectl/1.28.0/kubectl
```

#### 问题：Apple Silicon 兼容性

**解决方案：**
```bash
# 检查架构
arch
uname -m

# 强制使用 arm64 版本
vman config set download.force_arch arm64

# 或使用 Rosetta 运行
arch -x86_64 vman install kubectl 1.28.0
```

### Windows 问题

#### 问题：PowerShell 执行策略

**解决方案：**
```powershell
# 检查执行策略
Get-ExecutionPolicy

# 设置执行策略
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

#### 问题：路径长度限制

**解决方案：**
```powershell
# 启用长路径支持
New-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" -Name "LongPathsEnabled" -Value 1 -PropertyType DWORD -Force
```

## 🔒 权限问题

### 问题：无法写入配置目录

**症状：**
```bash
Error: permission denied: ~/.vman/config.yaml
```

**解决方案：**

1. **检查目录权限**
```bash
# 检查权限
ls -la ~/.vman/

# 修复权限
chmod 755 ~/.vman/
chmod 644 ~/.vman/config.yaml
```

2. **目录被其他用户创建**
```bash
# 检查所有者
ls -la ~ | grep .vman

# 修改所有者
sudo chown -R $USER:$USER ~/.vman/
```

3. **使用不同的配置目录**
```bash
# 临时使用其他目录
export VMAN_CONFIG_DIR=/tmp/vman-config
vman init
```

## 🌐 网络问题

### 问题：企业防火墙阻止

**解决方案：**

1. **配置代理**
```bash
# HTTP 代理
vman config set download.proxy.http "http://proxy.company.com:8080"
vman config set download.proxy.https "https://proxy.company.com:8080"

# 无代理列表
vman config set download.proxy.no_proxy "localhost,127.0.0.1,.company.com"
```

2. **使用内部镜像**
```bash
# 配置内部镜像
vman config set download.mirrors.kubectl "https://internal-mirror.company.com/kubernetes"
```

3. **离线安装**
```bash
# 在有网络的机器上下载
vman download kubectl 1.28.0 --offline-mode

# 传输到目标机器
scp -r ~/.vman/cache/ user@target-machine:~/.vman/

# 在目标机器上安装
vman install kubectl 1.28.0 --offline
```

### 问题：DNS 解析失败

**解决方案：**

1. **测试 DNS**
```bash
# 测试 DNS 解析
nslookup github.com
dig github.com

# 使用不同的 DNS
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

2. **使用 IP 地址**
```bash
# 添加 hosts 条目
echo "140.82.112.3 github.com" | sudo tee -a /etc/hosts
```

## ❓ 常见问题 FAQ

### Q: vman 和其他版本管理工具有冲突吗？

A: vman 设计为与其他版本管理工具共存。但要确保：
- PATH 中 vman shims 目录在其他工具前面
- 避免同时管理相同的工具
- 使用 `vman doctor` 检查冲突

### Q: 如何完全卸载 vman？

A: 按以下步骤卸载：
```bash
# 1. 移除二进制文件
sudo rm -f /usr/local/bin/vman

# 2. 移除配置和数据
rm -rf ~/.vman

# 3. 移除 shell 集成
# 编辑 ~/.bashrc、~/.zshrc 等，删除 vman init 相关行

# 4. 重新加载 shell
exec $SHELL
```

### Q: vman 支持私有仓库吗？

A: 支持。可以配置认证信息：
```bash
# 使用令牌认证
vman config set download.headers.Authorization "Bearer YOUR_TOKEN"

# 或使用基本认证
vman config set download.auth.username "your-username"
vman config set download.auth.password "your-password"
```

### Q: 如何备份 vman 配置？

A: 备份这些目录和文件：
```bash
# 备份配置
tar czf vman-backup.tar.gz ~/.vman/config.yaml ~/.vman/sources/

# 备份已安装的工具
vman export > tools-list.yaml

# 恢复时
vman import tools-list.yaml
```

### Q: vman 占用多少磁盘空间？

A: 这取决于安装的工具数量：
```bash
# 查看磁盘使用情况
vman info disk-usage

# 清理不必要的版本
vman cleanup --old-versions
vman cleanup --cache
```

### Q: 如何在团队中共享配置？

A: 使用以下方法：
1. 将 `.vmanrc` 添加到版本控制
2. 创建共享配置模板
3. 使用环境变量覆盖特定设置
4. 设置团队标准的工具源

### Q: vman 是否支持多架构？

A: 支持。vman 会自动检测系统架构：
```bash
# 查看支持的架构
vman info system

# 强制使用特定架构
vman config set download.force_arch arm64
```

### Q: 如何报告 Bug？

A: 请按以下步骤报告：
1. 运行 `vman doctor` 收集诊断信息
2. 查看日志文件 `~/.vman/logs/vman.log`
3. 在 GitHub 上创建 Issue，包含：
   - 操作系统和版本
   - vman 版本
   - 重现步骤
   - 错误信息
   - 诊断输出

---

## 🔗 相关资源

- [用户指南](user-guide.md)
- [配置参考](configuration.md)
- [GitHub Issues](https://github.com/songzhibin97/vman/issues)
- [贡献指南](../CONTRIBUTING.md)

如果本指南没有解决你的问题，请在 GitHub 上创建新的 Issue 或参与 Discussions。