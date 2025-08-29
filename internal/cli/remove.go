package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// removeCmd 删除工具版本命令
var removeCmd = &cobra.Command{
	Use:     "remove <tool> <version>",
	Aliases: []string{"uninstall", "rm"},
	Short:   "删除工具版本",
	Long: `删除已安装的工具版本。

示例:
  vman remove kubectl 1.28.0     # 删除指定版本
  vman remove terraform 1.5.0   # 删除指定版本
  vman rm kubectl 1.28.0        # 使用别名
  vman remove kubectl --all     # 删除所有版本`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		// 获取选项
		force, _ := cmd.Flags().GetBool("force")
		all, _ := cmd.Flags().GetBool("all")

		// 创建管理器
		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		if all {
			// 删除所有版本
			return removeAllVersions(tool, force, managers)
		}

		// 删除指定版本
		if len(args) != 2 {
			return fmt.Errorf("请指定要删除的版本，或使用 --all 删除所有版本")
		}

		version := args[1]
		return removeVersion(tool, version, force, managers)
	},
}

// removeVersion 删除指定版本
func removeVersion(tool, version string, force bool, managers *managers) error {
	// 检查版本是否存在
	if !managers.version.IsVersionInstalled(tool, version) {
		return fmt.Errorf("版本 %s@%s 未安装", tool, version)
	}

	// 检查是否为当前使用的版本
	currentVersion, _ := managers.version.GetCurrentVersion(tool)
	if currentVersion == version && !force {
		fmt.Printf("⚠️  版本 %s@%s 当前正在使用\n", tool, version)
		if !confirmAction("确定要删除吗？") {
			fmt.Println("操作已取消")
			return nil
		}
	}

	// 显示删除信息
	fmt.Printf("正在删除 %s@%s...\n", tool, version)

	// 执行删除
	if err := managers.version.RemoveVersion(tool, version); err != nil {
		return fmt.Errorf("删除版本失败: %w", err)
	}

	fmt.Printf("✅ 成功删除 %s@%s\n", tool, version)

	// 如果删除的是当前版本，清除引用
	if currentVersion == version {
		// 尝试设置为其他可用版本
		versions, err := managers.version.ListVersions(tool)
		if err == nil && len(versions) > 0 {
			newVersion := versions[0]
			if err := managers.version.SetGlobalVersion(tool, newVersion); err == nil {
				fmt.Printf("已自动切换到 %s@%s\n", tool, newVersion)
			}
		} else {
			fmt.Printf("⚠️  %s 没有其他可用版本\n", tool)
		}
	}

	// 重新生成垫片
	if err := regenerateShims(); err != nil {
		fmt.Printf("警告: 重新生成垫片失败: %v\n", err)
	}

	return nil
}

// removeAllVersions 删除所有版本
func removeAllVersions(tool string, force bool, managers *managers) error {
	// 获取所有版本
	versions, err := managers.version.ListVersions(tool)
	if err != nil {
		return fmt.Errorf("获取版本列表失败: %w", err)
	}

	if len(versions) == 0 {
		fmt.Printf("工具 %s 没有已安装的版本\n", tool)
		return nil
	}

	// 显示将要删除的版本
	fmt.Printf("将要删除 %s 的以下版本:\n", tool)
	for _, version := range versions {
		fmt.Printf("  - %s\n", version)
	}

	// 确认操作
	if !force {
		if !confirmAction(fmt.Sprintf("确定要删除 %s 的所有 %d 个版本吗？", tool, len(versions))) {
			fmt.Println("操作已取消")
			return nil
		}
	}

	// 执行删除
	fmt.Printf("正在删除 %s 的所有版本...\n", tool)

	successCount := 0
	for _, version := range versions {
		if err := managers.version.RemoveVersion(tool, version); err != nil {
			fmt.Printf("❌ 删除 %s@%s 失败: %v\n", tool, version, err)
		} else {
			fmt.Printf("✅ 已删除 %s@%s\n", tool, version)
			successCount++
		}
	}

	fmt.Printf("\n删除完成: %d/%d 个版本成功删除\n", successCount, len(versions))

	// 重新生成垫片
	if err := regenerateShims(); err != nil {
		fmt.Printf("警告: 重新生成垫片失败: %v\n", err)
	}

	return nil
}

// confirmAction 确认用户操作
func confirmAction(message string) bool {
	fmt.Printf("%s [y/N]: ", message)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func init() {
	// 添加remove命令到根命令
	rootCmd.AddCommand(removeCmd)

	// 添加选项
	removeCmd.Flags().BoolP("force", "f", false, "强制删除，跳过确认提示")
	removeCmd.Flags().Bool("all", false, "删除指定工具的所有版本")
}
