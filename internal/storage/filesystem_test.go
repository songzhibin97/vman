package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/songzhibin97/vman/pkg/types"
)

func TestFilesystemManager(t *testing.T) {
	// 创建内存文件系统用于测试
	fs := afero.NewMemMapFs()
	homeDir := "/home/test"
	configPaths := types.DefaultConfigPaths(homeDir)

	manager := NewFilesystemManagerWithFs(fs, configPaths)

	t.Run("GetDirectories", func(t *testing.T) {
		// 测试获取各种目录路径
		assert.Equal(t, configPaths.ToolsDir, manager.GetToolsDir())
		assert.Equal(t, configPaths.ConfigDir, manager.GetConfigDir())
		assert.Equal(t, configPaths.ShimsDir, manager.GetShimsDir())
		assert.Equal(t, configPaths.CacheDir, manager.GetCacheDir())
		assert.Equal(t, configPaths.VersionsDir, manager.GetVersionsDir())
		assert.Equal(t, configPaths.BinDir, manager.GetBinDir())
		assert.Equal(t, configPaths.LogsDir, manager.GetLogsDir())
		assert.Equal(t, configPaths.TempDir, manager.GetTempDir())

		expectedSourcesDir := filepath.Join(configPaths.CacheDir, "sources")
		assert.Equal(t, expectedSourcesDir, manager.GetSourcesDir())
	})

	t.Run("EnsureDirectories", func(t *testing.T) {
		// 确保目录被创建
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		// 验证所有目录都存在
		dirs := []string{
			configPaths.ConfigDir,
			configPaths.ToolsDir,
			configPaths.BinDir,
			configPaths.ShimsDir,
			configPaths.VersionsDir,
			configPaths.LogsDir,
			configPaths.CacheDir,
			configPaths.TempDir,
			manager.GetSourcesDir(),
		}

		for _, dir := range dirs {
			exists, err := afero.DirExists(fs, dir)
			require.NoError(t, err)
			assert.True(t, exists, "Directory should exist: %s", dir)
		}
	})

	t.Run("VersionPaths", func(t *testing.T) {
		tool := "kubectl"
		version := "1.29.0"

		// 获取版本路径
		versionPath := manager.GetToolVersionPath(tool, version)
		expectedVersionPath := filepath.Join(configPaths.VersionsDir, tool, version)
		assert.Equal(t, expectedVersionPath, versionPath)

		// 获取二进制文件路径
		binaryPath := manager.GetBinaryPath(tool, version)
		expectedBinaryPath := filepath.Join(expectedVersionPath, "bin", tool)
		assert.Equal(t, expectedBinaryPath, binaryPath)

		// 获取元数据文件路径
		metadataPath := manager.GetVersionMetadataPath(tool, version)
		expectedMetadataPath := filepath.Join(expectedVersionPath, "metadata.json")
		assert.Equal(t, expectedMetadataPath, metadataPath)
	})

	t.Run("CreateAndRemoveVersionDir", func(t *testing.T) {
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "terraform"
		version := "1.6.0"

		// 创建版本目录
		err = manager.CreateVersionDir(tool, version)
		require.NoError(t, err)

		// 验证目录存在
		versionPath := manager.GetToolVersionPath(tool, version)
		binPath := filepath.Join(versionPath, "bin")

		exists, err := afero.DirExists(fs, versionPath)
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = afero.DirExists(fs, binPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// 删除版本目录
		err = manager.RemoveVersionDir(tool, version)
		require.NoError(t, err)

		// 验证目录已被删除
		exists, err = afero.DirExists(fs, versionPath)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("IsVersionInstalled", func(t *testing.T) {
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "sqlc"
		version := "1.25.0"

		// 初始状态应该未安装
		assert.False(t, manager.IsVersionInstalled(tool, version))

		// 创建版本目录和二进制文件
		err = manager.CreateVersionDir(tool, version)
		require.NoError(t, err)

		binaryPath := manager.GetBinaryPath(tool, version)
		err = afero.WriteFile(fs, binaryPath, []byte("test binary"), 0755)
		require.NoError(t, err)

		// 现在应该显示已安装
		assert.True(t, manager.IsVersionInstalled(tool, version))
	})

	t.Run("GetToolVersions", func(t *testing.T) {
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "multi-version"
		versions := []string{"1.0.0", "1.1.0", "2.0.0"}

		// 创建多个版本
		for _, v := range versions {
			err = manager.CreateVersionDir(tool, v)
			require.NoError(t, err)

			binaryPath := manager.GetBinaryPath(tool, v)
			err = afero.WriteFile(fs, binaryPath, []byte("test binary"), 0755)
			require.NoError(t, err)
		}

		// 获取版本列表
		installedVersions, err := manager.GetToolVersions(tool)
		require.NoError(t, err)

		// 验证所有版本都被列出
		assert.Len(t, installedVersions, len(versions))
		for _, v := range versions {
			assert.Contains(t, installedVersions, v)
		}

		// 测试不存在的工具
		emptyVersions, err := manager.GetToolVersions("nonexistent-tool")
		require.NoError(t, err)
		assert.Empty(t, emptyVersions)
	})

	t.Run("SaveAndLoadVersionMetadata", func(t *testing.T) {
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "metadata-test"
		version := "1.0.0"

		// 创建版本目录
		err = manager.CreateVersionDir(tool, version)
		require.NoError(t, err)

		// 创建测试元数据
		metadata := &types.VersionMetadata{
			Version:     version,
			ToolName:    tool,
			InstallPath: manager.GetToolVersionPath(tool, version),
			BinaryPath:  manager.GetBinaryPath(tool, version),
			InstalledAt: time.Now(),
			InstallType: "manual",
			Size:        1024,
			Checksum:    "abc123",
			Source:      "/tmp/test-binary",
		}

		// 保存元数据
		err = manager.SaveVersionMetadata(tool, version, metadata)
		require.NoError(t, err)

		// 验证文件存在
		metadataPath := manager.GetVersionMetadataPath(tool, version)
		exists, err := afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// 加载元数据
		loadedMetadata, err := manager.LoadVersionMetadata(tool, version)
		require.NoError(t, err)

		// 验证数据一致
		assert.Equal(t, metadata.Version, loadedMetadata.Version)
		assert.Equal(t, metadata.ToolName, loadedMetadata.ToolName)
		assert.Equal(t, metadata.InstallPath, loadedMetadata.InstallPath)
		assert.Equal(t, metadata.BinaryPath, loadedMetadata.BinaryPath)
		assert.Equal(t, metadata.InstallType, loadedMetadata.InstallType)
		assert.Equal(t, metadata.Size, loadedMetadata.Size)
		assert.Equal(t, metadata.Checksum, loadedMetadata.Checksum)
		assert.Equal(t, metadata.Source, loadedMetadata.Source)

		// 测试加载不存在的元数据
		_, err = manager.LoadVersionMetadata(tool, "nonexistent")
		assert.Error(t, err)
	})

	t.Run("CleanupOrphaned", func(t *testing.T) {
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "cleanup-test"

		// 创建一个正常的版本（有二进制文件）
		validVersion := "1.0.0"
		err = manager.CreateVersionDir(tool, validVersion)
		require.NoError(t, err)

		binaryPath := manager.GetBinaryPath(tool, validVersion)
		err = afero.WriteFile(fs, binaryPath, []byte("test binary"), 0755)
		require.NoError(t, err)

		// 创建一个孤立的版本目录（没有二进制文件）
		orphanedVersion := "2.0.0"
		err = manager.CreateVersionDir(tool, orphanedVersion)
		require.NoError(t, err)
		// 注意：不创建二进制文件，这样就成了孤立目录

		// 在临时目录中创建一些旧文件
		tempDir := manager.GetTempDir()
		oldFile := filepath.Join(tempDir, "old-temp-file")
		err = afero.WriteFile(fs, oldFile, []byte("old temp"), 0644)
		require.NoError(t, err)

		// 修改文件时间为25小时前（超过24小时清理阈值）
		oldTime := time.Now().Add(-25 * time.Hour)
		err = fs.Chtimes(oldFile, oldTime, oldTime)
		require.NoError(t, err)

		// 执行清理
		err = manager.CleanupOrphaned()
		require.NoError(t, err)

		// 验证正常版本仍然存在
		assert.True(t, manager.IsVersionInstalled(tool, validVersion))

		// 验证孤立版本被清理
		orphanedPath := manager.GetToolVersionPath(tool, orphanedVersion)
		exists, err := afero.DirExists(fs, orphanedPath)
		require.NoError(t, err)
		assert.False(t, exists)

		// 验证旧临时文件被清理
		exists, err = afero.Exists(fs, oldFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestFilesystemManagerIntegration(t *testing.T) {
	// 创建临时目录进行真实文件系统测试
	tmpDir, err := os.MkdirTemp("", "vman-storage-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPaths := types.DefaultConfigPaths(tmpDir)
	manager := NewFilesystemManager(configPaths)

	t.Run("RealFilesystem", func(t *testing.T) {
		// 确保目录存在
		err := manager.EnsureDirectories()
		require.NoError(t, err)

		tool := "real-tool"
		version := "1.0.0"

		// 创建版本目录
		err = manager.CreateVersionDir(tool, version)
		require.NoError(t, err)

		// 创建实际的二进制文件
		binaryPath := manager.GetBinaryPath(tool, version)
		content := []byte("#!/bin/bash\necho 'hello world'")
		err = os.WriteFile(binaryPath, content, 0755)
		require.NoError(t, err)

		// 验证版本已安装
		assert.True(t, manager.IsVersionInstalled(tool, version))

		// 创建并保存元数据
		metadata := &types.VersionMetadata{
			Version:     version,
			ToolName:    tool,
			InstallPath: manager.GetToolVersionPath(tool, version),
			BinaryPath:  binaryPath,
			InstalledAt: time.Now(),
			InstallType: "manual",
			Size:        int64(len(content)),
			Source:      "/tmp/test-source",
		}

		err = manager.SaveVersionMetadata(tool, version, metadata)
		require.NoError(t, err)

		// 加载并验证元数据
		loadedMetadata, err := manager.LoadVersionMetadata(tool, version)
		require.NoError(t, err)
		assert.Equal(t, metadata.Version, loadedMetadata.Version)
		assert.Equal(t, metadata.ToolName, loadedMetadata.ToolName)

		// 获取版本列表
		versions, err := manager.GetToolVersions(tool)
		require.NoError(t, err)
		assert.Contains(t, versions, version)

		// 清理测试
		err = manager.RemoveVersionDir(tool, version)
		require.NoError(t, err)
		assert.False(t, manager.IsVersionInstalled(tool, version))
	})
}
