# vman 部署指南

本文档提供了 vman 在不同环境下的部署和配置指南，包括企业环境、CI/CD 集成、容器化部署等场景。

## 📋 目录

- [🏢 企业环境部署](#-企业环境部署)
- [🔄 CI/CD 集成](#-cicd-集成)
- [🐳 容器化部署](#-容器化部署)
- [☁️ 云平台部署](#️-云平台部署)
- [⚙️ 配置管理](#️-配置管理)
- [🔒 安全配置](#-安全配置)
- [📊 监控和日志](#-监控和日志)
- [🚀 性能优化](#-性能优化)

## 🏢 企业环境部署

### 集中式部署

#### 共享存储部署

```bash
# 在共享存储上安装 vman
sudo mkdir -p /opt/vman
sudo chown admin:staff /opt/vman

# 下载并安装
cd /opt/vman
curl -fsSL https://get.vman.dev | bash

# 创建全局配置
cat > /opt/vman/config.yaml << EOF
version: "1.0"
settings:
  download:
    timeout: 600s
    concurrent_downloads: 1
    cache_dir: "/opt/vman/cache"
  proxy:
    enabled: true
    shims_in_path: true
  logging:
    level: "info"
    file: "/opt/vman/logs/vman.log"

# 企业内部工具配置
tools:
  kubectl:
    source:
      type: "direct"
      url_template: "https://internal-mirror.company.com/kubernetes/v{version}/bin/{os}/{arch}/kubectl"
  terraform:
    source:
      type: "direct"
      url_template: "https://internal-mirror.company.com/terraform/{version}/terraform_{version}_{os}_{arch}.zip"
EOF

# 设置环境变量
echo 'export VMAN_CONFIG_DIR="/opt/vman"' >> /etc/environment
echo 'export PATH="/opt/vman/shims:$PATH"' >> /etc/environment
```

#### 用户环境配置

```bash
# 为每个用户创建符号链接
for user in $(ls /home); do
    sudo -u $user ln -sf /opt/vman /home/$user/.vman
done

# 或使用用户配置脚本
cat > /etc/profile.d/vman.sh << 'EOF'
export VMAN_CONFIG_DIR="/opt/vman"
export PATH="/opt/vman/shims:$PATH"
eval "$(vman init bash 2>/dev/null || true)"
EOF
```

### 网络代理配置

```yaml
# /opt/vman/config.yaml
version: "1.0"
settings:
  download:
    proxy:
      http: "http://proxy.company.com:8080"
      https: "https://proxy.company.com:8080"
      no_proxy: "localhost,127.0.0.1,.company.com"
    headers:
      User-Agent: "vman/1.0.0 (Enterprise)"
      Authorization: "Bearer ${COMPANY_DOWNLOAD_TOKEN}"
```

### 离线部署

#### 1. 准备离线包

```bash
# 在有网络的机器上准备
mkdir vman-offline
cd vman-offline

# 下载 vman 二进制
curl -L https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz | tar xz

# 下载常用工具版本
./vman add kubectl
./vman add terraform
./vman download kubectl 1.28.0 1.29.0 --offline-mode
./vman download terraform 1.6.0 1.7.0 --offline-mode

# 打包
tar czf vman-offline.tar.gz .
```

#### 2. 离线安装

```bash
# 在离线机器上安装
tar xzf vman-offline.tar.gz
sudo cp vman /usr/local/bin/
sudo cp -r .vman ~/.vman

# 配置离线模式
vman config set download.offline_mode true
vman config set download.offline_cache_dir ~/.vman/offline-cache
```

## 🔄 CI/CD 集成

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy with vman

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Setup vman
      run: |
        curl -fsSL https://get.vman.dev | bash
        echo "$HOME/.vman/shims" >> $GITHUB_PATH
    
    - name: Install tools
      run: |
        vman install kubectl 1.28.0
        vman install terraform 1.6.0
        vman install helm 3.12.0
    
    - name: Deploy to Kubernetes
      run: |
        kubectl apply -f k8s/
        helm upgrade --install myapp ./charts/myapp
    
    - name: Terraform apply
      run: |
        terraform init
        terraform apply -auto-approve
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - setup
  - deploy

variables:
  VMAN_CONFIG_DIR: "$CI_PROJECT_DIR/.vman"

setup-tools:
  stage: setup
  script:
    - curl -fsSL https://get.vman.dev | bash
    - export PATH="$HOME/.vman/shims:$PATH"
    - vman install kubectl 1.28.0
    - vman install terraform 1.6.0
  artifacts:
    paths:
      - .vman/
    expire_in: 1 hour

deploy:
  stage: deploy
  dependencies:
    - setup-tools
  script:
    - export PATH="$PWD/.vman/shims:$PATH"
    - kubectl apply -f k8s/
    - terraform init && terraform apply -auto-approve
```

### Jenkins Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    environment {
        VMAN_CONFIG_DIR = "${WORKSPACE}/.vman"
    }
    
    stages {
        stage('Setup Tools') {
            steps {
                sh '''
                    curl -fsSL https://get.vman.dev | bash
                    export PATH="$HOME/.vman/shims:$PATH"
                    vman install kubectl 1.28.0
                    vman install terraform 1.6.0
                '''
            }
        }
        
        stage('Deploy') {
            steps {
                sh '''
                    export PATH="$HOME/.vman/shims:$PATH"
                    kubectl apply -f k8s/
                    terraform init
                    terraform apply -auto-approve
                '''
            }
        }
    }
    
    post {
        always {
            sh 'vman cleanup --cache'
        }
    }
}
```

### Azure DevOps

```yaml
# azure-pipelines.yml
trigger:
- main

pool:
  vmImage: 'ubuntu-latest'

variables:
  VMAN_CONFIG_DIR: '$(Pipeline.Workspace)/.vman'

steps:
- script: |
    curl -fsSL https://get.vman.dev | bash
    echo "##vso[task.prependpath]$(HOME)/.vman/shims"
  displayName: 'Install vman'

- script: |
    vman install kubectl 1.28.0
    vman install terraform 1.6.0
  displayName: 'Install tools'

- script: |
    kubectl apply -f k8s/
    terraform init && terraform apply -auto-approve
  displayName: 'Deploy'
```

## 🐳 容器化部署

### Docker 镜像构建

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o vman ./cmd/vman

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

COPY --from=builder /app/vman /usr/local/bin/

# 预安装常用工具
RUN vman init && \
    vman add kubectl && \
    vman add terraform && \
    vman install kubectl 1.28.0 && \
    vman install terraform 1.6.0

ENV PATH="/root/.vman/shims:$PATH"

CMD ["vman"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  vman:
    build: .
    volumes:
      - vman-config:/root/.vman
      - ./workspace:/workspace
    working_dir: /workspace
    environment:
      - VMAN_CONFIG_DIR=/root/.vman
    command: tail -f /dev/null

  # 开发环境服务
  dev-env:
    extends: vman
    volumes:
      - ./:/workspace
    command: bash -c "vman install kubectl 1.29.0 && kubectl proxy"

volumes:
  vman-config:
```

### Kubernetes 部署

```yaml
# k8s/vman-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: vman-tools
spec:
  replicas: 1
  selector:
    matchLabels:
      app: vman-tools
  template:
    metadata:
      labels:
        app: vman-tools
    spec:
      containers:
      - name: vman
        image: songzhibin97/vman:latest
        volumeMounts:
        - name: vman-config
          mountPath: /root/.vman
        - name: workspace
          mountPath: /workspace
        env:
        - name: VMAN_CONFIG_DIR
          value: /root/.vman
      volumes:
      - name: vman-config
        persistentVolumeClaim:
          claimName: vman-config-pvc
      - name: workspace
        emptyDir: {}

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: vman-config-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

## ☁️ 云平台部署

### AWS

#### EC2 实例部署

```bash
#!/bin/bash
# aws-userdata.sh

# 安装 vman
curl -fsSL https://get.vman.dev | bash

# 配置 AWS 特定设置
cat > ~/.vman/config.yaml << EOF
version: "1.0"
settings:
  download:
    timeout: 300s
    concurrent_downloads: 2
  logging:
    level: "info"
    file: "/var/log/vman.log"

global_versions:
  kubectl: "1.28.0"
  aws-cli: "2.13.0"
  terraform: "1.6.0"
EOF

# 安装 AWS 相关工具
vman add aws-cli
vman add kubectl
vman add terraform
vman install aws-cli 2.13.0
vman install kubectl 1.28.0
vman install terraform 1.6.0

# 配置 shell 集成
echo 'eval "$(vman init bash)"' >> /home/ec2-user/.bashrc
```

#### Lambda 层

```bash
# 创建 Lambda 层
mkdir lambda-layer
cd lambda-layer

# 下载 vman
curl -L https://github.com/songzhibin97/vman/releases/latest/download/vman-linux-amd64.tar.gz | tar xz

# 预安装工具
./vman init
./vman add kubectl
./vman install kubectl 1.28.0

# 打包层
zip -r vman-layer.zip .
aws lambda publish-layer-version \
    --layer-name vman-tools \
    --zip-file fileb://vman-layer.zip \
    --compatible-runtimes python3.9 nodejs18.x
```

### Google Cloud Platform

```bash
# gcp-startup.sh
#!/bin/bash

# 安装 vman
curl -fsSL https://get.vman.dev | bash

# 配置 GCP 工具
vman add gcloud
vman add kubectl
vman install gcloud latest
vman install kubectl 1.28.0

# 设置 GCP 认证
gcloud auth configure-docker
```

### Azure

```bash
# azure-init.sh
#!/bin/bash

# 安装 vman
curl -fsSL https://get.vman.dev | bash

# 配置 Azure 工具
vman add azure-cli
vman add kubectl
vman install azure-cli latest
vman install kubectl 1.28.0

# Azure 登录
az login --service-principal --username $AZURE_CLIENT_ID --password $AZURE_CLIENT_SECRET --tenant $AZURE_TENANT_ID
```

## ⚙️ 配置管理

### 环境特定配置

#### 开发环境

```yaml
# ~/.vman/config.yaml
version: "1.0"
settings:
  download:
    timeout: 300s
    concurrent_downloads: 3
    verify_checksum: false  # 开发环境可以跳过校验
  logging:
    level: "debug"
    file: "/tmp/vman-dev.log"

global_versions:
  kubectl: "1.29.0"  # 使用最新版本
  terraform: "1.7.0"
```

#### 生产环境

```yaml
# ~/.vman/config.yaml
version: "1.0"
settings:
  download:
    timeout: 600s
    concurrent_downloads: 1
    verify_checksum: true   # 生产环境必须校验
    retry_attempts: 3
  logging:
    level: "warn"
    file: "/var/log/vman.log"
    max_size: "100MB"
    max_files: 10

global_versions:
  kubectl: "1.28.0"  # 使用稳定版本
  terraform: "1.6.0"
```

### 配置模板

#### 团队标准配置

```yaml
# team-config-template.yaml
version: "1.0"

# 标准工具版本
global_versions:
  kubectl: "1.28.0"
  terraform: "1.6.0"
  helm: "3.12.0"
  sqlc: "1.20.0"

# 统一设置
settings:
  download:
    timeout: 300s
    concurrent_downloads: 2
    verify_checksum: true
  proxy:
    enabled: true
    shims_in_path: true
  logging:
    level: "info"

# 工具源配置
tools:
  kubectl:
    source:
      type: "github"
      repository: "kubernetes/kubernetes"
      asset_pattern: "kubectl-{os}-{arch}"
  terraform:
    source:
      type: "direct"
      url_template: "https://releases.hashicorp.com/terraform/{version}/terraform_{version}_{os}_{arch}.zip"
```

### 配置验证脚本

```bash
#!/bin/bash
# validate-config.sh

set -e

echo "验证 vman 配置..."

# 检查配置文件语法
vman config validate

# 检查工具源可用性
for tool in kubectl terraform helm; do
    echo "检查 $tool 工具源..."
    vman list-all $tool --limit 1 >/dev/null
done

# 检查网络连接
echo "检查网络连接..."
curl -s https://github.com >/dev/null

# 检查磁盘空间
echo "检查磁盘空间..."
df -h ~/.vman

echo "配置验证完成！"
```

## 🔒 安全配置

### 网络安全

```yaml
# 安全配置示例
version: "1.0"
settings:
  download:
    # 强制 HTTPS
    force_https: true
    
    # 验证证书
    verify_ssl: true
    
    # 校验文件完整性
    verify_checksum: true
    
    # 允许的域名白名单
    allowed_domains:
      - "github.com"
      - "releases.hashicorp.com"
      - "storage.googleapis.com"
    
    # 请求头设置
    headers:
      User-Agent: "vman/1.0.0 (Company-Name)"
    
    # 超时设置
    timeout: 300s
    connect_timeout: 30s
```

### 访问控制

```bash
# 设置文件权限
chmod 700 ~/.vman
chmod 600 ~/.vman/config.yaml

# 限制可执行文件权限
find ~/.vman/tools -type f -executable -exec chmod 755 {} \;

# 设置组权限（团队共享）
chgrp -R dev-team ~/.vman
chmod -R g+r ~/.vman
```

### 审计日志

```yaml
# 启用审计日志
settings:
  logging:
    level: "info"
    file: "/var/log/vman-audit.log"
    audit:
      enabled: true
      events:
        - "install"
        - "uninstall"
        - "download"
        - "config_change"
```

## 📊 监控和日志

### 日志管理

#### 结构化日志配置

```yaml
settings:
  logging:
    level: "info"
    format: "json"  # 或 "text"
    file: "/var/log/vman.log"
    max_size: "100MB"
    max_files: 10
    compress: true
    
    # 日志字段
    fields:
      service: "vman"
      environment: "production"
      version: "1.0.0"
```

#### 日志轮转

```bash
# /etc/logrotate.d/vman
/var/log/vman*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
```

### 性能监控

#### Prometheus 指标

```go
// 在代码中暴露指标
import "github.com/prometheus/client_golang/prometheus"

var (
    downloadDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "vman_download_duration_seconds",
            Help: "Time spent downloading tools",
        },
        []string{"tool", "version"},
    )
    
    activeVersions = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "vman_active_versions_total",
            Help: "Number of active tool versions",
        },
        []string{"tool"},
    )
)
```

#### 健康检查端点

```bash
# 启动健康检查服务
vman server --health-check --port 8080

# 检查端点
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

### 告警配置

```yaml
# prometheus-alerts.yml
groups:
- name: vman
  rules:
  - alert: VmanDownloadFailure
    expr: increase(vman_download_failures_total[5m]) > 0
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "vman download failure detected"
      
  - alert: VmanDiskSpaceHigh
    expr: vman_disk_usage_percent > 90
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "vman disk usage is high"
```

## 🚀 性能优化

### 下载优化

```yaml
settings:
  download:
    # 并发下载
    concurrent_downloads: 3
    
    # 连接池
    max_idle_conns: 10
    max_conns_per_host: 3
    
    # 缓存设置
    cache:
      enabled: true
      ttl: "24h"
      max_size: "1GB"
    
    # 压缩
    compression: true
    
    # 断点续传
    resume_downloads: true
```

### 存储优化

```bash
# 启用文件系统压缩（如果支持）
# ZFS
zfs set compression=lz4 pool/vman

# Btrfs
chattr +c ~/.vman

# 定期清理
vman cleanup --aggressive
vman cleanup --cache --older-than 30d
```

### 网络优化

```yaml
settings:
  download:
    # CDN 配置
    mirrors:
      - "https://mirror1.company.com"
      - "https://mirror2.company.com"
    
    # DNS 缓存
    dns_cache_ttl: "1h"
    
    # 连接复用
    keep_alive: true
    keep_alive_timeout: "30s"
```

---

## 🔗 相关文档

- [安装指南](../INSTALL.md)
- [用户指南](user-guide.md)
- [配置参考](configuration.md)
- [故障排除](troubleshooting.md)
- [API 文档](api.md)

如需更多部署相关的帮助，请查看 [GitHub Issues](https://github.com/songzhibin97/vman/issues) 或联系支持团队。