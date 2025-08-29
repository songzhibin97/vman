#!/bin/bash

# vman 测试脚本
# 这个脚本提供了完整的测试运行功能，包括单元测试、集成测试、功能测试和基准测试

set -e  # 遇到错误时退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_FILE="coverage.out"
COVERAGE_HTML="coverage.html"
TEST_TIMEOUT="30m"
BENCHMARK_TIME="10s"

# 帮助信息
show_help() {
    cat << EOF
vman 测试脚本

用法: $0 [选项] [测试类型]

测试类型:
    unit        运行单元测试
    integration 运行集成测试
    functional  运行功能测试
    all         运行所有测试（默认）
    bench       运行基准测试
    coverage    生成测试覆盖率报告
    clean       清理测试文件

选项:
    -v, --verbose    详细输出
    -s, --short      跳过长时间运行的测试
    -r, --race       启用竞态检测
    -c, --count N    运行测试 N 次（默认 1）
    -p, --parallel N 并行运行测试（默认 GOMAXPROCS）
    --timeout T      设置测试超时时间（默认 30m）
    --bench-time T   设置基准测试时间（默认 10s）
    --no-cache       禁用测试缓存
    --fail-fast      遇到第一个失败就停止
    -h, --help       显示此帮助信息

示例:
    $0                    # 运行所有测试
    $0 unit -v           # 详细运行单元测试
    $0 integration -s    # 跳过长时间集成测试
    $0 bench             # 运行基准测试
    $0 coverage          # 生成覆盖率报告
    $0 clean             # 清理测试文件

环境变量:
    VMAN_TEST_VERBOSE    设置为 "true" 启用详细输出
    VMAN_TEST_SHORT      设置为 "true" 跳过长时间测试
    VMAN_TEST_RACE       设置为 "true" 启用竞态检测
EOF
}

# 日志函数
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

# 检查依赖
check_dependencies() {
    log_info "检查依赖项..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装或不在 PATH 中"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go 版本: $GO_VERSION"
    
    # 检查 Go 模块
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        log_error "未找到 go.mod 文件"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 设置测试环境
setup_test_env() {
    log_info "设置测试环境..."
    
    # 切换到项目根目录
    cd "$PROJECT_ROOT"
    
    # 下载依赖
    go mod download
    go mod tidy
    
    # 创建测试目录
    mkdir -p test/tmp
    mkdir -p test/reports
    mkdir -p test/artifacts
    
    log_success "测试环境设置完成"
}

# 构建测试标志
build_test_flags() {
    local flags=""
    
    # 基本标志
    flags="$flags -timeout=$TEST_TIMEOUT"
    
    # 详细输出
    if [ "$VERBOSE" = "true" ]; then
        flags="$flags -v"
    fi
    
    # 短测试
    if [ "$SHORT" = "true" ]; then
        flags="$flags -short"
    fi
    
    # 竞态检测
    if [ "$RACE" = "true" ]; then
        flags="$flags -race"
    fi
    
    # 运行次数
    if [ -n "$COUNT" ]; then
        flags="$flags -count=$COUNT"
    fi
    
    # 并行度
    if [ -n "$PARALLEL" ]; then
        flags="$flags -parallel=$PARALLEL"
    fi
    
    # 禁用缓存
    if [ "$NO_CACHE" = "true" ]; then
        flags="$flags -count=1"
    fi
    
    # 快速失败
    if [ "$FAIL_FAST" = "true" ]; then
        flags="$flags -failfast"
    fi
    
    echo "$flags"
}

# 运行单元测试
run_unit_tests() {
    log_info "运行单元测试..."
    
    local flags=$(build_test_flags)
    local packages=(
        "./internal/config/..."
        "./internal/version/..."
        "./internal/storage/..."
        "./internal/download/..."
        "./internal/proxy/..."
        "./internal/cli/..."
        "./pkg/..."
    )
    
    for package in "${packages[@]}"; do
        log_info "测试包: $package"
        if ! go test $flags "$package"; then
            log_error "单元测试失败: $package"
            return 1
        fi
    done
    
    log_success "单元测试完成"
}

# 运行集成测试
run_integration_tests() {
    log_info "运行集成测试..."
    
    local flags=$(build_test_flags)
    local test_dir="./test/integration/..."
    
    log_info "测试目录: $test_dir"
    if ! go test $flags "$test_dir"; then
        log_error "集成测试失败"
        return 1
    fi
    
    log_success "集成测试完成"
}

# 运行功能测试
run_functional_tests() {
    log_info "运行功能测试..."
    
    local flags=$(build_test_flags)
    local test_dir="./test/functional/..."
    
    log_info "测试目录: $test_dir"
    if ! go test $flags "$test_dir"; then
        log_error "功能测试失败"
        return 1
    fi
    
    log_success "功能测试完成"
}

# 运行基准测试
run_benchmark_tests() {
    log_info "运行基准测试..."
    
    local flags="-bench=. -benchtime=$BENCHMARK_TIME -benchmem"
    
    if [ "$VERBOSE" = "true" ]; then
        flags="$flags -v"
    fi
    
    local packages=(
        "./internal/config/..."
        "./internal/version/..."
        "./internal/download/..."
        "./internal/proxy/..."
    )
    
    local bench_file="test/reports/benchmark.txt"
    echo "# vman 基准测试报告" > "$bench_file"
    echo "# 生成时间: $(date)" >> "$bench_file"
    echo "" >> "$bench_file"
    
    for package in "${packages[@]}"; do
        log_info "基准测试包: $package"
        echo "## $package" >> "$bench_file"
        if go test $flags "$package" >> "$bench_file" 2>&1; then
            log_success "基准测试完成: $package"
        else
            log_warning "基准测试失败或无基准测试: $package"
        fi
        echo "" >> "$bench_file"
    done
    
    log_success "基准测试报告保存到: $bench_file"
}

# 生成覆盖率报告
generate_coverage() {
    log_info "生成测试覆盖率报告..."
    
    local flags=$(build_test_flags)
    local coverage_profile="test/reports/$COVERAGE_FILE"
    local coverage_html="test/reports/$COVERAGE_HTML"
    
    # 运行测试并生成覆盖率数据
    log_info "收集覆盖率数据..."
    go test $flags -coverprofile="$coverage_profile" -covermode=atomic ./...
    
    if [ ! -f "$coverage_profile" ]; then
        log_error "覆盖率文件生成失败"
        return 1
    fi
    
    # 生成HTML报告
    log_info "生成HTML覆盖率报告..."
    go tool cover -html="$coverage_profile" -o "$coverage_html"
    
    # 显示覆盖率统计
    log_info "覆盖率统计:"
    go tool cover -func="$coverage_profile" | tail -1
    
    # 生成详细覆盖率报告
    local coverage_report="test/reports/coverage-report.txt"
    echo "# vman 覆盖率报告" > "$coverage_report"
    echo "# 生成时间: $(date)" >> "$coverage_report"
    echo "" >> "$coverage_report"
    go tool cover -func="$coverage_profile" >> "$coverage_report"
    
    log_success "覆盖率报告生成完成:"
    log_success "  详细报告: $coverage_report"
    log_success "  HTML报告: $coverage_html"
}

# 运行所有测试
run_all_tests() {
    log_info "运行所有测试..."
    
    local start_time=$(date +%s)
    
    # 运行各种测试
    run_unit_tests || return 1
    run_integration_tests || return 1
    run_functional_tests || return 1
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_success "所有测试完成，耗时: ${duration}秒"
}

# 清理测试文件
clean_test_files() {
    log_info "清理测试文件..."
    
    # 删除测试生成的文件
    rm -rf test/tmp/*
    rm -f test/reports/*
    rm -f *.out
    rm -f *.html
    
    # 清理Go缓存
    go clean -testcache
    go clean -cache
    
    log_success "测试文件清理完成"
}

# 验证代码质量
verify_code_quality() {
    log_info "验证代码质量..."
    
    # 格式化检查
    if ! gofmt -l . | grep -q .; then
        log_success "代码格式检查通过"
    else
        log_warning "发现未格式化的代码文件:"
        gofmt -l .
    fi
    
    # 静态检查（如果安装了工具）
    if command -v golangci-lint &> /dev/null; then
        log_info "运行 golangci-lint..."
        golangci-lint run
    else
        log_warning "golangci-lint 未安装，跳过静态检查"
    fi
    
    # 模块整理检查
    if go mod tidy && git diff --exit-code go.mod go.sum; then
        log_success "Go 模块整理检查通过"
    else
        log_warning "go.mod 或 go.sum 需要整理"
    fi
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    local report_file="test/reports/test-summary.md"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    cat > "$report_file" << EOF
# vman 测试报告

**生成时间:** $timestamp
**Go 版本:** $(go version)
**平台:** $(uname -a)

## 测试配置

- 超时时间: $TEST_TIMEOUT
- 竞态检测: $RACE
- 短测试模式: $SHORT
- 详细输出: $VERBOSE

## 测试结果

EOF
    
    # 添加覆盖率信息（如果存在）
    if [ -f "test/reports/$COVERAGE_FILE" ]; then
        echo "### 覆盖率统计" >> "$report_file"
        echo '```' >> "$report_file"
        go tool cover -func="test/reports/$COVERAGE_FILE" | tail -5 >> "$report_file"
        echo '```' >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # 添加基准测试结果（如果存在）
    if [ -f "test/reports/benchmark.txt" ]; then
        echo "### 基准测试结果" >> "$report_file"
        echo "详见: [benchmark.txt](benchmark.txt)" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    log_success "测试报告生成: $report_file"
}

# 解析命令行参数
parse_args() {
    VERBOSE=${VMAN_TEST_VERBOSE:-false}
    SHORT=${VMAN_TEST_SHORT:-false}
    RACE=${VMAN_TEST_RACE:-false}
    COUNT=""
    PARALLEL=""
    NO_CACHE=false
    FAIL_FAST=false
    TEST_TYPE="all"
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -s|--short)
                SHORT=true
                shift
                ;;
            -r|--race)
                RACE=true
                shift
                ;;
            -c|--count)
                COUNT="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL="$2"
                shift 2
                ;;
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --bench-time)
                BENCHMARK_TIME="$2"
                shift 2
                ;;
            --no-cache)
                NO_CACHE=true
                shift
                ;;
            --fail-fast)
                FAIL_FAST=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            unit|integration|functional|all|bench|coverage|clean)
                TEST_TYPE="$1"
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 主函数
main() {
    parse_args "$@"
    
    log_info "vman 测试脚本启动"
    log_info "测试类型: $TEST_TYPE"
    
    check_dependencies
    setup_test_env
    
    case $TEST_TYPE in
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        functional)
            run_functional_tests
            ;;
        all)
            run_all_tests
            verify_code_quality
            ;;
        bench)
            run_benchmark_tests
            ;;
        coverage)
            generate_coverage
            ;;
        clean)
            clean_test_files
            ;;
        *)
            log_error "未知测试类型: $TEST_TYPE"
            exit 1
            ;;
    esac
    
    # 总是生成测试报告
    if [ "$TEST_TYPE" != "clean" ]; then
        generate_test_report
    fi
    
    log_success "测试脚本执行完成"
}

# 信号处理
trap 'log_warning "测试被中断"; exit 130' INT TERM

# 执行主函数
main "$@"