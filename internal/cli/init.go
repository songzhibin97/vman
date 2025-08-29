package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/songzhibin97/vman/pkg/utils"
)

// initCmd 初始化vman环境
var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "初始化vman环境",
	Long: `初始化vman环境，包括：
- 创建必要的目录结构
- 生成默认配置文件
- 设置shell集成
- 配置代理环境

支持的shell: bash, zsh, fish, powershell

示例:
  vman init          # 自动检测当前shell
  vman init bash     # 为bash生成配置
  vman init zsh      # 为zsh生成配置`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 获取选项
		force, _ := cmd.Flags().GetBool("force")
		skipShell, _ := cmd.Flags().GetBool("skip-shell")

		// 确定shell类型
		var shell string
		if len(args) == 1 {
			shell = args[0]
		} else {
			shell = detectShell()
		}

		// 验证shell类型
		if !isValidShell(shell) {
			return fmt.Errorf("不支持的shell类型: %s", shell)
		}

		// 初始化目录结构
		if err := initDirectories(force); err != nil {
			return fmt.Errorf("初始化目录结构失败: %w", err)
		}

		// 初始化配置文件
		if err := initConfig(force); err != nil {
			return fmt.Errorf("初始化配置文件失败: %w", err)
		}

		// 设置shell集成
		if !skipShell {
			if err := setupShellIntegration(shell, force); err != nil {
				return fmt.Errorf("设置shell集成失败: %w", err)
			}
		}

		// 设置代理环境
		if err := setupProxyEnvironment(); err != nil {
			fmt.Printf("警告: 设置代理环境失败: %v\n", err)
		}

		fmt.Printf("✅ vman初始化完成！\n\n")

		if !skipShell {
			printPostInitInstructions(shell)
		}

		return nil
	},
}

// detectShell 自动检测当前shell
func detectShell() string {
	// 首先检查SHELL环境变量
	if shell := os.Getenv("SHELL"); shell != "" {
		return filepath.Base(shell)
	}

	// 检查特定的环境变量
	switch {
	case os.Getenv("ZSH_VERSION") != "":
		return "zsh"
	case os.Getenv("BASH_VERSION") != "":
		return "bash"
	case os.Getenv("FISH_VERSION") != "":
		return "fish"
	case runtime.GOOS == "windows":
		return "powershell"
	default:
		return "bash" // 默认值
	}
}

// isValidShell 检查shell是否有效
func isValidShell(shell string) bool {
	validShells := []string{"bash", "zsh", "fish", "powershell", "cmd"}
	for _, valid := range validShells {
		if shell == valid {
			return true
		}
	}
	return false
}

// initDirectories 初始化必要的目录结构
func initDirectories(force bool) error {
	fmt.Println("📁 创建目录结构...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	// 创建必要的目录
	directories := []string{
		filepath.Join(homeDir, ".vman"),
		filepath.Join(homeDir, ".vman", "versions"),
		filepath.Join(homeDir, ".vman", "shims"),
		filepath.Join(homeDir, ".vman", "cache"),
		filepath.Join(homeDir, ".vman", "logs"),
		filepath.Join(homeDir, ".vman", "tmp"),
	}

	for _, dir := range directories {
		if utils.FileExists(dir) && !force {
			fmt.Printf("  ⏭  %s (已存在)\n", dir)
			continue
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", dir, err)
		}
		fmt.Printf("  ✅ %s\n", dir)
	}

	return nil
}

// initConfig 初始化配置文件
func initConfig(force bool) error {
	fmt.Println("⚙️  创建配置文件...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	configPath := filepath.Join(homeDir, ".vman", "config.yaml")

	// 检查配置文件是否已存在
	if utils.FileExists(configPath) && !force {
		fmt.Printf("  ⏭  %s (已存在)\n", configPath)
		return nil
	}

	// 创建默认配置
	defaultConfig := `# vman 配置文件
# 全局设置
global:
  # 默认下载源
  default_source: "github"
  # 下载超时时间（秒）
  download_timeout: 300
  # 并发下载数
  concurrent_downloads: 3
  # 代理设置
  proxy: ""

# 源配置
sources:
  github:
    type: "github"
    base_url: "https://api.github.com"
    timeout: 30
  
# 工具特定配置
tools:
  # 示例工具配置
  # kubectl:
  #   source: "github"
  #   repository: "kubernetes/kubernetes"
  #   asset_pattern: "kubectl-{{os}}-{{arch}}"

# 缓存设置
cache:
  # 缓存生存时间（小时）
  ttl: 24
  # 最大缓存大小（MB）
  max_size: 100

# 日志设置
logging:
  # 日志级别: debug, info, warn, error
  level: "info"
  # 日志文件路径
  file: "~/.vman/logs/vman.log"
  # 最大日志文件大小（MB）
  max_size: 10
  # 最大日志文件数量
  max_backups: 5
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}

	fmt.Printf("  ✅ %s\n", configPath)
	return nil
}

// setupShellIntegration 设置shell集成
func setupShellIntegration(shell string, force bool) error {
	fmt.Printf("🐚 设置%s集成...\n", shell)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	// 生成shell初始化脚本
	initScript := generateShellInitScript(shell)
	if initScript == "" {
		return fmt.Errorf("不支持的shell类型: %s", shell)
	}

	// 确定shell配置文件路径
	configFile := getShellConfigFile(shell, homeDir)
	if configFile == "" {
		return fmt.Errorf("无法确定%s的配置文件路径", shell)
	}

	// 检查是否已经集成
	if utils.FileExists(configFile) {
		content, err := os.ReadFile(configFile)
		if err == nil && strings.Contains(string(content), "# vman initialization") {
			if !force {
				fmt.Printf("  ⏭  %s (已集成)\n", configFile)
				return nil
			}
		}
	}

	// 添加初始化脚本到shell配置文件
	file, err := os.OpenFile(configFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开shell配置文件失败: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(initScript); err != nil {
		return fmt.Errorf("写入shell配置失败: %w", err)
	}

	fmt.Printf("  ✅ %s\n", configFile)
	return nil
}

// generateShellInitScript 生成shell初始化脚本
func generateShellInitScript(shell string) string {
	homeDir, _ := os.UserHomeDir()
	vmanDir := filepath.Join(homeDir, ".vman")
	shimsDir := filepath.Join(vmanDir, "shims")

	switch shell {
	case "bash", "zsh":
		return fmt.Sprintf(`
# vman initialization
export VMAN_ROOT="%s"
export PATH="%s:$PATH"

# vman shell integration
if command -v vman >/dev/null 2>&1; then
  eval "$(vman shell-init %s)"
fi
`, vmanDir, shimsDir, shell)

	case "fish":
		return fmt.Sprintf(`
# vman initialization
set -gx VMAN_ROOT "%s"
set -gx PATH "%s" $PATH

# vman shell integration
if command -v vman >/dev/null 2>&1
  vman shell-init fish | source
end
`, vmanDir, shimsDir)

	case "powershell":
		return fmt.Sprintf(`
# vman initialization
$env:VMAN_ROOT = "%s"
$env:PATH = "%s;" + $env:PATH

# vman shell integration
if (Get-Command vman -ErrorAction SilentlyContinue) {
  vman shell-init powershell | Invoke-Expression
}
`, vmanDir, shimsDir)

	default:
		return ""
	}
}

// getShellConfigFile 获取shell配置文件路径
func getShellConfigFile(shell, homeDir string) string {
	switch shell {
	case "bash":
		// 优先使用.bashrc，如果不存在则使用.bash_profile
		bashrc := filepath.Join(homeDir, ".bashrc")
		if utils.FileExists(bashrc) {
			return bashrc
		}
		return filepath.Join(homeDir, ".bash_profile")
	case "zsh":
		return filepath.Join(homeDir, ".zshrc")
	case "fish":
		configDir := filepath.Join(homeDir, ".config", "fish")
		os.MkdirAll(configDir, 0755)
		return filepath.Join(configDir, "config.fish")
	case "powershell":
		// PowerShell配置文件路径比较复杂，这里使用默认路径
		return filepath.Join(homeDir, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	default:
		return ""
	}
}

// setupProxyEnvironment 设置代理环境
func setupProxyEnvironment() error {
	fmt.Println("🔧 设置代理环境...")

	// 初始化代理
	if err := initProxy(); err != nil {
		return err
	}

	// 设置代理
	if err := commandProxy.SetupProxy(); err != nil {
		return err
	}

	fmt.Println("  ✅ 代理环境设置完成")
	return nil
}

// printPostInitInstructions 打印初始化后的说明
func printPostInitInstructions(shell string) {
	fmt.Printf("🚀 使用说明:\n\n")
	fmt.Printf("1. 重新启动shell或运行以下命令以激活vman:\n")

	switch shell {
	case "bash":
		fmt.Printf("   source ~/.bashrc\n")
	case "zsh":
		fmt.Printf("   source ~/.zshrc\n")
	case "fish":
		fmt.Printf("   source ~/.config/fish/config.fish\n")
	case "powershell":
		fmt.Printf("   . $PROFILE\n")
	}

	fmt.Printf("\n2. 安装工具:\n")
	fmt.Printf("   vman install kubectl 1.29.0\n")
	fmt.Printf("   vman install terraform 1.6.0\n")

	fmt.Printf("\n3. 切换版本:\n")
	fmt.Printf("   vman use kubectl 1.29.0\n")
	fmt.Printf("   vman global terraform 1.6.0\n")

	fmt.Printf("\n4. 查看状态:\n")
	fmt.Printf("   vman list\n")
	fmt.Printf("   vman current\n")

	fmt.Printf("\n📚 获取更多帮助: vman --help\n")
}

func init() {
	// 添加init命令到根命令
	rootCmd.AddCommand(initCmd)

	// 添加选项
	initCmd.Flags().BoolP("force", "f", false, "强制重新初始化（覆盖现有文件）")
	initCmd.Flags().Bool("skip-shell", false, "跳过shell集成设置")
}
