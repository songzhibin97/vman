package version

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/spf13/afero"
)

// SetGlobalVersion 设置全局版本
func (m *DefaultManager) SetGlobalVersion(tool, version string) error {
	m.logger.Debugf("Setting global version %s@%s", tool, version)

	if !m.IsVersionInstalled(tool, version) {
		return fmt.Errorf("version %s@%s is not installed", tool, version)
	}

	return m.configManager.SetToolVersion(tool, version, true, "")
}

// SetLocalVersion 设置项目级版本（当前目录）
func (m *DefaultManager) SetLocalVersion(tool, version string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	return m.SetProjectVersion(tool, version, cwd)
}

// SetProjectVersion 设置项目版本（带项目路径）
func (m *DefaultManager) SetProjectVersion(tool, version, projectPath string) error {
	m.logger.Debugf("Setting project version %s@%s for project %s", tool, version, projectPath)

	if !m.IsVersionInstalled(tool, version) {
		return fmt.Errorf("version %s@%s is not installed", tool, version)
	}

	return m.configManager.SetToolVersion(tool, version, false, projectPath)
}

// GetCurrentVersion 获取当前使用的版本
func (m *DefaultManager) GetCurrentVersion(tool string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return m.GetEffectiveVersion(tool, cwd)
}

// GetEffectiveVersion 获取有效版本（考虑项目和全局配置）
func (m *DefaultManager) GetEffectiveVersion(tool, projectPath string) (string, error) {
	return m.configManager.GetEffectiveVersion(tool, projectPath)
}

// IsVersionInstalled 检查版本是否已安装
func (m *DefaultManager) IsVersionInstalled(tool, version string) bool {
	return m.storageManager.IsVersionInstalled(tool, version)
}

// GetInstalledVersions 获取已安装版本列表
func (m *DefaultManager) GetInstalledVersions(tool string) ([]string, error) {
	return m.storageManager.GetToolVersions(tool)
}

// ValidateVersion 验证版本格式
func (m *DefaultManager) ValidateVersion(version string) error {
	// 支持semver格式
	if _, err := semver.NewVersion(version); err == nil {
		return nil
	}

	// 支持简单的版本号格式（如 1.2.3, v1.2.3）
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}

	// 检查基本格式
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return fmt.Errorf("invalid version format: %s", version)
	}

	// 检查每个部分都是数字或包含数字
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid version format: %s", version)
		}
	}

	return nil
}

// GetLatestVersion 获取最新版本
func (m *DefaultManager) GetLatestVersion(tool string) (string, error) {
	versions, err := m.GetInstalledVersions(tool)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions installed for tool %s", tool)
	}

	// 尝试使用semver排序
	var semverVersions []*semver.Version
	var nonSemverVersions []string

	for _, v := range versions {
		cleanV := v
		if strings.HasPrefix(v, "v") {
			cleanV = v[1:]
		}

		if sv, err := semver.NewVersion(cleanV); err == nil {
			semverVersions = append(semverVersions, sv)
		} else {
			nonSemverVersions = append(nonSemverVersions, v)
		}
	}

	// 如果有semver版本，返回最高版本
	if len(semverVersions) > 0 {
		sort.Sort(semver.Collection(semverVersions))
		latest := semverVersions[len(semverVersions)-1]
		// 保持原始格式（是否带v前缀）
		for _, v := range versions {
			cleanV := v
			if strings.HasPrefix(v, "v") {
				cleanV = v[1:]
			}
			if cleanV == latest.String() {
				return v, nil
			}
		}
	}

	// 否则按字符串排序返回最后一个
	sort.Strings(nonSemverVersions)
	if len(nonSemverVersions) > 0 {
		return nonSemverVersions[len(nonSemverVersions)-1], nil
	}

	return versions[0], nil
}

// GetVersionMetadata 获取版本元数据
func (m *DefaultManager) GetVersionMetadata(tool, version string) (*types.VersionMetadata, error) {
	return m.storageManager.LoadVersionMetadata(tool, version)
}

// ListAllTools 列出所有已安装的工具
func (m *DefaultManager) ListAllTools() ([]string, error) {
	versionsDir := m.storageManager.GetVersionsDir()
	entries, err := afero.ReadDir(m.fs, versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var tools []string
	for _, entry := range entries {
		if entry.IsDir() {
			toolName := entry.Name()
			// 使用更严格的验证：直接检查工具目录下是否有版本目录
			toolDir := filepath.Join(versionsDir, toolName)
			toolEntries, err := afero.ReadDir(m.fs, toolDir)
			if err != nil {
				continue
			}
			
			// 检查是否有有效的版本目录
			hasValidVersion := false
			for _, versionEntry := range toolEntries {
				if versionEntry.IsDir() {
					versionName := versionEntry.Name()
					// 使用存储管理器的IsVersionInstalled方法验证
					if m.storageManager.IsVersionInstalled(toolName, versionName) {
						hasValidVersion = true
						break
					}
				}
			}
			
			if hasValidVersion {
				tools = append(tools, toolName)
			}
		}
	}

	sort.Strings(tools)
	return tools, nil
}

// copyBinary 复制二进制文件
func (m *DefaultManager) copyBinary(sourcePath, targetPath string) error {
	// 确保目标目录存在
	targetDir := filepath.Dir(targetPath)
	if err := m.fs.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// 打开源文件
	src, err := m.fs.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := m.fs.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// 设置可执行权限
	if err := m.fs.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	return nil
}

// updateInstalledVersions 更新配置中的已安装版本
func (m *DefaultManager) updateInstalledVersions(tool, version string) error {
	config, err := m.configManager.LoadGlobal()
	if err != nil {
		return err
	}

	if config.Tools == nil {
		config.Tools = make(map[string]types.ToolInfo)
	}

	toolInfo := config.Tools[tool]

	// 添加到已安装版本列表（如果不存在）
	found := false
	for _, v := range toolInfo.InstalledVersions {
		if v == version {
			found = true
			break
		}
	}
	if !found {
		toolInfo.InstalledVersions = append(toolInfo.InstalledVersions, version)
		sort.Strings(toolInfo.InstalledVersions)
	}

	config.Tools[tool] = toolInfo
	return m.configManager.SaveGlobal(config)
}

// InstallVersion 自动下载并安装工具版本 (基础版本不支持)
func (m *DefaultManager) InstallVersion(tool, version string) error {
	return fmt.Errorf("基础版本管理器不支持自动下载，请使用 register 命令手动注册")
}

// InstallVersionWithProgress 带进度显示的安装 (基础版本不支持)
func (m *DefaultManager) InstallVersionWithProgress(tool, version string, progress ProgressCallback) error {
	return fmt.Errorf("基础版本管理器不支持自动下载，请使用 register 命令手动注册")
}

// InstallLatestVersion 安装最新版本 (基础版本不支持)
func (m *DefaultManager) InstallLatestVersion(tool string) (string, error) {
	return "", fmt.Errorf("基础版本管理器不支持自动下载，请使用 register 命令手动注册")
}

// SearchAvailableVersions 搜索可用版本 (基础版本不支持)
func (m *DefaultManager) SearchAvailableVersions(tool string) ([]*types.VersionInfo, error) {
	return nil, fmt.Errorf("基础版本管理器不支持搜索功能，请使用集成版本管理器")
}

// IsVersionAvailable 检查版本是否可下载 (基础版本不支持)
func (m *DefaultManager) IsVersionAvailable(tool, version string) bool {
	return false
}

// UpdateTool 更新工具到最新版本 (基础版本不支持)
func (m *DefaultManager) UpdateTool(tool string) (string, error) {
	return "", fmt.Errorf("基础版本管理器不支持更新功能，请使用集成版本管理器")
}

// removeFromInstalledVersions 从配置中移除已安装版本
func (m *DefaultManager) removeFromInstalledVersions(tool, version string) error {
	config, err := m.configManager.LoadGlobal()
	if err != nil {
		return err
	}

	if config.Tools == nil {
		return nil
	}

	toolInfo := config.Tools[tool]

	// 从已安装版本列表中移除
	var newVersions []string
	for _, v := range toolInfo.InstalledVersions {
		if v != version {
			newVersions = append(newVersions, v)
		}
	}
	toolInfo.InstalledVersions = newVersions

	// 如果移除的是当前版本，清空当前版本
	if toolInfo.CurrentVersion == version {
		toolInfo.CurrentVersion = ""
	}

	config.Tools[tool] = toolInfo
	return m.configManager.SaveGlobal(config)
}
