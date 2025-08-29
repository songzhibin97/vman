package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/songzhibin97/vman/pkg/utils"
)

// useCmd 快速切换工具版本命令
var useCmd = &cobra.Command{
	Use:   "use <tool> <version>",
	Short: "切换工具版本",
	Long: `快速切换工具版本。支持全局切换和本地项目切换。

示例:
  vman use kubectl 1.29.0        # 在当前项目中使用kubectl 1.29.0
  vman use kubectl 1.29.0 -g     # 全局切换到kubectl 1.29.0
  vman use terraform latest      # 使用最新版本
  vman use terraform system      # 使用系统版本`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		version := args[1]

		// 获取选项
		global, _ := cmd.Flags().GetBool("global")

		// 创建管理器
		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		// 处理特殊版本
		resolvedVersion, err := resolveVersion(tool, version, managers)
		if err != nil {
			return fmt.Errorf("版本解析失败: %w", err)
		}

		// 检查版本是否已安装
		if resolvedVersion != "system" && !managers.version.IsVersionInstalled(tool, resolvedVersion) {
			return fmt.Errorf("版本 %s@%s 未安装。请先运行: vman install %s %s", tool, resolvedVersion, tool, resolvedVersion)
		}

		if global {
			// 全局切换
			if err := managers.version.SetGlobalVersion(tool, resolvedVersion); err != nil {
				return fmt.Errorf("设置全局版本失败: %w", err)
			}
			fmt.Printf("✅ 成功设置 %s@%s 为全局版本\n", tool, resolvedVersion)
		} else {
			// 本地项目切换
			if err := setLocalVersion(tool, resolvedVersion); err != nil {
				return fmt.Errorf("设置本地版本失败: %w", err)
			}
			fmt.Printf("✅ 成功设置 %s@%s 为当前项目版本\n", tool, resolvedVersion)
		}

		// 重新生成垫片
		if err := regenerateShims(); err != nil {
			fmt.Printf("警告: 重新生成垫片失败: %v\n", err)
		}

		return nil
	},
}

// resolveVersion 解析版本号（处理latest、system等特殊版本）
func resolveVersion(tool, version string, managers *managers) (string, error) {
	switch version {
	case "latest":
		// 获取最新版本
		versions, err := managers.version.ListVersions(tool)
		if err != nil {
			return "", fmt.Errorf("获取版本列表失败: %w", err)
		}
		if len(versions) == 0 {
			return "", fmt.Errorf("工具 %s 没有已安装的版本", tool)
		}
		// 返回第一个版本（通常是最新的）
		return versions[0], nil

	case "system":
		// 使用系统版本
		return "system", nil

	default:
		// 直接返回指定版本
		return version, nil
	}
}

// setLocalVersion 设置本地项目版本
func setLocalVersion(tool, version string) error {
	// 查找项目根目录
	projectRoot, err := findProjectRoot()
	if err != nil {
		// 如果找不到项目根目录，就在当前目录创建
		projectRoot, _ = os.Getwd()
	}

	// 读取现有的 .vman-version 文件
	versionFile := filepath.Join(projectRoot, ".vman-version")
	versions := make(map[string]string)

	if utils.FileExists(versionFile) {
		content, err := os.ReadFile(versionFile)
		if err != nil {
			return fmt.Errorf("读取版本文件失败: %w", err)
		}

		// 解析现有版本
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) == 2 {
				versions[parts[0]] = parts[1]
			}
		}
	}

	// 更新版本
	versions[tool] = version

	// 写入文件
	var content strings.Builder
	content.WriteString("# vman版本配置文件\n")
	content.WriteString("# 格式: <工具名> <版本>\n\n")

	for t, v := range versions {
		content.WriteString(fmt.Sprintf("%s %s\n", t, v))
	}

	if err := os.WriteFile(versionFile, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("写入版本文件失败: %w", err)
	}

	return nil
}

// findProjectRoot 查找项目根目录
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 向上查找项目标识文件
	projectFiles := []string{
		".git",
		".vman-version",
		".tool-versions",
		"package.json",
		"go.mod",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"requirements.txt",
		"Pipfile",
	}

	dir := currentDir
	for {
		for _, file := range projectFiles {
			if utils.FileExists(filepath.Join(dir, file)) {
				return dir, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 到达根目录
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到项目根目录")
}

// regenerateShims 重新生成垫片
func regenerateShims() error {
	// 初始化代理
	if err := initProxy(); err != nil {
		return err
	}

	// 重新生成垫片
	return commandProxy.RehashShims()
}

func init() {
	// 添加use命令到根命令
	rootCmd.AddCommand(useCmd)

	// 添加选项
	useCmd.Flags().BoolP("global", "g", false, "设置为全局版本（而非项目本地版本）")
}
