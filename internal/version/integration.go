package version

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/pkg/types"
)

// IntegratedManager 集成版本管理器，包含下载功能
type IntegratedManager struct {
	*DefaultManager
	downloadManager DownloadManager
}

// DownloadManager 下载管理器接口（避免循环导入）
type DownloadManager interface {
	Download(ctx context.Context, tool, version string, options *DownloadOptions) error
	DownloadWithProgress(ctx context.Context, tool, version string, options *DownloadOptions, progress ProgressCallback) error
	SearchVersions(ctx context.Context, tool string) ([]*types.VersionInfo, error)
	GetVersionInfo(ctx context.Context, tool, version string) (*types.VersionInfo, error)
	AddSource(tool string, metadata *types.ToolMetadata) error
}

// DownloadOptions 下载选项（避免循环导入）
type DownloadOptions struct {
	Force        bool
	SkipChecksum bool
	Timeout      int
	Retries      int
	Resume       bool
	TempDir      string
	KeepDownload bool
	Headers      map[string]string
}

// NewIntegratedManager 创建集成版本管理器
func NewIntegratedManager(storageManager storage.Manager, configManager config.Manager, downloadManager DownloadManager) Manager {
	baseManager := &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             afero.NewOsFs(),
		logger:         logrus.New(),
	}

	return &IntegratedManager{
		DefaultManager:  baseManager,
		downloadManager: downloadManager,
	}
}

// NewIntegratedManagerWithFs 使用指定文件系统创建集成版本管理器
func NewIntegratedManagerWithFs(storageManager storage.Manager, configManager config.Manager, downloadManager DownloadManager, fs afero.Fs) Manager {
	baseManager := &DefaultManager{
		storageManager: storageManager,
		configManager:  configManager,
		fs:             fs,
		logger:         logrus.New(),
	}

	return &IntegratedManager{
		DefaultManager:  baseManager,
		downloadManager: downloadManager,
	}
}

// InstallVersion 自动下载并安装工具版本
func (im *IntegratedManager) InstallVersion(tool, version string) error {
	im.logger.Debugf("安装版本 %s@%s", tool, version)

	// 检查版本是否已安装
	if im.IsVersionInstalled(tool, version) {
		im.logger.Infof("版本 %s@%s 已安装", tool, version)
		return nil
	}

	// 使用下载管理器下载并安装
	ctx := context.Background()
	options := &DownloadOptions{
		Force: false,
	}

	if err := im.downloadManager.Download(ctx, tool, version, options); err != nil {
		return fmt.Errorf("下载安装失败: %w", err)
	}

	im.logger.Infof("成功安装 %s@%s", tool, version)
	return nil
}

// InstallVersionWithProgress 带进度显示的安装
func (im *IntegratedManager) InstallVersionWithProgress(tool, version string, progress ProgressCallback) error {
	im.logger.Debugf("带进度安装版本 %s@%s", tool, version)

	// 检查版本是否已安装
	if im.IsVersionInstalled(tool, version) {
		// 发送完成进度
		if progress != nil {
			progress(&types.ProgressInfo{
				Percentage: 100.0,
				Status:     "已安装",
			})
		}
		return nil
	}

	// 使用下载管理器下载并安装
	ctx := context.Background()
	options := &DownloadOptions{
		Force: false,
	}

	if err := im.downloadManager.DownloadWithProgress(ctx, tool, version, options, progress); err != nil {
		return fmt.Errorf("下载安装失败: %w", err)
	}

	im.logger.Infof("成功安装 %s@%s", tool, version)
	return nil
}

// InstallLatestVersion 安装最新版本
func (im *IntegratedManager) InstallLatestVersion(tool string) (string, error) {
	im.logger.Debugf("安装最新版本: %s", tool)

	// 搜索可用版本
	versions, err := im.SearchAvailableVersions(tool)
	if err != nil {
		return "", fmt.Errorf("搜索可用版本失败: %w", err)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("未找到可用版本")
	}

	// 选择最新的稳定版本
	var latestVersion string
	for _, version := range versions {
		if !version.IsPrerelease {
			latestVersion = version.Version
			break
		}
	}

	// 如果没有稳定版本，选择最新的预发布版本
	if latestVersion == "" {
		latestVersion = versions[0].Version
	}

	// 安装版本
	if err := im.InstallVersion(tool, latestVersion); err != nil {
		return "", fmt.Errorf("安装版本失败: %w", err)
	}

	return latestVersion, nil
}

// SearchAvailableVersions 搜索可用版本
func (im *IntegratedManager) SearchAvailableVersions(tool string) ([]*types.VersionInfo, error) {
	im.logger.Debugf("搜索可用版本: %s", tool)

	ctx := context.Background()
	return im.downloadManager.SearchVersions(ctx, tool)
}

// IsVersionAvailable 检查版本是否可下载
func (im *IntegratedManager) IsVersionAvailable(tool, version string) bool {
	ctx := context.Background()
	_, err := im.downloadManager.GetVersionInfo(ctx, tool, version)
	return err == nil
}

// UpdateTool 更新工具到最新版本
func (im *IntegratedManager) UpdateTool(tool string) (string, error) {
	im.logger.Debugf("更新工具: %s", tool)

	// 获取当前版本
	currentVersion, err := im.GetCurrentVersion(tool)
	if err != nil {
		// 如果没有当前版本，直接安装最新版本
		return im.InstallLatestVersion(tool)
	}

	// 获取最新版本
	latestVersion, err := im.InstallLatestVersion(tool)
	if err != nil {
		return "", fmt.Errorf("获取最新版本失败: %w", err)
	}

	if currentVersion == latestVersion {
		im.logger.Infof("工具 %s 已是最新版本 %s", tool, currentVersion)
		return currentVersion, nil
	}

	// 设置为当前版本
	if err := im.SetGlobalVersion(tool, latestVersion); err != nil {
		im.logger.Warnf("设置全局版本失败: %v", err)
	}

	im.logger.Infof("成功更新 %s 从 %s 到 %s", tool, currentVersion, latestVersion)
	return latestVersion, nil
}

// AddDownloadSource 添加下载源
func (im *IntegratedManager) AddDownloadSource(tool string, metadata *types.ToolMetadata) error {
	im.logger.Debugf("添加下载源: %s", tool)
	return im.downloadManager.AddSource(tool, metadata)
}

// InstallFromSource 从指定源安装工具
func (im *IntegratedManager) InstallFromSource(tool, version string, sourceType string, config map[string]string) error {
	im.logger.Debugf("从源安装 %s@%s (类型: %s)", tool, version, sourceType)

	// 创建临时工具元数据
	metadata := &types.ToolMetadata{
		Name: tool,
		DownloadConfig: types.DownloadConfig{
			Type:    sourceType,
			Headers: config,
		},
	}

	// 根据源类型设置配置
	switch sourceType {
	case "github":
		if repo, ok := config["repository"]; ok {
			metadata.DownloadConfig.Repository = repo
		}
		if pattern, ok := config["asset_pattern"]; ok {
			metadata.DownloadConfig.AssetPattern = pattern
		}
	case "direct", "archive":
		if urlTemplate, ok := config["url_template"]; ok {
			metadata.DownloadConfig.URLTemplate = urlTemplate
		}
	}

	// 添加下载源
	if err := im.AddDownloadSource(tool, metadata); err != nil {
		return fmt.Errorf("添加下载源失败: %w", err)
	}

	// 安装版本
	return im.InstallVersion(tool, version)
}

// BatchInstall 批量安装工具
func (im *IntegratedManager) BatchInstall(tools map[string]string, progress func(tool, version string, info *types.ProgressInfo)) error {
	im.logger.Debugf("批量安装 %d 个工具", len(tools))

	for tool, version := range tools {
		im.logger.Infof("安装 %s@%s", tool, version)

		toolProgress := func(info *types.ProgressInfo) {
			if progress != nil {
				progress(tool, version, info)
			}
		}

		if err := im.InstallVersionWithProgress(tool, version, toolProgress); err != nil {
			im.logger.Errorf("安装 %s@%s 失败: %v", tool, version, err)
			return fmt.Errorf("安装 %s@%s 失败: %w", tool, version, err)
		}
	}

	im.logger.Info("批量安装完成")
	return nil
}

// GetInstallStatus 获取安装状态
func (im *IntegratedManager) GetInstallStatus(tool string) (*InstallStatus, error) {
	status := &InstallStatus{
		Tool:      tool,
		Installed: false,
	}

	// 获取已安装版本
	versions, err := im.GetInstalledVersions(tool)
	if err != nil {
		return status, err
	}

	if len(versions) == 0 {
		return status, nil
	}

	status.Installed = true
	status.InstalledVersions = versions

	// 获取当前版本
	if currentVersion, err := im.GetCurrentVersion(tool); err == nil {
		status.CurrentVersion = currentVersion
	}

	// 获取最新版本（已安装的）
	if latestVersion, err := im.GetLatestVersion(tool); err == nil {
		status.LatestInstalled = latestVersion
	}

	return status, nil
}

// InstallStatus 安装状态
type InstallStatus struct {
	Tool              string   `json:"tool"`
	Installed         bool     `json:"installed"`
	CurrentVersion    string   `json:"current_version,omitempty"`
	InstalledVersions []string `json:"installed_versions,omitempty"`
	LatestInstalled   string   `json:"latest_installed,omitempty"`
	LatestAvailable   string   `json:"latest_available,omitempty"`
	UpdateAvailable   bool     `json:"update_available"`
}
