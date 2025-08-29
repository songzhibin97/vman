package download

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/songzhibin97/vman/pkg/types"
)

// VersionDiscovery 版本发现接口
type VersionDiscovery interface {
	// DiscoverVersions 发现可用版本
	DiscoverVersions(ctx context.Context) ([]*types.VersionInfo, error)

	// GetLatestVersion 获取最新版本
	GetLatestVersion(ctx context.Context) (string, error)

	// FilterByPlatform 根据平台过滤版本
	FilterByPlatform(versions []*types.VersionInfo, platform *types.PlatformInfo) []*types.VersionInfo

	// SortVersions 版本排序
	SortVersions(versions []*types.VersionInfo) []*types.VersionInfo
}

// PlatformMatcher 平台匹配器接口
type PlatformMatcher interface {
	// Match 匹配平台
	Match(platform *types.PlatformInfo) bool

	// GetPattern 获取匹配模式
	GetPattern() string

	// IsArchiveSupported 是否支持压缩包格式
	IsArchiveSupported(filename string) bool
}

// DefaultVersionDiscovery 默认版本发现实现
type DefaultVersionDiscovery struct {
	strategy Strategy
	matcher  PlatformMatcher
}

// NewVersionDiscovery 创建版本发现器
func NewVersionDiscovery(strategy Strategy) VersionDiscovery {
	return &DefaultVersionDiscovery{
		strategy: strategy,
		matcher:  NewDefaultPlatformMatcher(),
	}
}

// DiscoverVersions 发现可用版本
func (d *DefaultVersionDiscovery) DiscoverVersions(ctx context.Context) ([]*types.VersionInfo, error) {
	versions, err := d.strategy.ListVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %w", err)
	}

	// 获取当前平台信息
	platform := types.GetCurrentPlatform()

	// 过滤适用于当前平台的版本
	filteredVersions := d.FilterByPlatform(versions, platform)

	// 排序版本
	sortedVersions := d.SortVersions(filteredVersions)

	return sortedVersions, nil
}

// GetLatestVersion 获取最新版本
func (d *DefaultVersionDiscovery) GetLatestVersion(ctx context.Context) (string, error) {
	versions, err := d.DiscoverVersions(ctx)
	if err != nil {
		return "", fmt.Errorf("发现版本失败: %w", err)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("未找到可用版本")
	}

	// 过滤预发布版本，优先选择稳定版本
	for _, version := range versions {
		if !version.IsPrerelease {
			return version.Version, nil
		}
	}

	// 如果没有稳定版本，返回最新的预发布版本
	return versions[0].Version, nil
}

// FilterByPlatform 根据平台过滤版本
func (d *DefaultVersionDiscovery) FilterByPlatform(versions []*types.VersionInfo, platform *types.PlatformInfo) []*types.VersionInfo {
	var filtered []*types.VersionInfo

	for _, version := range versions {
		if d.hasCompatibleDownload(version, platform) {
			filtered = append(filtered, version)
		}
	}

	return filtered
}

// SortVersions 版本排序
func (d *DefaultVersionDiscovery) SortVersions(versions []*types.VersionInfo) []*types.VersionInfo {
	sorted := make([]*types.VersionInfo, len(versions))
	copy(sorted, versions)

	sort.Slice(sorted, func(i, j int) bool {
		return compareVersions(sorted[i].Version, sorted[j].Version) > 0
	})

	return sorted
}

// hasCompatibleDownload 检查版本是否有兼容的下载
func (d *DefaultVersionDiscovery) hasCompatibleDownload(version *types.VersionInfo, platform *types.PlatformInfo) bool {
	// 检查是否有精确匹配的平台
	platformKey := platform.GetPlatformKey()
	if _, exists := version.Downloads[platformKey]; exists {
		return true
	}

	// 检查是否有通用二进制文件
	for key := range version.Downloads {
		if strings.Contains(key, platform.OS) {
			return true
		}
	}

	return false
}

// DefaultPlatformMatcher 默认平台匹配器
type DefaultPlatformMatcher struct {
	osPatterns   map[string][]string
	archPatterns map[string][]string
}

// NewDefaultPlatformMatcher 创建默认平台匹配器
func NewDefaultPlatformMatcher() PlatformMatcher {
	return &DefaultPlatformMatcher{
		osPatterns: map[string][]string{
			"linux":   {"linux", "Linux"},
			"darwin":  {"darwin", "macOS", "osx", "Darwin"},
			"windows": {"windows", "win", "Windows", "Win"},
		},
		archPatterns: map[string][]string{
			"amd64": {"amd64", "x86_64", "x64", "64bit"},
			"arm64": {"arm64", "aarch64", "arm"},
			"386":   {"386", "i386", "x86", "32bit"},
		},
	}
}

// Match 匹配平台
func (m *DefaultPlatformMatcher) Match(platform *types.PlatformInfo) bool {
	osPatterns := m.osPatterns[platform.OS]
	archPatterns := m.archPatterns[platform.Arch]

	return len(osPatterns) > 0 && len(archPatterns) > 0
}

// GetPattern 获取匹配模式
func (m *DefaultPlatformMatcher) GetPattern() string {
	return fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
}

// IsArchiveSupported 是否支持压缩包格式
func (m *DefaultPlatformMatcher) IsArchiveSupported(filename string) bool {
	supportedExt := []string{".tar.gz", ".tgz", ".zip", ".tar.bz2", ".tar.xz"}

	for _, ext := range supportedExt {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return true
		}
	}

	return false
}

// VersionParser 版本解析器
type VersionParser struct {
	patterns map[string]*regexp.Regexp
}

// NewVersionParser 创建版本解析器
func NewVersionParser() *VersionParser {
	return &VersionParser{
		patterns: map[string]*regexp.Regexp{
			"semver": regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?(?:\+([a-zA-Z0-9.-]+))?$`),
			"simple": regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?$`),
			"date":   regexp.MustCompile(`^v?(\d{4})[-.]?(\d{2})[-.]?(\d{2})$`),
			"build":  regexp.MustCompile(`^v?(\d+)[-.]?([a-zA-Z0-9]+)$`),
		},
	}
}

// ParseVersion 解析版本
func (p *VersionParser) ParseVersion(version string) (*ParsedVersion, error) {
	// 移除常见的前缀
	cleanVersion := strings.TrimPrefix(version, "v")
	cleanVersion = strings.TrimPrefix(cleanVersion, "release-")

	// 尝试各种模式
	for patternType, pattern := range p.patterns {
		if matches := pattern.FindStringSubmatch(cleanVersion); matches != nil {
			return p.createParsedVersion(patternType, matches, version)
		}
	}

	return nil, fmt.Errorf("无法解析版本: %s", version)
}

// createParsedVersion 创建解析后的版本
func (p *VersionParser) createParsedVersion(patternType string, matches []string, original string) (*ParsedVersion, error) {
	parsed := &ParsedVersion{
		Original: original,
		Type:     patternType,
	}

	switch patternType {
	case "semver":
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch, _ := strconv.Atoi(matches[3])
		parsed.Major = major
		parsed.Minor = minor
		parsed.Patch = patch
		if len(matches) > 4 && matches[4] != "" {
			parsed.Prerelease = matches[4]
		}
		if len(matches) > 5 && matches[5] != "" {
			parsed.Build = matches[5]
		}

	case "simple":
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		parsed.Major = major
		parsed.Minor = minor
		if len(matches) > 3 && matches[3] != "" {
			patch, _ := strconv.Atoi(matches[3])
			parsed.Patch = patch
		}

	case "date":
		year, _ := strconv.Atoi(matches[1])
		month, _ := strconv.Atoi(matches[2])
		day, _ := strconv.Atoi(matches[3])
		parsed.Date = &time.Time{}
		*parsed.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	case "build":
		build, _ := strconv.Atoi(matches[1])
		parsed.Build = matches[2]
		parsed.Major = build
	}

	return parsed, nil
}

// ParsedVersion 解析后的版本
type ParsedVersion struct {
	Original   string     // 原始版本字符串
	Type       string     // 版本类型 (semver, simple, date, build)
	Major      int        // 主版本号
	Minor      int        // 次版本号
	Patch      int        // 修订版本号
	Prerelease string     // 预发布标识
	Build      string     // 构建元数据
	Date       *time.Time // 日期版本
}

// IsPrerelease 是否为预发布版本
func (v *ParsedVersion) IsPrerelease() bool {
	return v.Prerelease != ""
}

// IsNewer 是否比另一个版本更新
func (v *ParsedVersion) IsNewer(other *ParsedVersion) bool {
	if v.Type != other.Type {
		// 不同类型的版本比较复杂，这里简化处理
		return false
	}

	switch v.Type {
	case "semver", "simple":
		return compareSemVer(v, other) > 0
	case "date":
		if v.Date != nil && other.Date != nil {
			return v.Date.After(*other.Date)
		}
	case "build":
		return v.Major > other.Major
	}

	return false
}

// compareSemVer 比较语义版本
func compareSemVer(a, b *ParsedVersion) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	if a.Patch != b.Patch {
		return a.Patch - b.Patch
	}

	// 比较预发布版本
	if a.Prerelease == "" && b.Prerelease == "" {
		return 0
	}
	if a.Prerelease == "" && b.Prerelease != "" {
		return 1 // 正式版本 > 预发布版本
	}
	if a.Prerelease != "" && b.Prerelease == "" {
		return -1 // 预发布版本 < 正式版本
	}

	// 都是预发布版本，按字符串比较
	return strings.Compare(a.Prerelease, b.Prerelease)
}

// compareVersions 比较版本字符串
func compareVersions(a, b string) int {
	parser := NewVersionParser()

	versionA, errA := parser.ParseVersion(a)
	versionB, errB := parser.ParseVersion(b)

	// 如果解析失败，使用字符串比较
	if errA != nil || errB != nil {
		return strings.Compare(a, b)
	}

	if versionA.IsNewer(versionB) {
		return 1
	} else if versionB.IsNewer(versionA) {
		return -1
	}

	return 0
}

// PlatformResolver 平台解析器
type PlatformResolver struct {
	osMapping   map[string]string
	archMapping map[string]string
}

// NewPlatformResolver 创建平台解析器
func NewPlatformResolver() *PlatformResolver {
	return &PlatformResolver{
		osMapping: map[string]string{
			"darwin":  "macOS",
			"linux":   "Linux",
			"windows": "Windows",
		},
		archMapping: map[string]string{
			"amd64": "x86_64",
			"arm64": "ARM64",
			"386":   "i386",
		},
	}
}

// GetCurrentPlatform 获取当前平台信息
func (r *PlatformResolver) GetCurrentPlatform() *types.PlatformInfo {
	return &types.PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

// NormalizePlatform 标准化平台信息
func (r *PlatformResolver) NormalizePlatform(platform *types.PlatformInfo) *types.PlatformInfo {
	normalized := &types.PlatformInfo{
		OS:   platform.OS,
		Arch: platform.Arch,
	}

	// 标准化操作系统名称
	if mappedOS, exists := r.osMapping[platform.OS]; exists {
		normalized.OS = mappedOS
	}

	// 标准化架构名称
	if mappedArch, exists := r.archMapping[platform.Arch]; exists {
		normalized.Arch = mappedArch
	}

	return normalized
}

// MatchesPlatform 检查是否匹配平台
func (r *PlatformResolver) MatchesPlatform(filename string, platform *types.PlatformInfo) bool {
	filename = strings.ToLower(filename)

	// 检查操作系统匹配
	osPatterns := []string{
		strings.ToLower(platform.OS),
	}

	if mapped, exists := r.osMapping[platform.OS]; exists {
		osPatterns = append(osPatterns, strings.ToLower(mapped))
	}

	osMatch := false
	for _, pattern := range osPatterns {
		if strings.Contains(filename, pattern) {
			osMatch = true
			break
		}
	}

	if !osMatch {
		return false
	}

	// 检查架构匹配
	archPatterns := []string{
		strings.ToLower(platform.Arch),
	}

	if mapped, exists := r.archMapping[platform.Arch]; exists {
		archPatterns = append(archPatterns, strings.ToLower(mapped))
	}

	for _, pattern := range archPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}

	return false
}
