package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/songzhibin97/vman/internal/download"
	"github.com/songzhibin97/vman/internal/version"
	"github.com/songzhibin97/vman/pkg/types"
)

// 注册下载相关的命令
func init() {
	// 注册下载命令
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(addSourceCmd)
	rootCmd.AddCommand(listSourcesCmd)
	rootCmd.AddCommand(removeSourceCmd)
}

var installCmd = &cobra.Command{
	Use:   "install <tool> [version]",
	Short: "安装工具版本",
	Long: `自动下载并安装指定工具的版本。如果不指定版本，则安装最新版本。

示例:
  vman install kubectl 1.29.0    # 安装指定版本
  vman install kubectl           # 安装最新版本
  vman install terraform         # 安装最新版本`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		var versionStr string

		// 获取选项
		force, _ := cmd.Flags().GetBool("force")
		global, _ := cmd.Flags().GetBool("global")

		// 创建集成管理器
		integratedManager, err := createIntegratedManager()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		// 确定版本
		if len(args) == 2 {
			versionStr = args[1]
		} else {
			// 安装最新版本
			fmt.Printf("正在获取 %s 的最新版本...\n", tool)
			latestVersion, err := integratedManager.InstallLatestVersion(tool)
			if err != nil {
				return fmt.Errorf("安装最新版本失败: %w", err)
			}
			versionStr = latestVersion
			fmt.Printf("成功安装最新版本: %s@%s\n", tool, versionStr)

			// 设置为全局版本（如果指定）
			if global {
				if err := integratedManager.SetGlobalVersion(tool, versionStr); err != nil {
					fmt.Printf("警告: 设置全局版本失败: %v\n", err)
				} else {
					fmt.Printf("设置 %s@%s 为全局版本\n", tool, versionStr)
				}
			}
			return nil
		}

		// 检查版本是否已安装
		if !force && integratedManager.IsVersionInstalled(tool, versionStr) {
			fmt.Printf("版本 %s@%s 已安装\n", tool, versionStr)

			// 设置为全局版本（如果指定）
			if global {
				if err := integratedManager.SetGlobalVersion(tool, versionStr); err != nil {
					fmt.Printf("警告: 设置全局版本失败: %v\n", err)
				} else {
					fmt.Printf("设置 %s@%s 为全局版本\n", tool, versionStr)
				}
			}
			return nil
		}

		// 安装版本（带进度）
		fmt.Printf("正在安装 %s@%s...\n", tool, versionStr)

		// 进度回调
		progressCallback := func(info *types.ProgressInfo) {
			if info.Total > 0 {
				fmt.Printf("\r下载进度: %.1f%% (%s) - %s",
					info.Percentage,
					formatBytes(info.Downloaded),
					info.Status)
			} else {
				fmt.Printf("\r%s", info.Status)
			}
		}

		if err := integratedManager.InstallVersionWithProgress(tool, versionStr, progressCallback); err != nil {
			fmt.Println() // 换行
			return fmt.Errorf("安装失败: %w", err)
		}

		fmt.Printf("\n成功安装 %s@%s\n", tool, versionStr)

		// 设置为全局版本（如果指定）
		if global {
			if err := integratedManager.SetGlobalVersion(tool, versionStr); err != nil {
				fmt.Printf("警告: 设置全局版本失败: %v\n", err)
			} else {
				fmt.Printf("设置 %s@%s 为全局版本\n", tool, versionStr)
			}
		}

		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update <tool>",
	Short: "更新工具到最新版本",
	Long: `更新指定工具到最新版本。

示例:
  vman update kubectl
  vman update terraform`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		// 创建集成管理器
		integratedManager, err := createIntegratedManager()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		fmt.Printf("正在更新 %s...\n", tool)

		newVersion, err := integratedManager.UpdateTool(tool)
		if err != nil {
			return fmt.Errorf("更新失败: %w", err)
		}

		fmt.Printf("成功更新到版本: %s\n", newVersion)
		return nil
	},
}

var searchCmd = &cobra.Command{
	Use:   "search <tool>",
	Short: "搜索可用的工具版本",
	Long: `搜索指定工具的所有可用版本。

示例:
  vman search kubectl
  vman search terraform`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		// 获取选项
		limit, _ := cmd.Flags().GetInt("limit")
		prerelease, _ := cmd.Flags().GetBool("prerelease")

		// 创建集成管理器
		integratedManager, err := createIntegratedManager()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		fmt.Printf("正在搜索 %s 的可用版本...\n", tool)

		versions, err := integratedManager.SearchAvailableVersions(tool)
		if err != nil {
			return fmt.Errorf("搜索失败: %w", err)
		}

		if len(versions) == 0 {
			fmt.Printf("未找到 %s 的可用版本\n", tool)
			return nil
		}

		fmt.Printf("找到 %d 个可用版本:\n", len(versions))

		count := 0
		for _, version := range versions {
			// 跳过预发布版本（除非明确指定）
			if version.IsPrerelease && !prerelease {
				continue
			}

			// 检查是否已安装
			installed := integratedManager.IsVersionInstalled(tool, version.Version)
			marker := "  "
			if installed {
				marker = "* "
			}

			status := ""
			if version.IsPrerelease {
				status = " (prerelease)"
			}
			if version.IsStable {
				status += " (stable)"
			}

			fmt.Printf("%s%s%s", marker, version.Version, status)

			if version.ReleaseDate != "" {
				if releaseTime, err := time.Parse(time.RFC3339, version.ReleaseDate); err == nil {
					fmt.Printf(" - %s", releaseTime.Format("2006-01-02"))
				}
			}

			fmt.Println()

			count++
			if limit > 0 && count >= limit {
				break
			}
		}

		if !prerelease {
			prereleaseCount := 0
			for _, version := range versions {
				if version.IsPrerelease {
					prereleaseCount++
				}
			}
			if prereleaseCount > 0 {
				fmt.Printf("\n提示: 使用 --prerelease 查看 %d 个预发布版本\n", prereleaseCount)
			}
		}

		return nil
	},
}

var addSourceCmd = &cobra.Command{
	Use:   "add-source <tool>",
	Short: "添加工具的下载源配置",
	Long: `为工具添加下载源配置。支持GitHub、直接URL等多种类型。

示例:
  # GitHub源
  vman add-source kubectl --type github --repo kubernetes/kubernetes --pattern "kubernetes-client-{os}-{arch}.tar.gz"
  
  # 直接URL源  
  vman add-source terraform --type direct --url "https://releases.hashicorp.com/terraform/{version}/terraform_{version}_{os}_{arch}.zip"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		// 获取选项
		sourceType, _ := cmd.Flags().GetString("type")
		repo, _ := cmd.Flags().GetString("repo")
		pattern, _ := cmd.Flags().GetString("pattern")
		urlTemplate, _ := cmd.Flags().GetString("url")
		description, _ := cmd.Flags().GetString("description")

		if sourceType == "" {
			return fmt.Errorf("必须指定 --type")
		}

		// 创建工具元数据
		metadata := &types.ToolMetadata{
			Name:        tool,
			Description: description,
			DownloadConfig: types.DownloadConfig{
				Type: sourceType,
			},
		}

		// 根据类型设置配置
		switch sourceType {
		case "github":
			if repo == "" {
				return fmt.Errorf("GitHub源必须指定 --repo")
			}
			metadata.DownloadConfig.Repository = repo
			if pattern != "" {
				metadata.DownloadConfig.AssetPattern = pattern
			}
		case "direct", "archive":
			if urlTemplate == "" {
				return fmt.Errorf("直接URL源必须指定 --url")
			}
			metadata.DownloadConfig.URLTemplate = urlTemplate
		default:
			return fmt.Errorf("不支持的源类型: %s", sourceType)
		}

		// 创建集成管理器
		integratedManager, err := createIntegratedManager()
		if err != nil {
			return fmt.Errorf("创建管理器失败: %w", err)
		}

		// 添加下载源
		if integManager, ok := integratedManager.(*version.IntegratedManager); ok {
			if err := integManager.AddDownloadSource(tool, metadata); err != nil {
				return fmt.Errorf("添加下载源失败: %w", err)
			}
		} else {
			return fmt.Errorf("当前管理器不支持添加下载源功能")
		}

		fmt.Printf("成功为 %s 添加 %s 类型的下载源\n", tool, sourceType)
		return nil
	},
}

var listSourcesCmd = &cobra.Command{
	Use:   "list-sources",
	Short: "列出所有下载源",
	Long: `列出所有已配置的工具下载源。

示例:
  vman list-sources`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 创建下载管理器
		downloadManager, err := createDownloadManager()
		if err != nil {
			return fmt.Errorf("创建下载管理器失败: %w", err)
		}

		sources, err := downloadManager.ListSources()
		if err != nil {
			return fmt.Errorf("获取下载源列表失败: %w", err)
		}

		if len(sources) == 0 {
			fmt.Println("未配置任何下载源")
			return nil
		}

		fmt.Println("已配置的下载源:")
		for _, source := range sources {
			fmt.Printf("  %s\n", source)
		}

		return nil
	},
}

var removeSourceCmd = &cobra.Command{
	Use:   "remove-source <tool>",
	Short: "移除工具的下载源配置",
	Long: `移除指定工具的下载源配置。

示例:
  vman remove-source kubectl
  vman remove-source terraform`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]

		// 创建下载管理器
		downloadManager, err := createDownloadManager()
		if err != nil {
			return fmt.Errorf("创建下载管理器失败: %w", err)
		}

		if err := downloadManager.RemoveSource(tool); err != nil {
			return fmt.Errorf("移除下载源失败: %w", err)
		}

		fmt.Printf("成功移除 %s 的下载源配置\n", tool)
		return nil
	},
}

// createIntegratedManager 创建集成管理器
func createIntegratedManager() (version.Manager, error) {
	// 创建基础管理器
	managers, err := createManagers()
	if err != nil {
		return nil, err
	}

	// 创建下载管理器
	downloadManager, err := createDownloadManager()
	if err != nil {
		return nil, err
	}

	// 创建适配器
	adapter := &DownloadManagerAdapter{
		Manager: downloadManager,
	}

	// 创建集成管理器
	integratedManager := version.NewIntegratedManager(
		managers.storage,
		managers.config,
		adapter,
	)

	return integratedManager, nil
}

// DownloadManagerAdapter 下载管理器适配器
type DownloadManagerAdapter struct {
	Manager download.Manager
}

func (a *DownloadManagerAdapter) Download(ctx context.Context, tool, version string, options *version.DownloadOptions) error {
	// 转换选项类型
	downloadOpts := &download.DownloadOptions{}
	if options != nil {
		downloadOpts.Force = options.Force
		downloadOpts.SkipChecksum = options.SkipChecksum
		downloadOpts.Timeout = options.Timeout
		downloadOpts.Retries = options.Retries
		downloadOpts.Resume = options.Resume
		downloadOpts.TempDir = options.TempDir
		downloadOpts.KeepDownload = options.KeepDownload
		downloadOpts.Headers = options.Headers
	}
	return a.Manager.Download(ctx, tool, version, downloadOpts)
}

func (a *DownloadManagerAdapter) DownloadWithProgress(ctx context.Context, tool, version string, options *version.DownloadOptions, progress version.ProgressCallback) error {
	// 转换选项类型
	downloadOpts := &download.DownloadOptions{}
	if options != nil {
		downloadOpts.Force = options.Force
		downloadOpts.SkipChecksum = options.SkipChecksum
		downloadOpts.Timeout = options.Timeout
		downloadOpts.Retries = options.Retries
		downloadOpts.Resume = options.Resume
		downloadOpts.TempDir = options.TempDir
		downloadOpts.KeepDownload = options.KeepDownload
		downloadOpts.Headers = options.Headers
	}

	// 转换进度回调
	var progressAdapter download.ProgressCallback
	if progress != nil {
		progressAdapter = func(info *download.ProgressInfo) {
			// 转换为types.ProgressInfo
			typesInfo := &types.ProgressInfo{
				Total:      info.Total,
				Downloaded: info.Downloaded,
				Percentage: info.Percentage,
				Speed:      info.Speed,
				ETA:        info.ETA,
				Status:     info.Status,
			}
			progress(typesInfo)
		}
	}

	return a.Manager.DownloadWithProgress(ctx, tool, version, downloadOpts, progressAdapter)
}

func (a *DownloadManagerAdapter) SearchVersions(ctx context.Context, tool string) ([]*types.VersionInfo, error) {
	return a.Manager.SearchVersions(ctx, tool)
}

func (a *DownloadManagerAdapter) GetVersionInfo(ctx context.Context, tool, version string) (*types.VersionInfo, error) {
	return a.Manager.GetVersionInfo(ctx, tool, version)
}

func (a *DownloadManagerAdapter) AddSource(tool string, metadata *types.ToolMetadata) error {
	return a.Manager.AddSource(tool, metadata)
}

// createDownloadManager 创建下载管理器
func createDownloadManager() (download.Manager, error) {
	// 创建基础管理器
	managers, err := createManagers()
	if err != nil {
		return nil, err
	}

	// 创建下载管理器
	downloadManager := download.NewManager(managers.storage, managers.config)

	return downloadManager, nil
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
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

// 初始化命令标志
func init() {
	// install命令的标志
	installCmd.Flags().BoolP("force", "f", false, "强制重新安装")
	installCmd.Flags().BoolP("global", "g", false, "安装后设置为全局版本")

	// search命令的标志
	searchCmd.Flags().IntP("limit", "l", 20, "限制显示的版本数量")
	searchCmd.Flags().Bool("prerelease", false, "包含预发布版本")

	// add-source命令的标志
	addSourceCmd.Flags().String("type", "", "下载源类型 (github, direct, archive)")
	addSourceCmd.Flags().String("repo", "", "GitHub仓库 (格式: owner/repo)")
	addSourceCmd.Flags().String("pattern", "", "资产文件名匹配模式")
	addSourceCmd.Flags().String("url", "", "URL模板")
	addSourceCmd.Flags().String("description", "", "工具描述")
	addSourceCmd.MarkFlagRequired("type")
}
