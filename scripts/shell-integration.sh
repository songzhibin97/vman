#!/bin/bash
# vman Shell Integration Setup Script

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
VMAN_DIR="${VMAN_DIR:-$HOME/.vman}"
VMAN_SHIMS_DIR="${VMAN_SHIMS_DIR:-$VMAN_DIR/shims}"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# 函数定义
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检测shell类型
detect_shell() {
    if [ -n "$ZSH_VERSION" ]; then
        echo "zsh"
    elif [ -n "$BASH_VERSION" ]; then
        echo "bash"
    elif [ -n "$FISH_VERSION" ]; then
        echo "fish"
    else
        # 检查$SHELL环境变量
        case "$SHELL" in
            */zsh)
                echo "zsh"
                ;;
            */bash)
                echo "bash"
                ;;
            */fish)
                echo "fish"
                ;;
            *)
                echo "unknown"
                ;;
        esac
    fi
}

# 获取shell配置文件路径
get_shell_config() {
    local shell_type="$1"
    case "$shell_type" in
        "bash")
            if [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        "zsh")
            echo "$HOME/.zshrc"
            ;;
        "fish")
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# 检查vman是否安装
check_vman_installed() {
    if ! command -v vman &> /dev/null; then
        log_error "vman command not found. Please install vman first."
        exit 1
    fi
}

# 创建必要的目录
create_directories() {
    log_info "Creating necessary directories..."
    
    mkdir -p "$VMAN_DIR"
    mkdir -p "$VMAN_SHIMS_DIR"
    mkdir -p "$VMAN_DIR/logs"
    mkdir -p "$VMAN_DIR/cache"
    mkdir -p "$VMAN_DIR/tmp"
    
    log_success "Directories created successfully"
}

# 安装shell钩子
install_shell_hook() {
    local shell_type="$1"
    local config_file
    
    config_file=$(get_shell_config "$shell_type")
    
    log_info "Installing shell hook for $shell_type..."
    
    # 使用vman命令生成shell钩子
    if vman init "$shell_type" > /dev/null 2>&1; then
        local hook_content
        hook_content=$(vman init "$shell_type")
        
        # 检查是否已经安装
        if [ -f "$config_file" ] && grep -q "# vman shell integration" "$config_file"; then
            log_warning "Shell hook already installed in $config_file"
            return 0
        fi
        
        # 创建配置文件目录（如果不存在）
        mkdir -p "$(dirname "$config_file")"
        
        # 添加钩子到配置文件
        {
            echo ""
            echo "# vman shell integration"
            echo "$hook_content"
            echo "# vman shell integration"
        } >> "$config_file"
        
        log_success "Shell hook installed to $config_file"
        return 0
    else
        log_error "Failed to generate shell hook for $shell_type"
        return 1
    fi
}

# 添加PATH设置
setup_path() {
    local shell_type="$1"
    local config_file
    
    config_file=$(get_shell_config "$shell_type")
    
    log_info "Setting up PATH for $shell_type..."
    
    # 检查PATH是否已经设置
    if [ -f "$config_file" ] && grep -q "$VMAN_SHIMS_DIR" "$config_file"; then
        log_warning "PATH already configured in $config_file"
        return 0
    fi
    
    # 根据shell类型添加PATH设置
    case "$shell_type" in
        "fish")
            {
                echo ""
                echo "# vman PATH setup"
                echo "set -gx PATH \"$VMAN_SHIMS_DIR\" \$PATH"
                echo "# vman PATH setup"
            } >> "$config_file"
            ;;
        *)
            {
                echo ""
                echo "# vman PATH setup"
                echo "export PATH=\"$VMAN_SHIMS_DIR:\$PATH\""
                echo "# vman PATH setup"
            } >> "$config_file"
            ;;
    esac
    
    log_success "PATH configured in $config_file"
}

# 生成所有shims
generate_shims() {
    log_info "Generating shims for installed tools..."
    
    if vman proxy rehash > /dev/null 2>&1; then
        log_success "Shims generated successfully"
    else
        log_warning "Failed to generate shims (this is normal if no tools are installed)"
    fi
}

# 验证安装
verify_installation() {
    log_info "Verifying installation..."
    
    # 检查shims目录是否在PATH中
    if echo "$PATH" | grep -q "$VMAN_SHIMS_DIR"; then
        log_success "Shims directory is in PATH"
    else
        log_warning "Shims directory is not in current PATH (restart shell to activate)"
    fi
    
    # 检查代理状态
    if vman proxy status > /dev/null 2>&1; then
        log_success "vman proxy is working"
    else
        log_warning "vman proxy status check failed"
    fi
}

# 显示后续步骤
show_next_steps() {
    local shell_type="$1"
    local config_file
    
    config_file=$(get_shell_config "$shell_type")
    
    echo ""
    log_success "vman shell integration setup completed!"
    echo ""
    echo "Next steps:"
    echo "1. Restart your shell or run:"
    echo "   source $config_file"
    echo ""
    echo "2. Install some tools:"
    echo "   vman install kubectl latest"
    echo "   vman install terraform latest"
    echo ""
    echo "3. Check the status:"
    echo "   vman proxy status"
    echo ""
    echo "4. Use your tools transparently:"
    echo "   kubectl version"
    echo "   terraform version"
    echo ""
}

# 卸载函数
uninstall_shell_integration() {
    local shell_type="$1"
    local config_file
    
    config_file=$(get_shell_config "$shell_type")
    
    log_info "Uninstalling shell integration for $shell_type..."
    
    if [ -f "$config_file" ]; then
        # 使用sed删除vman相关的行
        if command -v sed &> /dev/null; then
            # 创建备份
            cp "$config_file" "$config_file.vman.bak"
            
            # 删除vman集成部分
            sed '/# vman shell integration/,/# vman shell integration/d' "$config_file.vman.bak" > "$config_file.tmp"
            sed '/# vman PATH setup/,/# vman PATH setup/d' "$config_file.tmp" > "$config_file"
            
            # 清理临时文件
            rm -f "$config_file.tmp"
            
            log_success "Shell integration removed from $config_file"
            log_info "Backup saved to $config_file.vman.bak"
        else
            log_error "sed command not found, please manually remove vman integration from $config_file"
        fi
    else
        log_warning "Config file $config_file does not exist"
    fi
}

# 主函数
main() {
    local action="${1:-install}"
    local shell_type="${2:-}"
    
    # 检测shell类型
    if [ -z "$shell_type" ]; then
        shell_type=$(detect_shell)
        if [ "$shell_type" = "unknown" ]; then
            log_error "Cannot detect shell type. Please specify shell type manually."
            echo "Usage: $0 [install|uninstall] [bash|zsh|fish]"
            exit 1
        fi
    fi
    
    log_info "Detected shell: $shell_type"
    
    case "$action" in
        "install")
            check_vman_installed
            create_directories
            setup_path "$shell_type"
            install_shell_hook "$shell_type"
            generate_shims
            verify_installation
            show_next_steps "$shell_type"
            ;;
        "uninstall")
            uninstall_shell_integration "$shell_type"
            log_success "Shell integration uninstalled"
            echo "Note: You may need to restart your shell for changes to take effect"
            ;;
        *)
            echo "Usage: $0 [install|uninstall] [bash|zsh|fish]"
            echo ""
            echo "Actions:"
            echo "  install   - Install vman shell integration (default)"
            echo "  uninstall - Remove vman shell integration"
            echo ""
            echo "Shells:"
            echo "  bash - Bash shell"
            echo "  zsh  - Zsh shell"
            echo "  fish - Fish shell"
            echo ""
            echo "If no shell is specified, it will be auto-detected."
            exit 1
            ;;
    esac
}

# 脚本入口点
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi