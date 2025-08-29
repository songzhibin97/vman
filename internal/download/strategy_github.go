package download

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

// GitHubStrategy GitHub Releases下载策略
type GitHubStrategy struct {
	metadata   *types.ToolMetadata
	fs         afero.Fs
	logger     *logrus.Logger
	downloader Downloader
	extractor  *PackageProcessor
	client     *http.Client
}

// GitHubRelease GitHub发布信息
type GitHubRelease struct {
	ID          int           `json:"id"`
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	CreatedAt   string        `json:"created_at"`
	PublishedAt string        `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset GitHub资产信息
type GitHubAsset struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Label              string `json:"label"`
	ContentType        string `json:"content_type"`
	Size               int64  `json:"size"`
	DownloadCount      int    `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// NewGitHubStrategy 创建GitHub下载策略
func NewGitHubStrategy(metadata *types.ToolMetadata, fs afero.Fs, logger *logrus.Logger) Strategy {
	return &GitHubStrategy{
		metadata:   metadata,
		fs:         fs,
		logger:     logger,
		downloader: NewHTTPDownloader(fs, logger),
		extractor:  NewPackageProcessor(fs, logger),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDownloadInfo 获取下载信息
func (g *GitHubStrategy) GetDownloadInfo(ctx context.Context, version string) (*types.DownloadInfo, error) {
	g.logger.Debugf("获取GitHub下载信息: %s@%s", g.metadata.Name, version)

	// 获取发布信息
	release, err := g.getRelease(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("获取GitHub发布信息失败: %w", err)
	}

	// 匹配当前平台的资产
	asset, err := g.matchAsset(release.Assets, types.GetCurrentPlatform())
	if err != nil {
		return nil, fmt.Errorf("匹配平台资产失败: %w", err)
	}

	return &types.DownloadInfo{
		URL:      asset.BrowserDownloadURL,
		Filename: asset.Name,
		Size:     asset.Size,
	}, nil
}

// GetDownloadURL 获取下载链接
func (g *GitHubStrategy) GetDownloadURL(ctx context.Context, version string) (string, error) {
	downloadInfo, err := g.GetDownloadInfo(ctx, version)
	if err != nil {
		return "", err
	}
	return downloadInfo.URL, nil
}

// Download 执行下载
func (g *GitHubStrategy) Download(ctx context.Context, url, targetPath string, options *DownloadOptions) error {
	return g.downloader.Download(ctx, url, targetPath, options)
}

// DownloadWithProgress 带进度的下载
func (g *GitHubStrategy) DownloadWithProgress(ctx context.Context, url, targetPath string, options *DownloadOptions, progress ProgressCallback) error {
	return g.downloader.DownloadWithProgress(ctx, url, targetPath, options, progress)
}

// ExtractArchive 解压下载的压缩包
func (g *GitHubStrategy) ExtractArchive(archivePath, targetPath string) error {
	_, err := g.extractor.ProcessPackage(archivePath, targetPath, g.metadata.Name, g.metadata)
	return err
}

// GetLatestVersion 获取最新版本
func (g *GitHubStrategy) GetLatestVersion(ctx context.Context) (string, error) {
	g.logger.Debugf("获取最新版本: %s", g.metadata.Name)

	// 调用GitHub API获取最新发布
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", g.metadata.DownloadConfig.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置GitHub API请求头
	g.setGitHubHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求GitHub API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API请求失败，状态码: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	return g.normalizeVersion(release.TagName), nil
}

// ListVersions 列出所有可用版本
func (g *GitHubStrategy) ListVersions(ctx context.Context) ([]*types.VersionInfo, error) {
	g.logger.Debugf("列出所有版本: %s", g.metadata.Name)

	releases, err := g.getAllReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取所有发布失败: %w", err)
	}

	var versions []*types.VersionInfo
	platform := types.GetCurrentPlatform()

	for _, release := range releases {
		// 跳过草稿版本
		if release.Draft {
			continue
		}

		// 检查是否有适合当前平台的资产
		if asset, err := g.matchAsset(release.Assets, platform); err == nil {
			versionInfo := &types.VersionInfo{
				Version:      g.normalizeVersion(release.TagName),
				ReleaseDate:  release.PublishedAt,
				ChangeLog:    release.Body,
				IsPrerelease: release.Prerelease,
				IsStable:     !release.Prerelease,
				Downloads: map[string]types.DownloadInfo{
					platform.GetPlatformKey(): {
						URL:      asset.BrowserDownloadURL,
						Filename: asset.Name,
						Size:     asset.Size,
					},
				},
			}
			versions = append(versions, versionInfo)
		}
	}

	// 按版本排序
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})

	return versions, nil
}

// ValidateVersion 验证版本是否存在
func (g *GitHubStrategy) ValidateVersion(ctx context.Context, version string) error {
	_, err := g.getRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("版本不存在: %s", version)
	}
	return nil
}

// GetChecksum 获取文件校验和
func (g *GitHubStrategy) GetChecksum(ctx context.Context, version string) (string, error) {
	// GitHub通常不直接提供校验和，可以尝试查找checksums文件
	release, err := g.getRelease(ctx, version)
	if err != nil {
		return "", err
	}

	// 查找校验和文件
	for _, asset := range release.Assets {
		if g.isChecksumFile(asset.Name) {
			// 下载并解析校验和文件
			return g.parseChecksumFile(ctx, asset.BrowserDownloadURL, version)
		}
	}

	return "", nil // 没有找到校验和
}

// SupportsResume 是否支持断点续传
func (g *GitHubStrategy) SupportsResume() bool {
	return true // GitHub一般支持Range请求
}

// GetToolMetadata 获取工具元数据
func (g *GitHubStrategy) GetToolMetadata() *types.ToolMetadata {
	return g.metadata
}

// 私有方法

// getRelease 获取指定版本的发布信息
func (g *GitHubStrategy) getRelease(ctx context.Context, version string) (*GitHubRelease, error) {
	// 规范化版本号
	normalizedVersion := g.normalizeVersionForAPI(version)

	// 尝试通过tag获取
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s",
		g.metadata.DownloadConfig.Repository, normalizedVersion)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	g.setGitHubHeaders(req)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求GitHub API失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("版本不存在: %s", version)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API请求失败，状态码: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &release, nil
}

// getAllReleases 获取所有发布
func (g *GitHubStrategy) getAllReleases(ctx context.Context) ([]GitHubRelease, error) {
	var allReleases []GitHubRelease
	page := 1
	perPage := 50

	for {
		apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases?page=%d&per_page=%d",
			g.metadata.DownloadConfig.Repository, page, perPage)

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}

		g.setGitHubHeaders(req)

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("请求GitHub API失败: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API请求失败，状态码: %d", resp.StatusCode)
		}

		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		resp.Body.Close()

		if len(releases) == 0 {
			break
		}

		allReleases = append(allReleases, releases...)

		// 如果返回的结果少于每页数量，说明是最后一页
		if len(releases) < perPage {
			break
		}

		page++
	}

	return allReleases, nil
}

// matchAsset 匹配平台资产
func (g *GitHubStrategy) matchAsset(assets []GitHubAsset, platform *types.PlatformInfo) (*GitHubAsset, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("没有可用的资产")
	}

	// 如果配置了资产模式，使用模式匹配
	if g.metadata.DownloadConfig.AssetPattern != "" {
		return g.matchAssetByPattern(assets, platform)
	}

	// 默认匹配逻辑
	return g.matchAssetByDefault(assets, platform)
}

// matchAssetByPattern 使用模式匹配资产
func (g *GitHubStrategy) matchAssetByPattern(assets []GitHubAsset, platform *types.PlatformInfo) (*GitHubAsset, error) {
	pattern := g.metadata.DownloadConfig.AssetPattern

	// 替换模式中的变量
	osName := g.mapOSName(platform.OS)
	archName := g.mapArchName(platform.Arch)
	pattern = strings.ReplaceAll(pattern, "{os}", osName)
	pattern = strings.ReplaceAll(pattern, "{arch}", archName)

	g.logger.Debugf("平台信息: OS=%s, Arch=%s", platform.OS, platform.Arch)
	g.logger.Debugf("映射后: OS=%s, Arch=%s", osName, archName)
	g.logger.Debugf("资产模式: %s → %s", g.metadata.DownloadConfig.AssetPattern, pattern)

	// 编译正则表达式
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("无效的资产模式: %w", err)
	}

	// 显示所有可用资产以便调试
	g.logger.Debugf("可用资产:")
	for _, asset := range assets {
		g.logger.Debugf("  - %s", asset.Name)
	}

	// 匹配资产
	for _, asset := range assets {
		if regex.MatchString(asset.Name) {
			g.logger.Debugf("匹配成功: %s", asset.Name)
			return &asset, nil
		}
	}

	return nil, fmt.Errorf("没有找到匹配模式的资产: %s", pattern)
}

// matchAssetByDefault 默认资产匹配
func (g *GitHubStrategy) matchAssetByDefault(assets []GitHubAsset, platform *types.PlatformInfo) (*GitHubAsset, error) {
	// 支持多种操作系统命名约定
	osNames := []string{platform.OS}
	switch platform.OS {
	case "darwin":
		osNames = append(osNames, "macos", "osx", "mac")
	case "linux":
		osNames = append(osNames, "Linux")
	case "windows":
		osNames = append(osNames, "win", "Win", "Windows")
	}

	// 支持多种架构命名约定
	archNames := []string{platform.Arch}
	switch platform.Arch {
	case "amd64":
		archNames = append(archNames, "x86_64", "x64", "64bit")
	case "arm64":
		archNames = append(archNames, "aarch64", "arm")
	case "386":
		archNames = append(archNames, "i386", "x86", "32bit")
	}

	// 首先尝试精确匹配
	for _, asset := range assets {
		assetName := strings.ToLower(asset.Name)

		osMatch := false
		for _, osName := range osNames {
			if strings.Contains(assetName, strings.ToLower(osName)) {
				osMatch = true
				break
			}
		}

		if osMatch {
			for _, archName := range archNames {
				if strings.Contains(assetName, strings.ToLower(archName)) {
					return &asset, nil
				}
			}
		}
	}

	// 如果没有精确匹配，尝试只匹配操作系统
	for _, asset := range assets {
		assetName := strings.ToLower(asset.Name)

		for _, osName := range osNames {
			if strings.Contains(assetName, strings.ToLower(osName)) {
				return &asset, nil
			}
		}
	}

	// 如果还是没有，返回第一个
	if len(assets) > 0 {
		return &assets[0], nil
	}

	return nil, fmt.Errorf("没有找到适合的资产")
}

// setGitHubHeaders 设置GitHub API请求头
func (g *GitHubStrategy) setGitHubHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "vman/1.0")

	// 如果配置了GitHub Token
	if g.metadata.DownloadConfig.Headers != nil {
		for key, value := range g.metadata.DownloadConfig.Headers {
			req.Header.Set(key, value)
		}
	}
}

// mapOSName 映射操作系统名称
func (g *GitHubStrategy) mapOSName(os string) string {
	// 为了与工具配置文件中的asset_pattern保持一致，直接返回原始操作系统名称
	return os
}

// mapArchName 映射架构名称
func (g *GitHubStrategy) mapArchName(arch string) string {
	// 为了与工具配置文件中的asset_pattern保持一致，直接返回原始架构名称
	return arch
}

// normalizeVersion 规范化版本号
func (g *GitHubStrategy) normalizeVersion(version string) string {
	// 移除 'v' 前缀
	return strings.TrimPrefix(version, "v")
}

// normalizeVersionForAPI 为API调用规范化版本号
func (g *GitHubStrategy) normalizeVersionForAPI(version string) string {
	// 如果没有 'v' 前缀，添加它
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// isChecksumFile 判断是否为校验和文件
func (g *GitHubStrategy) isChecksumFile(filename string) bool {
	checksumPatterns := []string{
		"checksums", "checksum", "sha256", "sha512", "md5",
	}

	filename = strings.ToLower(filename)
	for _, pattern := range checksumPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}

	return false
}

// parseChecksumFile 解析校验和文件
func (g *GitHubStrategy) parseChecksumFile(ctx context.Context, url, version string) (string, error) {
	// 这里应该下载校验和文件并解析
	// 为了简化，现在返回空字符串
	return "", nil
}
