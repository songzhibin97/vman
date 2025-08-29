package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// PathManager PATH环境变量管理器接口
type PathManager interface {
	// AddToPath 将目录添加到PATH环境变量
	AddToPath(dir string) error

	// RemoveFromPath 从PATH环境变量中移除目录
	RemoveFromPath(dir string) error

	// IsInPath 检查目录是否在PATH中
	IsInPath(dir string) bool

	// GetPathDirs 获取当前PATH中的所有目录
	GetPathDirs() []string

	// SetupShimPath 设置shim目录到PATH
	SetupShimPath(shimDir string) error

	// CleanupShimPath 从PATH中清理shim目录
	CleanupShimPath(shimDir string) error

	// BackupPath 备份当前PATH设置
	BackupPath() error

	// RestorePath 恢复PATH设置
	RestorePath() error

	// GetShellProfile 获取shell配置文件路径
	GetShellProfile() string

	// UpdateShellProfile 更新shell配置文件
	UpdateShellProfile(content string) error
}

// DefaultPathManager 默认PATH管理器实现
type DefaultPathManager struct {
	fs       afero.Fs
	logger   *logrus.Logger
	shell    string // 当前使用的shell
	homePath string // 用户主目录
}

// NewPathManager 创建新的PATH管理器
func NewPathManager() PathManager {
	return NewPathManagerWithFs(afero.NewOsFs())
}

// NewPathManagerWithFs 使用指定文件系统创建PATH管理器（用于测试）
func NewPathManagerWithFs(fs afero.Fs) PathManager {
	homeDir, _ := os.UserHomeDir()
	shell := detectShell()

	return &DefaultPathManager{
		fs:       fs,
		logger:   logrus.New(),
		shell:    shell,
		homePath: homeDir,
	}
}

// detectShell 检测当前使用的shell
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		// 在Windows上默认使用cmd
		if runtime.GOOS == "windows" {
			return "cmd"
		}
		return "bash"
	}

	// 提取shell名称
	shellName := filepath.Base(shell)
	return shellName
}

// AddToPath 将目录添加到PATH环境变量
func (pm *DefaultPathManager) AddToPath(dir string) error {
	pm.logger.Debugf("Adding directory to PATH: %s", dir)

	// 检查目录是否存在
	if _, err := pm.fs.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}

	// 检查是否已在PATH中
	if pm.IsInPath(dir) {
		pm.logger.Debugf("Directory already in PATH: %s", dir)
		return nil
	}

	// 获取当前PATH
	currentPath := os.Getenv("PATH")
	pathSeparator := getPathSeparator()

	// 将新目录添加到PATH开头（优先级最高）
	var newPath string
	if currentPath == "" {
		newPath = dir
	} else {
		newPath = dir + pathSeparator + currentPath
	}

	// 设置新的PATH
	if err := os.Setenv("PATH", newPath); err != nil {
		return fmt.Errorf("failed to set PATH: %w", err)
	}

	pm.logger.Infof("Successfully added %s to PATH", dir)
	return nil
}

// RemoveFromPath 从PATH环境变量中移除目录
func (pm *DefaultPathManager) RemoveFromPath(dir string) error {
	pm.logger.Debugf("Removing directory from PATH: %s", dir)

	if !pm.IsInPath(dir) {
		pm.logger.Debugf("Directory not in PATH: %s", dir)
		return nil
	}

	// 获取当前PATH
	currentPath := os.Getenv("PATH")
	pathSeparator := getPathSeparator()

	// 分割PATH并移除指定目录
	pathDirs := strings.Split(currentPath, pathSeparator)
	var newPathDirs []string

	for _, pathDir := range pathDirs {
		// 清理路径并比较
		cleanDir := filepath.Clean(pathDir)
		cleanTarget := filepath.Clean(dir)

		if cleanDir != cleanTarget {
			newPathDirs = append(newPathDirs, pathDir)
		}
	}

	// 重新构建PATH
	newPath := strings.Join(newPathDirs, pathSeparator)

	// 设置新的PATH
	if err := os.Setenv("PATH", newPath); err != nil {
		return fmt.Errorf("failed to set PATH: %w", err)
	}

	pm.logger.Infof("Successfully removed %s from PATH", dir)
	return nil
}

// IsInPath 检查目录是否在PATH中
func (pm *DefaultPathManager) IsInPath(dir string) bool {
	pathDirs := pm.GetPathDirs()
	cleanTarget := filepath.Clean(dir)

	for _, pathDir := range pathDirs {
		cleanDir := filepath.Clean(pathDir)
		if cleanDir == cleanTarget {
			return true
		}
	}

	return false
}

// GetPathDirs 获取当前PATH中的所有目录
func (pm *DefaultPathManager) GetPathDirs() []string {
	currentPath := os.Getenv("PATH")
	if currentPath == "" {
		return []string{}
	}

	pathSeparator := getPathSeparator()
	return strings.Split(currentPath, pathSeparator)
}

// SetupShimPath 设置shim目录到PATH
func (pm *DefaultPathManager) SetupShimPath(shimDir string) error {
	pm.logger.Infof("Setting up shim path: %s", shimDir)

	// 确保shim目录存在
	if err := pm.fs.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	// 添加到PATH
	if err := pm.AddToPath(shimDir); err != nil {
		return fmt.Errorf("failed to add shim directory to PATH: %w", err)
	}

	// 更新shell配置文件以持久化PATH设置
	if err := pm.updateShellConfiguration(shimDir, true); err != nil {
		pm.logger.Warnf("Failed to update shell configuration: %v", err)
		// 不返回错误，因为PATH已经在当前会话中设置
	}

	return nil
}

// CleanupShimPath 从PATH中清理shim目录
func (pm *DefaultPathManager) CleanupShimPath(shimDir string) error {
	pm.logger.Infof("Cleaning up shim path: %s", shimDir)

	// 从PATH中移除
	if err := pm.RemoveFromPath(shimDir); err != nil {
		return fmt.Errorf("failed to remove shim directory from PATH: %w", err)
	}

	// 更新shell配置文件
	if err := pm.updateShellConfiguration(shimDir, false); err != nil {
		pm.logger.Warnf("Failed to update shell configuration: %v", err)
	}

	return nil
}

// BackupPath 备份当前PATH设置
func (pm *DefaultPathManager) BackupPath() error {
	currentPath := os.Getenv("PATH")
	backupFile := filepath.Join(pm.homePath, ".vman", "path_backup")

	// 确保备份目录存在
	if err := pm.fs.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// 写入备份文件
	if err := afero.WriteFile(pm.fs, backupFile, []byte(currentPath), 0644); err != nil {
		return fmt.Errorf("failed to write PATH backup: %w", err)
	}

	pm.logger.Infof("PATH backup saved to: %s", backupFile)
	return nil
}

// RestorePath 恢复PATH设置
func (pm *DefaultPathManager) RestorePath() error {
	backupFile := filepath.Join(pm.homePath, ".vman", "path_backup")

	// 检查备份文件是否存在
	if _, err := pm.fs.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("PATH backup not found: %s", backupFile)
	}

	// 读取备份内容
	backupContent, err := afero.ReadFile(pm.fs, backupFile)
	if err != nil {
		return fmt.Errorf("failed to read PATH backup: %w", err)
	}

	// 恢复PATH
	if err := os.Setenv("PATH", string(backupContent)); err != nil {
		return fmt.Errorf("failed to restore PATH: %w", err)
	}

	pm.logger.Infof("PATH restored from backup")
	return nil
}

// GetShellProfile 获取shell配置文件路径
func (pm *DefaultPathManager) GetShellProfile() string {
	switch pm.shell {
	case "bash":
		// 优先检查 .bash_profile，然后是 .bashrc
		bashProfile := filepath.Join(pm.homePath, ".bash_profile")
		if exists, _ := afero.Exists(pm.fs, bashProfile); exists {
			return bashProfile
		}
		return filepath.Join(pm.homePath, ".bashrc")
	case "zsh":
		return filepath.Join(pm.homePath, ".zshrc")
	case "fish":
		return filepath.Join(pm.homePath, ".config", "fish", "config.fish")
	default:
		// 默认使用 .profile
		return filepath.Join(pm.homePath, ".profile")
	}
}

// UpdateShellProfile 更新shell配置文件
func (pm *DefaultPathManager) UpdateShellProfile(content string) error {
	profilePath := pm.GetShellProfile()

	// 确保目录存在
	if err := pm.fs.MkdirAll(filepath.Dir(profilePath), 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// 写入内容
	if err := afero.WriteFile(pm.fs, profilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write shell profile: %w", err)
	}

	pm.logger.Infof("Updated shell profile: %s", profilePath)
	return nil
}

// updateShellConfiguration 更新shell配置文件中的PATH设置
func (pm *DefaultPathManager) updateShellConfiguration(shimDir string, add bool) error {
	profilePath := pm.GetShellProfile()

	// 生成PATH配置行
	var pathLine string
	switch pm.shell {
	case "fish":
		if add {
			pathLine = fmt.Sprintf("set -gx PATH %s $PATH", shimDir)
		} else {
			// Fish shell的PATH移除比较复杂，这里暂时跳过
			return nil
		}
	default:
		if add {
			pathLine = fmt.Sprintf("export PATH=\"%s:$PATH\"", shimDir)
		} else {
			// Bash/Zsh的PATH移除，这里暂时使用注释
			pathLine = fmt.Sprintf("# export PATH=\"%s:$PATH\"", shimDir)
		}
	}

	// 读取现有内容
	var existingContent []byte
	if exists, _ := afero.Exists(pm.fs, profilePath); exists {
		content, err := afero.ReadFile(pm.fs, profilePath)
		if err != nil {
			return fmt.Errorf("failed to read shell profile: %w", err)
		}
		existingContent = content
	}

	// 检查是否已存在相关配置
	existingLines := strings.Split(string(existingContent), "\n")
	vmanMarker := "# vman shim path"

	// 移除旧的vman配置
	var newLines []string
	inVmanSection := false

	for _, line := range existingLines {
		if strings.Contains(line, vmanMarker) {
			inVmanSection = !inVmanSection
			if add && !inVmanSection {
				// 如果是添加操作且退出vman段，添加新配置
				newLines = append(newLines, vmanMarker)
				newLines = append(newLines, pathLine)
				newLines = append(newLines, vmanMarker)
			}
			continue
		}

		if !inVmanSection {
			newLines = append(newLines, line)
		}
	}

	// 如果是添加操作且没有找到现有配置
	if add && !strings.Contains(string(existingContent), vmanMarker) {
		newLines = append(newLines, "")
		newLines = append(newLines, vmanMarker)
		newLines = append(newLines, pathLine)
		newLines = append(newLines, vmanMarker)
	}

	// 写入更新后的内容
	newContent := strings.Join(newLines, "\n")
	return pm.UpdateShellProfile(newContent)
}

// getPathSeparator 获取路径分隔符
func getPathSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}
