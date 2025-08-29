package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/songzhibin97/vman/internal/config"
	"github.com/songzhibin97/vman/internal/storage"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
	"github.com/songzhibin97/vman/pkg/utils"
)

// 注册版本管理相关的命令
func init() {
	// 注册版本命令
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(globalCmd)
	rootCmd.AddCommand(localCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(whichCmd)
}

var registerCmd = &cobra.Command{
	Use:   "register <tool> <version> <binary_path>",
	Short: "手动注册工具版本",
	Long: `手动注册一个工具版本，将指定的二进制文件注册到vman中。

示例:
  vman register kubectl 1.29.0 /usr/local/bin/kubectl
  vman register terraform 1.6.0 ./terraform`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		versionStr := args[1]
		binaryPath := args[2]

		// 检查二进制文件是否存在
		if !utils.FileExists(binaryPath) {
			return fmt.Errorf("binary file not found: %s", binaryPath)
		}

		// 检查是否为可执行文件
		if !utils.IsExecutable(binaryPath) {
			return fmt.Errorf("file is not executable: %s", binaryPath)
		}

		// 创建管理器
		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		// 注册版本
		if err := managers.version.RegisterVersion(tool, versionStr, binaryPath); err != nil {
			return fmt.Errorf("failed to register version: %w", err)
		}

		fmt.Printf("Successfully registered %s@%s\n", tool, versionStr)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list [tool]",
	Short: "列出工具版本",
	Long: `列出已安装的工具版本。如果指定了工具名，则列出该工具的所有版本；否则列出所有工具。

示例:
  vman list              # 列出所有工具
  vman list kubectl      # 列出kubectl的所有版本`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		if len(args) == 1 {
			// 列出指定工具的版本
			tool := args[0]
			versions, err := managers.version.ListVersions(tool)
			if err != nil {
				return fmt.Errorf("failed to list versions for %s: %w", tool, err)
			}

			if len(versions) == 0 {
				fmt.Printf("No versions installed for %s\n", tool)
				return nil
			}

			// 获取当前版本
			currentVersion, _ := managers.version.GetCurrentVersion(tool)

			fmt.Printf("Installed versions for %s:\n", tool)
			for _, v := range versions {
				marker := "  "
				if v == currentVersion {
					marker = "* "
				}
				fmt.Printf("%s%s\n", marker, v)
			}
		} else {
			// 列出所有工具
			tools, err := managers.version.ListAllTools()
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}

			if len(tools) == 0 {
				fmt.Println("No tools installed")
				return nil
			}

			fmt.Println("Installed tools:")
			for _, tool := range tools {
				versions, err := managers.version.ListVersions(tool)
				if err != nil {
					fmt.Printf("  %s: <error getting versions>\n", tool)
					continue
				}

				currentVersion, _ := managers.version.GetCurrentVersion(tool)
				versionStr := strings.Join(versions, ", ")
				if currentVersion != "" {
					fmt.Printf("  %s: %s (current: %s)\n", tool, versionStr, currentVersion)
				} else {
					fmt.Printf("  %s: %s\n", tool, versionStr)
				}
			}
		}

		return nil
	},
}

var currentCmd = &cobra.Command{
	Use:   "current [tool]",
	Short: "显示当前使用的版本",
	Long: `显示当前使用的工具版本。如果指定了工具名，则显示该工具的当前版本；否则显示所有工具的当前版本。

示例:
  vman current           # 显示所有工具的当前版本
  vman current kubectl   # 显示kubectl的当前版本`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		if len(args) == 1 {
			// 显示指定工具的当前版本
			tool := args[0]
			version, err := managers.version.GetCurrentVersion(tool)
			if err != nil {
				return fmt.Errorf("failed to get current version for %s: %w", tool, err)
			}

			fmt.Printf("%s: %s\n", tool, version)
		} else {
			// 显示所有工具的当前版本
			tools, err := managers.version.ListAllTools()
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}

			if len(tools) == 0 {
				fmt.Println("No tools installed")
				return nil
			}

			fmt.Println("Current versions:")
			for _, tool := range tools {
				version, err := managers.version.GetCurrentVersion(tool)
				if err != nil {
					fmt.Printf("  %s: <not set>\n", tool)
				} else {
					fmt.Printf("  %s: %s\n", tool, version)
				}
			}
		}

		return nil
	},
}

var globalCmd = &cobra.Command{
	Use:   "global <tool> <version>",
	Short: "设置工具的全局版本",
	Long: `设置工具的全局默认版本。

示例:
  vman global kubectl 1.29.0
  vman global terraform 1.6.0`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		versionStr := args[1]

		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		if err := managers.version.SetGlobalVersion(tool, versionStr); err != nil {
			return fmt.Errorf("failed to set global version: %w", err)
		}

		fmt.Printf("Set global version for %s to %s\n", tool, versionStr)
		return nil
	},
}

var localCmd = &cobra.Command{
	Use:   "local <tool> <version>",
	Short: "设置工具的项目级版本",
	Long: `在当前目录设置工具的项目级版本。项目级版本优先于全局版本。

示例:
  vman local kubectl 1.28.0
  vman local terraform 1.5.0`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		versionStr := args[1]

		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		if err := managers.version.SetLocalVersion(tool, versionStr); err != nil {
			return fmt.Errorf("failed to set local version: %w", err)
		}

		cwd, _ := os.Getwd()
		fmt.Printf("Set local version for %s to %s in %s\n", tool, versionStr, cwd)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <tool> <version>",
	Short: "卸载工具版本",
	Long: `卸载指定的工具版本。

示例:
  vman uninstall kubectl 1.28.0
  vman uninstall terraform 1.5.0`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		versionStr := args[1]

		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		if err := managers.version.RemoveVersion(tool, versionStr); err != nil {
			return fmt.Errorf("failed to uninstall version: %w", err)
		}

		fmt.Printf("Successfully uninstalled %s@%s\n", tool, versionStr)
		return nil
	},
}

var whichCmd = &cobra.Command{
	Use:   "which <tool>",
	Short: "显示工具的当前二进制文件路径",
	Long: `显示工具当前版本的二进制文件路径。

示例:
  vman which kubectl
  vman which terraform`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		managers, err := createManagers()
		if err != nil {
			return fmt.Errorf("failed to create managers: %w", err)
		}

		// 获取当前版本
		version, err := managers.version.GetCurrentVersion(tool)
		if err != nil {
			return fmt.Errorf("failed to get current version: %w", err)
		}

		// 获取二进制文件路径
		binaryPath := managers.storage.GetBinaryPath(tool, version)
		if !utils.FileExists(binaryPath) {
			return fmt.Errorf("binary file not found: %s", binaryPath)
		}

		fmt.Println(binaryPath)
		return nil
	},
}

// managers 结构体用于管理各种管理器
type managers struct {
	version version.Manager
	config  config.Manager
	storage storage.Manager
}

// createManagers 创建管理器实例
func createManagers() (*managers, error) {
	// 获取配置目录
	homeDir, err := utils.GetHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPaths := types.DefaultConfigPaths(homeDir)

	// 创建配置管理器
	configManager, err := config.NewManager(homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	// 创建存储管理器
	storageManager := storage.NewFilesystemManager(configPaths)

	// 创建版本管理器
	versionManager := version.NewManager(storageManager, configManager)

	// 确保目录存在
	if err := storageManager.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to ensure directories: %w", err)
	}

	// 初始化配置
	if err := configManager.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	return &managers{
		version: versionManager,
		config:  configManager,
		storage: storageManager,
	}, nil
}
