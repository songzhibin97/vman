package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionCmd Tab补全命令
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "生成shell自动补全脚本",
	Long: `生成指定shell的自动补全脚本。

支持的shell:
- bash
- zsh  
- fish
- powershell

安装方法:

Bash:
  # 临时启用
  source <(vman completion bash)
  
  # 永久启用
  vman completion bash > /etc/bash_completion.d/vman

Zsh:
  # 如果尚未完成，需要在 ~/.zshrc 中启用补全:
  echo "autoload -U compinit; compinit" >> ~/.zshrc
  
  # 临时启用
  source <(vman completion zsh)
  
  # 永久启用 (方法1: 系统范围)
  vman completion zsh > "${fpath[1]}/_vman"
  
  # 永久启用 (方法2: 用户目录)
  vman completion zsh > ~/.zsh/completions/_vman

Fish:
  vman completion fish | source
  
  # 永久启用
  vman completion fish > ~/.config/fish/completions/vman.fish

PowerShell:
  # 临时启用
  vman completion powershell | Out-String | Invoke-Expression
  
  # 永久启用，将输出添加到您的PowerShell配置文件中`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run:                   runCompletionCommand,
}

func runCompletionCommand(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		cmd.Root().GenBashCompletion(os.Stdout)
	case "zsh":
		cmd.Root().GenZshCompletion(os.Stdout)
	case "fish":
		cmd.Root().GenFishCompletion(os.Stdout, true)
	case "powershell":
		cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
	}
}

// 自定义补全函数

// completeToolNames 补全工具名称
func completeToolNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// 创建管理器获取已安装的工具列表
	managers, err := createManagers()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	tools, err := managers.version.ListAllTools()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return tools, cobra.ShellCompDirectiveNoFileComp
}

// completeVersions 补全版本号
func completeVersions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return nil, cobra.ShellCompDirectiveError
	}

	tool := args[0]

	// 创建管理器获取工具的版本列表
	managers, err := createManagers()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	versions, err := managers.version.ListVersions(tool)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// 添加特殊别名
	versions = append(versions, "latest", "system")

	return versions, cobra.ShellCompDirectiveNoFileComp
}

// completeShells 补全shell类型
func completeShells(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	shells := []string{"bash", "zsh", "fish", "powershell", "cmd"}
	return shells, cobra.ShellCompDirectiveNoFileComp
}

// completeSourceTypes 补全源类型
func completeSourceTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{"github", "direct", "archive"}
	return types, cobra.ShellCompDirectiveNoFileComp
}

// setupCompletions 设置所有命令的补全
func setupCompletions() {
	// init命令补全
	if initCmd != nil {
		initCmd.ValidArgsFunction = completeShells
	}

	// install命令补全
	if installCmd != nil {
		installCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// 第一个参数：工具名（可以是任意字符串，但提供已安装的作为建议）
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				// 第二个参数：版本号
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// list命令补全
	if listCmd != nil {
		listCmd.ValidArgsFunction = completeToolNames
	}

	// use命令补全
	if useCmd != nil {
		useCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// remove命令补全
	if removeCmd != nil {
		removeCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// current命令补全
	if currentCmd != nil {
		currentCmd.ValidArgsFunction = completeToolNames
	}

	// global命令补全
	if globalCmd != nil {
		globalCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// local命令补全
	if localCmd != nil {
		localCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// register命令补全
	if registerCmd != nil {
		registerCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// 工具名
				return nil, cobra.ShellCompDirectiveNoFileComp
			} else if len(args) == 1 {
				// 版本号
				return nil, cobra.ShellCompDirectiveNoFileComp
			} else if len(args) == 2 {
				// 二进制文件路径
				return nil, cobra.ShellCompDirectiveDefault
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// search命令补全
	if searchCmd != nil {
		searchCmd.ValidArgsFunction = completeToolNames
	}

	// update命令补全
	if updateCmd != nil {
		updateCmd.ValidArgsFunction = completeToolNames
	}

	// which命令补全
	if whichCmd != nil {
		whichCmd.ValidArgsFunction = completeToolNames
	}

	// uninstall命令补全
	if uninstallCmd != nil {
		uninstallCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeToolNames(cmd, args, toComplete)
			} else if len(args) == 1 {
				return completeVersions(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	// proxy exec命令补全
	if execCmd != nil {
		execCmd.ValidArgsFunction = completeToolNames
	}

	// add-source命令补全
	if addSourceCmd != nil {
		addSourceCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// 工具名
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		// 标志补全
		addSourceCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeSourceTypes(cmd, args, toComplete)
		})
	}

	// remove-source命令补全
	if removeSourceCmd != nil {
		removeSourceCmd.ValidArgsFunction = completeToolNames
	}
}

// generateCompletionScript 生成自定义补全脚本
func generateCompletionScript(shell string) string {
	switch shell {
	case "bash":
		return `
# vman bash completion

_vman_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # 基础命令补全
    if [[ ${COMP_CWORD} == 1 ]]; then
        opts="init install list use remove current update search register global local uninstall which proxy completion help"
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi

    # 工具名称补全
    local tools=$(vman list 2>/dev/null | grep -E "^\s+[a-zA-Z]" | awk '{print $1}')
    
    case "${prev}" in
        install|use|remove|current|update|search|register|global|local|uninstall|which)
            COMPREPLY=( $(compgen -W "${tools}" -- ${cur}) )
            ;;
        *)
            ;;
    esac
}

complete -F _vman_completion vman
`

	case "zsh":
		return `
# vman zsh completion

#compdef vman

_vman() {
    local context state line
    typeset -A opt_args

    _arguments -C \
        '1: :->command' \
        '*: :->args'

    case $state in
        command)
            local commands=(
                'init:初始化vman环境'
                'install:安装工具版本'
                'list:列出工具版本'
                'use:切换版本'
                'remove:删除版本'
                'current:显示当前版本'
                'update:更新工具'
                'search:搜索版本'
                'register:注册工具版本'
                'global:设置全局版本'
                'local:设置本地版本'
                'uninstall:卸载版本'
                'which:显示工具路径'
                'proxy:代理管理'
                'completion:生成补全脚本'
                'help:显示帮助'
            )
            _describe 'commands' commands
            ;;
        args)
            case ${words[2]} in
                install|use|remove|current|update|search|register|global|local|uninstall|which)
                    local tools=($(vman list 2>/dev/null | grep -E "^\s+[a-zA-Z]" | awk '{print $1}'))
                    _describe 'tools' tools
                    ;;
            esac
            ;;
    esac
}

_vman "$@"
`

	case "fish":
		return `
# vman fish completion

function __vman_tools
    vman list 2>/dev/null | grep -E "^\s+[a-zA-Z]" | awk '{print $1}'
end

function __vman_versions
    if test (count $argv) -gt 0
        vman list $argv[1] 2>/dev/null | grep -E "^\s+[0-9v]" | awk '{print $1}' | sed 's/\*//'
    end
end

# 基础命令补全
complete -c vman -f -n '__fish_use_subcommand' -a 'init' -d '初始化vman环境'
complete -c vman -f -n '__fish_use_subcommand' -a 'install' -d '安装工具版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'list' -d '列出工具版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'use' -d '切换版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'remove' -d '删除版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'current' -d '显示当前版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'update' -d '更新工具'
complete -c vman -f -n '__fish_use_subcommand' -a 'search' -d '搜索版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'register' -d '注册工具版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'global' -d '设置全局版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'local' -d '设置本地版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'uninstall' -d '卸载版本'
complete -c vman -f -n '__fish_use_subcommand' -a 'which' -d '显示工具路径'
complete -c vman -f -n '__fish_use_subcommand' -a 'proxy' -d '代理管理'
complete -c vman -f -n '__fish_use_subcommand' -a 'completion' -d '生成补全脚本'

# 工具名称补全
complete -c vman -f -n '__fish_seen_subcommand_from install use remove current update search register global local uninstall which' -a '(__vman_tools)'

# 版本号补全
complete -c vman -f -n '__fish_seen_subcommand_from use remove register global local uninstall' -a '(__vman_versions (commandline -opc)[3])'

# shell补全
complete -c vman -f -n '__fish_seen_subcommand_from init completion' -a 'bash zsh fish powershell'
`

	default:
		return ""
	}
}

// installCompletionScript 安装补全脚本
func installCompletionScript(shell string) error {
	script := generateCompletionScript(shell)
	if script == "" {
		return fmt.Errorf("不支持的shell: %s", shell)
	}

	fmt.Print(script)
	return nil
}

// 注册completion命令
func init() {
	rootCmd.AddCommand(completionCmd)

	// 在根命令初始化完成后设置补全
	cobra.OnInitialize(setupCompletions)
}
