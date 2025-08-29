package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vman",
	Short: "通用命令行工具版本管理器",
	Long: `vman 是一个通用的命令行工具版本管理器，类似于 asdf。
它可以管理任意二进制程序的多个版本，并支持全局和项目级的版本切换。

特性:
- 支持任意二进制工具的版本管理
- 全局和项目级版本切换
- 自动下载和安装工具
- 透明的命令代理`,
	Version: "0.1.0",
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// 这里将添加全局标志和配置
	rootCmd.PersistentFlags().StringP("config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "详细输出")
	rootCmd.PersistentFlags().Bool("no-color", false, "禁用彩色输出")
	rootCmd.PersistentFlags().Bool("no-emoji", false, "禁用emoji图标")
}
