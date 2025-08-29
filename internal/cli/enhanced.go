package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// enhancedCurrentCmd 增强版当前版本命令
var enhancedCurrentCmd = &cobra.Command{
	Use:   "current [tool]",
	Short: "显示当前使用的版本（增强版）",
	Long: `显示当前使用的工具版本。支持多种输出格式和详细信息。

示例:
  vman current                 # 显示所有工具的当前版本
  vman current kubectl         # 显示kubectl的当前版本
  vman current --paths         # 显示版本对应的可执行文件路径
  vman current --json          # JSON格式输出`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEnhancedCurrentCommand,
}

func runEnhancedCurrentCommand(cmd *cobra.Command, args []string) error {
	// 获取选项
	showPaths, _ := cmd.Flags().GetBool("paths")
	jsonFormat, _ := cmd.Flags().GetBool("json")
	detailed, _ := cmd.Flags().GetBool("detailed")

	managers, err := createManagers()
	if err != nil {
		return fmt.Errorf("创建管理器失败: %w", err)
	}

	if len(args) == 1 {
		// 显示指定工具的当前版本
		tool := args[0]
		return showCurrentVersionEnhanced(managers, tool, showPaths, jsonFormat, detailed)
	} else {
		// 显示所有工具的当前版本
		return showAllCurrentVersionsEnhanced(managers, showPaths, jsonFormat, detailed)
	}
}

// showCurrentVersionEnhanced 显示指定工具的当前版本
func showCurrentVersionEnhanced(managers *managers, tool string, showPaths, jsonFormat, detailed bool) error {
	currentVersion, err := managers.version.GetCurrentVersion(tool)
	if err != nil {
		return fmt.Errorf("获取 %s 当前版本失败: %w", tool, err)
	}

	if jsonFormat {
		data := map[string]interface{}{
			"tool":    tool,
			"version": currentVersion,
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON编码失败: %w", err)
		}

		fmt.Println(string(jsonData))
		return nil
	}

	// 默认格式输出
	if currentVersion == "" {
		fmt.Printf("🔧 %s: 未设置当前版本\n", tool)

		// 提示可用版本
		if versions, err := managers.version.ListVersions(tool); err == nil && len(versions) > 0 {
			fmt.Printf("   💡 可用版本: %s\n", strings.Join(versions, ", "))
			fmt.Printf("   💡 使用命令设置: vman use %s <version>\n", tool)
		}
		return nil
	}

	fmt.Printf("🔧 %s: %s", tool, currentVersion)

	if detailed {
		fmt.Printf("\n   📦 大小: %s", "未知")
		fmt.Printf(" | 🕒 安装时间: %s", "未知")
	}

	fmt.Println()
	return nil
}

// showAllCurrentVersionsEnhanced 显示所有工具的当前版本
func showAllCurrentVersionsEnhanced(managers *managers, showPaths, jsonFormat, detailed bool) error {
	tools, err := managers.version.ListAllTools()
	if err != nil {
		return fmt.Errorf("获取工具列表失败: %w", err)
	}

	if len(tools) == 0 {
		fmt.Println("未安装任何工具")
		return nil
	}

	if jsonFormat {
		allVersions := make(map[string]interface{})

		for _, tool := range tools {
			currentVersion, _ := managers.version.GetCurrentVersion(tool)

			toolData := map[string]interface{}{
				"version": currentVersion,
			}

			allVersions[tool] = toolData
		}

		data := map[string]interface{}{
			"tools": allVersions,
			"total": len(tools),
		}

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("JSON编码失败: %w", err)
		}

		fmt.Println(string(jsonData))
		return nil
	}

	// 默认格式输出
	fmt.Printf("🛠️  当前版本 (%d 个工具):\n\n", len(tools))

	for i, tool := range tools {
		currentVersion, _ := managers.version.GetCurrentVersion(tool)

		fmt.Printf("  %d. 🔧 %s: %s", i+1, tool, getVersionOrNoneEnhanced(currentVersion))
		fmt.Println()
	}

	return nil
}

// 辅助函数
func getVersionOrNoneEnhanced(version string) string {
	if version == "" {
		return "<未设置>"
	}
	return version
}

func formatBytesEnhanced(bytes int64) string {
	if bytes <= 0 {
		return "未知"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 注册增强版命令
func init() {
	// 为enhancedCurrentCmd添加标志
	enhancedCurrentCmd.Flags().Bool("paths", false, "显示可执行文件路径")
	enhancedCurrentCmd.Flags().Bool("json", false, "使用JSON格式输出")
	enhancedCurrentCmd.Flags().BoolP("detailed", "d", false, "显示详细信息")
}
