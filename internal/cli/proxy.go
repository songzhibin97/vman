package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/proxy"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/internal/version"
)

var (
	commandProxy proxy.CommandProxy
)

// initProxy 初始化代理系统
func initProxy() error {
	if commandProxy != nil {
		return nil
	}

	// 创建配置管理器
	configManager, err := config.NewManager("")
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}
	if err := configManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize config manager: %w", err)
	}

	// 创建版本管理器
	storageManager := storage.NewManager()
	versionManager := version.NewManager(storageManager, configManager)

	// 创建代理
	commandProxy = proxy.NewCommandProxy(configManager, versionManager)

	return nil
}

// proxyCmd 代理相关命令的根命令
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "代理系统管理",
	Long: `管理vman的命令拦截和代理系统。

代理系统允许vman透明地拦截工具命令，自动选择正确的版本并执行。`,
}

// setupCmd 设置代理环境命令
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "设置代理环境",
	Long: `设置vman代理环境，包括：
- 将shims目录添加到PATH
- 安装shell钩子
- 生成工具垫片`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		fmt.Println("正在设置代理环境...")
		if err := commandProxy.SetupProxy(); err != nil {
			return fmt.Errorf("设置代理环境失败: %w", err)
		}

		fmt.Println("代理环境设置成功！")
		fmt.Println("请重新启动您的shell或运行以下命令以激活代理：")
		fmt.Println("  source ~/.bashrc   # 对于bash")
		fmt.Println("  source ~/.zshrc    # 对于zsh")
		return nil
	},
}

// cleanupCmd 清理代理环境命令
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "清理代理环境",
	Long: `清理vman代理环境，包括：
- 从PATH中移除shims目录
- 卸载shell钩子
- 清理所有垫片`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		fmt.Println("正在清理代理环境...")
		if err := commandProxy.CleanupProxy(); err != nil {
			return fmt.Errorf("清理代理环境失败: %w", err)
		}

		fmt.Println("代理环境清理完成！")
		return nil
	},
}

// statusCmd 显示代理状态命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示代理状态",
	Long:  `显示vman代理系统的当前状态信息。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		status := commandProxy.GetProxyStatus()

		fmt.Printf("代理状态: %s\n", getStatusText(status.Enabled))
		fmt.Printf("Shims目录: %s\n", status.ShimsDir)
		fmt.Printf("在PATH中: %s\n", getBoolText(status.InPath))
		fmt.Printf("垫片数量: %d\n", status.ShimCount)
		fmt.Printf("上次重刷: %s\n", formatTime(status.LastRehash))

		if len(status.ManagedTools) > 0 {
			fmt.Printf("\n管理的工具 (%d):\n", len(status.ManagedTools))
			for _, tool := range status.ManagedTools {
				fmt.Printf("  - %s\n", tool)
			}
		}

		return nil
	},
}

// rehashCmd 重新生成垫片命令
var rehashCmd = &cobra.Command{
	Use:   "rehash",
	Short: "重新生成所有垫片",
	Long: `重新生成所有已安装工具的垫片文件。

这个命令在以下情况下很有用：
- 安装了新工具
- 更改了工具版本
- 垫片文件损坏`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		fmt.Println("正在重新生成垫片...")
		if err := commandProxy.RehashShims(); err != nil {
			return fmt.Errorf("重新生成垫片失败: %w", err)
		}

		fmt.Println("垫片重新生成完成！")
		return nil
	},
}

// execCmd 执行命令
var execCmd = &cobra.Command{
	Use:   "exec <tool> [args...]",
	Short: "通过代理执行工具命令",
	Long: `通过vman代理系统执行工具命令。

这个命令会：
1. 解析当前上下文中工具的版本
2. 查找对应的可执行文件
3. 透明地转发所有参数`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		toolName := args[0]
		toolArgs := args[1:]

		// 执行命令
		if err := commandProxy.InterceptCommand(toolName, toolArgs); err != nil {
			// 检查是否是找不到工具的错误
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "not installed") {
				fmt.Fprintf(os.Stderr, "工具 '%s' 未找到或未安装\n", toolName)
				fmt.Fprintf(os.Stderr, "尝试运行以下命令安装：\n")
				fmt.Fprintf(os.Stderr, "  vman install %s <version>\n", toolName)
				os.Exit(127)
			}
			return err
		}

		return nil
	},
}

// shimCmd 垫片管理命令
var shimCmd = &cobra.Command{
	Use:   "shim",
	Short: "垫片管理",
	Long:  `管理vman的工具垫片。`,
}

// generateShimCmd 生成垫片命令
var generateShimCmd = &cobra.Command{
	Use:   "generate <tool> <version>",
	Short: "生成工具垫片",
	Long:  `为指定的工具和版本生成垫片文件。`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		toolName := args[0]
		version := args[1]

		fmt.Printf("正在为 %s@%s 生成垫片...\n", toolName, version)
		if err := commandProxy.GenerateShim(toolName, version); err != nil {
			return fmt.Errorf("生成垫片失败: %w", err)
		}

		fmt.Printf("垫片生成成功: %s\n", commandProxy.GetShimPath(toolName))
		return nil
	},
}

// removeShimCmd 移除垫片命令
var removeShimCmd = &cobra.Command{
	Use:   "remove <tool>",
	Short: "移除工具垫片",
	Long:  `移除指定工具的垫片文件。`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initProxy(); err != nil {
			return err
		}

		toolName := args[0]

		fmt.Printf("正在移除 %s 的垫片...\n", toolName)
		if err := commandProxy.RemoveShim(toolName); err != nil {
			return fmt.Errorf("移除垫片失败: %w", err)
		}

		fmt.Printf("垫片移除成功: %s\n", toolName)
		return nil
	},
}

// proxyInitCmd shell初始化命令
var proxyInitCmd = &cobra.Command{
	Use:   "shell-init [shell]",
	Short: "生成shell初始化脚本",
	Long: `生成shell初始化脚本，用于激活vman代理功能。

支持的shell类型：
- bash
- zsh  
- fish
- cmd (Windows)
- powershell (Windows)

如果不指定shell类型，会自动检测当前shell。

使用方法：
  eval "$(vman shell-init)"           # 自动检测shell
  eval "$(vman shell-init zsh)"       # 指定zsh
  source <(vman shell-init bash)      # bash语法`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var shellType string
		if len(args) > 0 {
			shellType = args[0]
		}

		// 创建shell集成器
		shellIntegrator := proxy.NewShellIntegrator()

		// 如果没有指定shell类型，自动检测
		if shellType == "" {
			shellType = shellIntegrator.DetectShell()
		}

		// 验证shell支持
		if !shellIntegrator.ValidateShellSupport(shellType) {
			return fmt.Errorf("不支持的shell类型: %s", shellType)
		}

		// 生成激活脚本
		vmanPath := os.Args[0] // 获取vman可执行文件路径
		script, err := shellIntegrator.GenerateActivationScript(shellType, vmanPath)
		if err != nil {
			return fmt.Errorf("生成激活脚本失败: %w", err)
		}

		fmt.Print(script)
		return nil
	},
}

// 辅助函数
func getStatusText(enabled bool) string {
	if enabled {
		return "启用"
	}
	return "禁用"
}

func getBoolText(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "从未"
	}
	return t.Format("2006-01-02 15:04:05")
}

func init() {
	// 添加代理相关的子命令
	proxyCmd.AddCommand(setupCmd)
	proxyCmd.AddCommand(cleanupCmd)
	proxyCmd.AddCommand(statusCmd)
	proxyCmd.AddCommand(rehashCmd)

	// 添加垫片管理子命令
	shimCmd.AddCommand(generateShimCmd)
	shimCmd.AddCommand(removeShimCmd)
	proxyCmd.AddCommand(shimCmd)

	// 将代理命令添加到根命令
	rootCmd.AddCommand(proxyCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(proxyInitCmd)

	// 设置标志
	setupCmd.Flags().Bool("force", false, "强制重新设置")
	cleanupCmd.Flags().Bool("all", false, "清理所有相关文件")
	statusCmd.Flags().BoolP("verbose", "v", false, "显示详细信息")
	rehashCmd.Flags().Bool("quiet", false, "静默模式")

	// 绑定配置
	viper.BindPFlag("proxy.force", setupCmd.Flags().Lookup("force"))
	viper.BindPFlag("proxy.cleanup_all", cleanupCmd.Flags().Lookup("all"))
	viper.BindPFlag("proxy.verbose", statusCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("proxy.quiet", rehashCmd.Flags().Lookup("quiet"))
}
