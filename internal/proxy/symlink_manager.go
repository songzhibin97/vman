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

// SymlinkManager 符号链接管理器接口
type SymlinkManager interface {
	// CreateSymlink 创建符号链接
	CreateSymlink(target, linkPath string) error

	// RemoveSymlink 移除符号链接
	RemoveSymlink(linkPath string) error

	// IsSymlink 检查是否为符号链接
	IsSymlink(path string) bool

	// GetSymlinkTarget 获取符号链接的目标
	GetSymlinkTarget(linkPath string) (string, error)

	// UpdateSymlink 更新符号链接目标
	UpdateSymlink(linkPath, newTarget string) error

	// CreateToolSymlinks 为工具创建符号链接
	CreateToolSymlinks(toolName, version, binPath, shimDir string) error

	// RemoveToolSymlinks 移除工具的符号链接
	RemoveToolSymlinks(toolName, shimDir string) error

	// ListSymlinks 列出目录中的所有符号链接
	ListSymlinks(dir string) (map[string]string, error)

	// ValidateSymlinks 验证符号链接的有效性
	ValidateSymlinks(dir string) ([]string, error)

	// CleanupBrokenSymlinks 清理损坏的符号链接
	CleanupBrokenSymlinks(dir string) error
}

// DefaultSymlinkManager 默认符号链接管理器实现
type DefaultSymlinkManager struct {
	fs     afero.Fs
	logger *logrus.Logger
}

// NewSymlinkManager 创建新的符号链接管理器
func NewSymlinkManager() SymlinkManager {
	return NewSymlinkManagerWithFs(afero.NewOsFs())
}

// NewSymlinkManagerWithFs 使用指定文件系统创建符号链接管理器（用于测试）
func NewSymlinkManagerWithFs(fs afero.Fs) SymlinkManager {
	return &DefaultSymlinkManager{
		fs:     fs,
		logger: logrus.New(),
	}
}

// CreateSymlink 创建符号链接
func (sm *DefaultSymlinkManager) CreateSymlink(target, linkPath string) error {
	sm.logger.Debugf("Creating symlink: %s -> %s", linkPath, target)

	// 检查目标文件是否存在
	if _, err := sm.fs.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("target file does not exist: %s", target)
	}

	// 检查链接路径是否已存在
	if _, err := sm.fs.Stat(linkPath); err == nil {
		return fmt.Errorf("symlink already exists: %s", linkPath)
	}

	// 确保目标目录存在
	linkDir := filepath.Dir(linkPath)
	if err := sm.fs.MkdirAll(linkDir, 0755); err != nil {
		return fmt.Errorf("failed to create symlink directory: %w", err)
	}

	// 创建符号链接
	// 注意：afero.Fs接口不支持符号链接，这里直接使用os包
	if err := os.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	sm.logger.Infof("Successfully created symlink: %s -> %s", linkPath, target)
	return nil
}

// RemoveSymlink 移除符号链接
func (sm *DefaultSymlinkManager) RemoveSymlink(linkPath string) error {
	sm.logger.Debugf("Removing symlink: %s", linkPath)

	// 检查是否为符号链接
	if !sm.IsSymlink(linkPath) {
		return fmt.Errorf("path is not a symlink: %s", linkPath)
	}

	// 移除符号链接
	if err := sm.fs.Remove(linkPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	sm.logger.Infof("Successfully removed symlink: %s", linkPath)
	return nil
}

// IsSymlink 检查是否为符号链接
func (sm *DefaultSymlinkManager) IsSymlink(path string) bool {
	// 由于afero不支持Lstat，这里直接使用os.Lstat
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeSymlink != 0
}

// GetSymlinkTarget 获取符号链接的目标
func (sm *DefaultSymlinkManager) GetSymlinkTarget(linkPath string) (string, error) {
	if !sm.IsSymlink(linkPath) {
		return "", fmt.Errorf("path is not a symlink: %s", linkPath)
	}

	// 使用os.Readlink因为afero不支持
	target, err := os.Readlink(linkPath)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink target: %w", err)
	}

	return target, nil
}

// UpdateSymlink 更新符号链接目标
func (sm *DefaultSymlinkManager) UpdateSymlink(linkPath, newTarget string) error {
	sm.logger.Debugf("Updating symlink: %s -> %s", linkPath, newTarget)

	// 检查新目标是否存在
	if _, err := sm.fs.Stat(newTarget); os.IsNotExist(err) {
		return fmt.Errorf("new target does not exist: %s", newTarget)
	}

	// 如果符号链接存在，先移除
	if sm.IsSymlink(linkPath) {
		if err := sm.RemoveSymlink(linkPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// 创建新的符号链接
	return sm.CreateSymlink(newTarget, linkPath)
}

// CreateToolSymlinks 为工具创建符号链接
func (sm *DefaultSymlinkManager) CreateToolSymlinks(toolName, version, binPath, shimDir string) error {
	sm.logger.Infof("Creating symlinks for tool %s@%s", toolName, version)

	// 检查二进制文件是否存在
	if _, err := sm.fs.Stat(binPath); os.IsNotExist(err) {
		return fmt.Errorf("binary file does not exist: %s", binPath)
	}

	// 确保shim目录存在
	if err := sm.fs.MkdirAll(shimDir, 0755); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	// 创建主要的工具符号链接
	mainLinkPath := filepath.Join(shimDir, toolName)
	if err := sm.createOrUpdateSymlink(binPath, mainLinkPath); err != nil {
		return fmt.Errorf("failed to create main symlink: %w", err)
	}

	// 创建带版本号的符号链接
	versionedLinkPath := filepath.Join(shimDir, fmt.Sprintf("%s-%s", toolName, version))
	if err := sm.createOrUpdateSymlink(binPath, versionedLinkPath); err != nil {
		sm.logger.Warnf("Failed to create versioned symlink: %v", err)
		// 不返回错误，因为主要符号链接已经创建成功
	}

	// Windows上需要创建.exe版本
	if runtime.GOOS == "windows" {
		mainExeLinkPath := mainLinkPath + ".exe"
		if err := sm.createOrUpdateSymlink(binPath, mainExeLinkPath); err != nil {
			sm.logger.Warnf("Failed to create .exe symlink: %v", err)
		}

		versionedExeLinkPath := versionedLinkPath + ".exe"
		if err := sm.createOrUpdateSymlink(binPath, versionedExeLinkPath); err != nil {
			sm.logger.Warnf("Failed to create versioned .exe symlink: %v", err)
		}
	}

	sm.logger.Infof("Successfully created symlinks for %s@%s", toolName, version)
	return nil
}

// RemoveToolSymlinks 移除工具的符号链接
func (sm *DefaultSymlinkManager) RemoveToolSymlinks(toolName, shimDir string) error {
	sm.logger.Infof("Removing symlinks for tool: %s", toolName)

	// 获取所有符号链接
	symlinks, err := sm.ListSymlinks(shimDir)
	if err != nil {
		return fmt.Errorf("failed to list symlinks: %w", err)
	}

	// 移除相关的符号链接
	removedCount := 0
	for linkPath, _ := range symlinks {
		linkName := filepath.Base(linkPath)

		// 检查是否为目标工具的符号链接
		if sm.isToolSymlink(linkName, toolName) {
			if err := sm.RemoveSymlink(linkPath); err != nil {
				sm.logger.Warnf("Failed to remove symlink %s: %v", linkPath, err)
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		sm.logger.Infof("Removed %d symlinks for tool: %s", removedCount, toolName)
	} else {
		sm.logger.Debugf("No symlinks found for tool: %s", toolName)
	}

	return nil
}

// ListSymlinks 列出目录中的所有符号链接
func (sm *DefaultSymlinkManager) ListSymlinks(dir string) (map[string]string, error) {
	symlinks := make(map[string]string)

	// 检查目录是否存在
	if _, err := sm.fs.Stat(dir); os.IsNotExist(err) {
		return symlinks, nil
	}

	// 遍历目录
	entries, err := afero.ReadDir(sm.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())

		if sm.IsSymlink(entryPath) {
			target, err := sm.GetSymlinkTarget(entryPath)
			if err != nil {
				sm.logger.Warnf("Failed to read symlink target for %s: %v", entryPath, err)
				symlinks[entryPath] = "broken"
			} else {
				symlinks[entryPath] = target
			}
		}
	}

	return symlinks, nil
}

// ValidateSymlinks 验证符号链接的有效性
func (sm *DefaultSymlinkManager) ValidateSymlinks(dir string) ([]string, error) {
	var brokenLinks []string

	symlinks, err := sm.ListSymlinks(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list symlinks: %w", err)
	}

	for linkPath, target := range symlinks {
		if target == "broken" {
			brokenLinks = append(brokenLinks, linkPath)
			continue
		}

		// 检查目标是否存在
		if _, err := sm.fs.Stat(target); os.IsNotExist(err) {
			brokenLinks = append(brokenLinks, linkPath)
		}
	}

	return brokenLinks, nil
}

// CleanupBrokenSymlinks 清理损坏的符号链接
func (sm *DefaultSymlinkManager) CleanupBrokenSymlinks(dir string) error {
	sm.logger.Infof("Cleaning up broken symlinks in: %s", dir)

	brokenLinks, err := sm.ValidateSymlinks(dir)
	if err != nil {
		return fmt.Errorf("failed to validate symlinks: %w", err)
	}

	removedCount := 0
	for _, linkPath := range brokenLinks {
		if err := sm.fs.Remove(linkPath); err != nil {
			sm.logger.Warnf("Failed to remove broken symlink %s: %v", linkPath, err)
		} else {
			sm.logger.Debugf("Removed broken symlink: %s", linkPath)
			removedCount++
		}
	}

	if removedCount > 0 {
		sm.logger.Infof("Cleaned up %d broken symlinks", removedCount)
	} else {
		sm.logger.Debugf("No broken symlinks found")
	}

	return nil
}

// createOrUpdateSymlink 创建或更新符号链接（内部方法）
func (sm *DefaultSymlinkManager) createOrUpdateSymlink(target, linkPath string) error {
	// 如果符号链接已存在
	if sm.IsSymlink(linkPath) {
		currentTarget, err := sm.GetSymlinkTarget(linkPath)
		if err != nil {
			// 如果无法读取目标，移除并重新创建
			sm.logger.Warnf("Cannot read current symlink target, recreating: %s", linkPath)
			sm.fs.Remove(linkPath)
		} else if currentTarget == target {
			// 目标相同，无需更新
			sm.logger.Debugf("Symlink already points to correct target: %s", linkPath)
			return nil
		} else {
			// 目标不同，更新符号链接
			return sm.UpdateSymlink(linkPath, target)
		}
	}

	// 如果路径存在但不是符号链接
	if _, err := sm.fs.Stat(linkPath); err == nil && !sm.IsSymlink(linkPath) {
		return fmt.Errorf("path exists but is not a symlink: %s", linkPath)
	}

	// 创建新的符号链接
	return sm.CreateSymlink(target, linkPath)
}

// isToolSymlink 检查符号链接名称是否属于指定工具（内部方法）
func (sm *DefaultSymlinkManager) isToolSymlink(linkName, toolName string) bool {
	// 移除可能的.exe扩展名
	if runtime.GOOS == "windows" && strings.HasSuffix(linkName, ".exe") {
		linkName = linkName[:len(linkName)-4]
	}

	// 检查是否为主要符号链接
	if linkName == toolName {
		return true
	}

	// 检查是否为带版本号的符号链接
	prefix := toolName + "-"
	if strings.HasPrefix(linkName, prefix) {
		return true
	}

	return false
}
