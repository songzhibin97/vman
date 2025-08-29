package proxy

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// ShellIntegrator Shell集成器接口
type ShellIntegrator interface {
	// GenerateShellHook 生成shell钩子脚本
	GenerateShellHook(shellType string) (string, error)

	// InstallShellHook 安装shell钩子
	InstallShellHook(shellType string, vmanPath string) error

	// UninstallShellHook 卸载shell钩子
	UninstallShellHook(shellType string) error

	// GenerateShim 生成命令垫片
	GenerateShim(toolName, shimPath, vmanPath string) error

	// GenerateActivationScript 生成激活脚本
	GenerateActivationScript(shellType, vmanPath string) (string, error)

	// DetectShell 检测当前使用的shell
	DetectShell() string

	// GetSupportedShells 获取支持的shell列表
	GetSupportedShells() []string

	// ValidateShellSupport 验证shell是否支持
	ValidateShellSupport(shellType string) bool
}

// DefaultShellIntegrator 默认Shell集成器实现
type DefaultShellIntegrator struct {
	fs     afero.Fs
	logger *logrus.Logger
}

// ShellHookData shell钩子模板数据
type ShellHookData struct {
	VmanPath      string
	ShimDir       string
	ConfigDir     string
	ShellType     string
	PathSeparator string
}

// ShimData 垫片模板数据
type ShimData struct {
	ToolName  string
	VmanPath  string
	ShellType string
	IsWindows bool
}

// NewShellIntegrator 创建新的Shell集成器
func NewShellIntegrator() ShellIntegrator {
	return NewShellIntegratorWithFs(afero.NewOsFs())
}

// NewShellIntegratorWithFs 使用指定文件系统创建Shell集成器（用于测试）
func NewShellIntegratorWithFs(fs afero.Fs) ShellIntegrator {
	return &DefaultShellIntegrator{
		fs:     fs,
		logger: logrus.New(),
	}
}

// GenerateShellHook 生成shell钩子脚本
func (si *DefaultShellIntegrator) GenerateShellHook(shellType string) (string, error) {
	si.logger.Debugf("Generating shell hook for: %s", shellType)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	data := ShellHookData{
		VmanPath:      "vman", // 假设vman已在PATH中
		ShimDir:       filepath.Join(homeDir, ".vman", "shims"),
		ConfigDir:     filepath.Join(homeDir, ".vman"),
		ShellType:     shellType,
		PathSeparator: getPathSeparator(),
	}

	var templateStr string
	switch shellType {
	case "bash", "zsh":
		templateStr = bashZshHookTemplate
	case "fish":
		templateStr = fishHookTemplate
	case "cmd":
		templateStr = cmdHookTemplate
	case "powershell":
		templateStr = powershellHookTemplate
	default:
		return "", fmt.Errorf("unsupported shell type: %s", shellType)
	}

	tmpl, err := template.New("hook").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// InstallShellHook 安装shell钩子
func (si *DefaultShellIntegrator) InstallShellHook(shellType string, vmanPath string) error {
	si.logger.Infof("Installing shell hook for: %s", shellType)

	// 生成钩子脚本
	hookScript, err := si.GenerateShellHook(shellType)
	if err != nil {
		return fmt.Errorf("failed to generate hook script: %w", err)
	}

	// 获取shell配置文件路径
	configPath, err := si.getShellConfigPath(shellType)
	if err != nil {
		return fmt.Errorf("failed to get shell config path: %w", err)
	}

	// 读取现有配置
	var existingContent string
	if exists, _ := afero.Exists(si.fs, configPath); exists {
		content, err := afero.ReadFile(si.fs, configPath)
		if err != nil {
			return fmt.Errorf("failed to read shell config: %w", err)
		}
		existingContent = string(content)
	}

	// 检查是否已安装钩子
	vmanMarker := getVmanMarker(shellType)
	if strings.Contains(existingContent, vmanMarker) {
		si.logger.Infof("Shell hook already installed for: %s", shellType)
		return nil
	}

	// 添加钩子脚本
	newContent := existingContent
	if newContent != "" && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += "\n" + vmanMarker + "\n"
	newContent += hookScript + "\n"
	newContent += vmanMarker + "\n"

	// 确保配置目录存在
	if err := si.fs.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入配置文件
	if err := afero.WriteFile(si.fs, configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write shell config: %w", err)
	}

	si.logger.Infof("Successfully installed shell hook for: %s", shellType)
	return nil
}

// UninstallShellHook 卸载shell钩子
func (si *DefaultShellIntegrator) UninstallShellHook(shellType string) error {
	si.logger.Infof("Uninstalling shell hook for: %s", shellType)

	// 获取shell配置文件路径
	configPath, err := si.getShellConfigPath(shellType)
	if err != nil {
		return fmt.Errorf("failed to get shell config path: %w", err)
	}

	// 检查配置文件是否存在
	if exists, _ := afero.Exists(si.fs, configPath); !exists {
		si.logger.Debugf("Shell config file does not exist: %s", configPath)
		return nil
	}

	// 读取现有配置
	content, err := afero.ReadFile(si.fs, configPath)
	if err != nil {
		return fmt.Errorf("failed to read shell config: %w", err)
	}

	// 移除vman钩子部分
	vmanMarker := getVmanMarker(shellType)
	newContent := si.removeVmanSection(string(content), vmanMarker)

	// 写入更新后的配置
	if err := afero.WriteFile(si.fs, configPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write shell config: %w", err)
	}

	si.logger.Infof("Successfully uninstalled shell hook for: %s", shellType)
	return nil
}

// GenerateShim 生成命令垫片
func (si *DefaultShellIntegrator) GenerateShim(toolName, shimPath, vmanPath string) error {
	si.logger.Debugf("Generating shim for tool: %s", toolName)

	data := ShimData{
		ToolName:  toolName,
		VmanPath:  vmanPath,
		ShellType: si.DetectShell(),
		IsWindows: runtime.GOOS == "windows",
	}

	var templateStr string
	if runtime.GOOS == "windows" {
		templateStr = windowsShimTemplate
	} else {
		templateStr = unixShimTemplate
	}

	tmpl, err := template.New("shim").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse shim template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute shim template: %w", err)
	}

	// 确保shim目录存在
	if err := si.fs.MkdirAll(filepath.Dir(shimPath), 0755); err != nil {
		return fmt.Errorf("failed to create shim directory: %w", err)
	}

	// 写入shim文件
	shimContent := buf.String()
	if err := afero.WriteFile(si.fs, shimPath, []byte(shimContent), 0755); err != nil {
		return fmt.Errorf("failed to write shim file: %w", err)
	}

	si.logger.Infof("Successfully generated shim for: %s", toolName)
	return nil
}

// GenerateActivationScript 生成激活脚本
func (si *DefaultShellIntegrator) GenerateActivationScript(shellType, vmanPath string) (string, error) {
	hookScript, err := si.GenerateShellHook(shellType)
	if err != nil {
		return "", fmt.Errorf("failed to generate hook script: %w", err)
	}

	activationTemplate := `#!/bin/bash
# vman activation script for %s
# Add this to your shell profile or run: source <(vman init %s)

%s

echo "vman activated for %s shell"
`

	return fmt.Sprintf(activationTemplate, shellType, shellType, hookScript, shellType), nil
}

// DetectShell 检测当前使用的shell
func (si *DefaultShellIntegrator) DetectShell() string {
	// 首先检查SHELL环境变量
	shell := os.Getenv("SHELL")
	if shell != "" {
		shellName := filepath.Base(shell)
		if si.ValidateShellSupport(shellName) {
			return shellName
		}
	}

	// 检查Windows环境
	if runtime.GOOS == "windows" {
		// 检查PowerShell
		if os.Getenv("PSModulePath") != "" {
			return "powershell"
		}
		return "cmd"
	}

	// 默认返回bash
	return "bash"
}

// GetSupportedShells 获取支持的shell列表
func (si *DefaultShellIntegrator) GetSupportedShells() []string {
	return []string{"bash", "zsh", "fish", "cmd", "powershell"}
}

// ValidateShellSupport 验证shell是否支持
func (si *DefaultShellIntegrator) ValidateShellSupport(shellType string) bool {
	supportedShells := si.GetSupportedShells()
	for _, supported := range supportedShells {
		if supported == shellType {
			return true
		}
	}
	return false
}

// getShellConfigPath 获取shell配置文件路径
func (si *DefaultShellIntegrator) getShellConfigPath(shellType string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	switch shellType {
	case "bash":
		// 优先使用 .bash_profile，然后是 .bashrc
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if exists, _ := afero.Exists(si.fs, bashProfile); exists {
			return bashProfile, nil
		}
		return filepath.Join(homeDir, ".bashrc"), nil
	case "zsh":
		return filepath.Join(homeDir, ".zshrc"), nil
	case "fish":
		return filepath.Join(homeDir, ".config", "fish", "config.fish"), nil
	case "cmd":
		// Windows CMD 不支持持久化配置，返回批处理文件
		return filepath.Join(homeDir, "vman_init.cmd"), nil
	case "powershell":
		// PowerShell profile
		return filepath.Join(homeDir, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"), nil
	default:
		return "", fmt.Errorf("unsupported shell type: %s", shellType)
	}
}

// getVmanMarker 获取vman标记注释
func getVmanMarker(shellType string) string {
	switch shellType {
	case "fish":
		return "# vman shell integration"
	case "cmd":
		return "REM vman shell integration"
	case "powershell":
		return "# vman shell integration"
	default:
		return "# vman shell integration"
	}
}

// removeVmanSection 移除vman配置段落
func (si *DefaultShellIntegrator) removeVmanSection(content, marker string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	inVmanSection := false

	for _, line := range lines {
		if strings.TrimSpace(line) == marker {
			inVmanSection = !inVmanSection
			continue
		}

		if !inVmanSection {
			newLines = append(newLines, line)
		}
	}

	return strings.Join(newLines, "\n")
}

// Shell钩子模板
const bashZshHookTemplate = `
# vman shell integration
export VMAN_DIR="{{.ConfigDir}}"
export VMAN_SHIMS_DIR="{{.ShimDir}}"

# Add shims to PATH if not already present
if [[ ":$PATH:" != *":{{.ShimDir}}:"* ]]; then
    export PATH="{{.ShimDir}}:$PATH"
fi

# Command not found hook
command_not_found_handle() {
    if command -v vman >/dev/null 2>&1; then
        vman exec "$1" "$@"
    else
        echo "bash: $1: command not found"
        return 127
    fi
}

# Change directory hook for project-specific tool versions
vman_cd_hook() {
    if command -v vman >/dev/null 2>&1; then
        vman rehash >/dev/null 2>&1
    fi
}

# Set up cd hook
if [[ "${{.ShellType}}" == "zsh" ]]; then
    autoload -U add-zsh-hook
    add-zsh-hook chpwd vman_cd_hook
else
    PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND$'\n'}vman_cd_hook"
fi
`

const fishHookTemplate = `
# vman shell integration
set -gx VMAN_DIR "{{.ConfigDir}}"
set -gx VMAN_SHIMS_DIR "{{.ShimDir}}"

# Add shims to PATH
if not contains "{{.ShimDir}}" $PATH
    set -gx PATH "{{.ShimDir}}" $PATH
end

# Command not found hook
function fish_command_not_found
    if command -v vman >/dev/null 2>&1
        vman exec $argv[1] $argv
    else
        echo "fish: Unknown command: $argv[1]"
        return 127
    end
end

# Change directory hook
function vman_cd_hook --on-variable PWD
    if command -v vman >/dev/null 2>&1
        vman rehash >/dev/null 2>&1
    end
end
`

const cmdHookTemplate = `
REM vman shell integration
@echo off
set VMAN_DIR={{.ConfigDir}}
set VMAN_SHIMS_DIR={{.ShimDir}}
set PATH={{.ShimDir}};%PATH%
`

const powershellHookTemplate = `
# vman shell integration
$env:VMAN_DIR = "{{.ConfigDir}}"
$env:VMAN_SHIMS_DIR = "{{.ShimDir}}"

# Add shims to PATH
if ($env:PATH -notlike "*{{.ShimDir}}*") {
    $env:PATH = "{{.ShimDir}}" + [System.IO.Path]::PathSeparator + $env:PATH
}

# Command not found hook
$ExecutionContext.InvokeCommand.CommandNotFoundAction = {
    param($CommandName, $CommandLookupEventArgs)
    
    if (Get-Command vman -ErrorAction SilentlyContinue) {
        try {
            vman exec $CommandName @args
            $CommandLookupEventArgs.StopSearch = $true
        } catch {
            # Continue with default behavior
        }
    }
}
`

// Shim模板
const unixShimTemplate = `#!/bin/bash
# vman shim for {{.ToolName}}
exec "{{.VmanPath}}" exec "{{.ToolName}}" "$@"
`

const windowsShimTemplate = `@echo off
REM vman shim for {{.ToolName}}
"{{.VmanPath}}" exec "{{.ToolName}}" %*
`
