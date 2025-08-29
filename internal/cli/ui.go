package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/songzhibin97/vman/pkg/types"
)

// 颜色定义
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
	ColorDim    = "\033[2m"
)

// Emoji 定义
const (
	EmojiCheckMark = "✅"
	EmojiCrossMark = "❌"
	EmojiWarning   = "⚠️"
	EmojiInfo      = "ℹ️"
	EmojiTool      = "🔧"
	EmojiPackage   = "📦"
	EmojiDownload  = "⬇️"
	EmojiUpload    = "⬆️"
	EmojiFolder    = "📁"
	EmojiClock     = "🕒"
	EmojiRocket    = "🚀"
	EmojiSparkles  = "✨"
)

// UIOptions UI选项
type UIOptions struct {
	NoColor     bool
	NoEmoji     bool
	Verbose     bool
	Interactive bool
}

// ColorSupport 检查终端是否支持颜色
func ColorSupport() bool {
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")

	// 检查是否明确禁用颜色
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return false
	}

	// 检查是否明确启用颜色
	if colorTerm != "" || os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}

	// 检查常见的支持颜色的终端
	supportedTerms := []string{"xterm", "xterm-256color", "screen", "tmux"}
	for _, supportedTerm := range supportedTerms {
		if strings.Contains(term, supportedTerm) {
			return true
		}
	}

	return false
}

// Colorize 为文本添加颜色
func Colorize(text, color string, options *UIOptions) string {
	if options != nil && options.NoColor || !ColorSupport() {
		return text
	}
	return color + text + ColorReset
}

// ColorizeSuccess 成功信息颜色
func ColorizeSuccess(text string, options *UIOptions) string {
	return Colorize(text, ColorGreen, options)
}

// ColorizeError 错误信息颜色
func ColorizeError(text string, options *UIOptions) string {
	return Colorize(text, ColorRed, options)
}

// ColorizeWarning 警告信息颜色
func ColorizeWarning(text string, options *UIOptions) string {
	return Colorize(text, ColorYellow, options)
}

// ColorizeInfo 信息颜色
func ColorizeInfo(text string, options *UIOptions) string {
	return Colorize(text, ColorBlue, options)
}

// ColorizeBold 加粗文本
func ColorizeBold(text string, options *UIOptions) string {
	return Colorize(text, ColorBold, options)
}

// Emoji 获取emoji，支持禁用
func Emoji(emoji string, options *UIOptions) string {
	if options != nil && options.NoEmoji {
		return ""
	}
	return emoji + " "
}

// PrintSuccess 打印成功消息
func PrintSuccess(message string, options *UIOptions) {
	emoji := Emoji(EmojiCheckMark, options)
	colored := ColorizeSuccess(message, options)
	fmt.Printf("%s%s\n", emoji, colored)
}

// PrintError 打印错误消息
func PrintError(message string, options *UIOptions) {
	emoji := Emoji(EmojiCrossMark, options)
	colored := ColorizeError(message, options)
	fmt.Printf("%s%s\n", emoji, colored)
}

// PrintWarning 打印警告消息
func PrintWarning(message string, options *UIOptions) {
	emoji := Emoji(EmojiWarning, options)
	colored := ColorizeWarning(message, options)
	fmt.Printf("%s%s\n", emoji, colored)
}

// PrintInfo 打印信息消息
func PrintInfo(message string, options *UIOptions) {
	emoji := Emoji(EmojiInfo, options)
	colored := ColorizeInfo(message, options)
	fmt.Printf("%s%s\n", emoji, colored)
}

// ProgressBar 进度条结构
type ProgressBar struct {
	total     int64
	current   int64
	width     int
	prefix    string
	suffix    string
	showBytes bool
	showETA   bool
	startTime time.Time
	options   *UIOptions
}

// NewProgressBar 创建新的进度条
func NewProgressBar(total int64, options *UIOptions) *ProgressBar {
	return &ProgressBar{
		total:     total,
		width:     50,
		showBytes: true,
		showETA:   true,
		startTime: time.Now(),
		options:   options,
	}
}

// SetPrefix 设置前缀
func (pb *ProgressBar) SetPrefix(prefix string) *ProgressBar {
	pb.prefix = prefix
	return pb
}

// SetSuffix 设置后缀
func (pb *ProgressBar) SetSuffix(suffix string) *ProgressBar {
	pb.suffix = suffix
	return pb
}

// Update 更新进度
func (pb *ProgressBar) Update(current int64) {
	pb.current = current
	pb.Render()
}

// Render 渲染进度条
func (pb *ProgressBar) Render() {
	percentage := float64(pb.current) / float64(pb.total) * 100
	filled := int(float64(pb.width) * float64(pb.current) / float64(pb.total))

	// 构建进度条
	bar := "["
	for i := 0; i < pb.width; i++ {
		if i < filled {
			bar += "="
		} else if i == filled {
			bar += ">"
		} else {
			bar += " "
		}
	}
	bar += "]"

	// 添加颜色
	if filled > pb.width*3/4 {
		bar = Colorize(bar, ColorGreen, pb.options)
	} else if filled > pb.width/2 {
		bar = Colorize(bar, ColorYellow, pb.options)
	} else {
		bar = Colorize(bar, ColorRed, pb.options)
	}

	// 构建完整的进度信息
	info := fmt.Sprintf("%s %s %.1f%%", pb.prefix, bar, percentage)

	if pb.showBytes && pb.total > 0 {
		info += fmt.Sprintf(" (%s/%s)", formatBytesUI(pb.current), formatBytesUI(pb.total))
	}

	if pb.showETA && pb.current > 0 {
		elapsed := time.Since(pb.startTime)
		estimated := time.Duration(float64(elapsed) * float64(pb.total) / float64(pb.current))
		remaining := estimated - elapsed
		if remaining > 0 {
			info += fmt.Sprintf(" ETA: %s", formatDuration(remaining))
		}
	}

	if pb.suffix != "" {
		info += " " + pb.suffix
	}

	// 清除当前行并打印新信息
	fmt.Printf("\r%s", strings.Repeat(" ", 100)) // 清除行
	fmt.Printf("\r%s", info)
}

// Finish 完成进度条
func (pb *ProgressBar) Finish() {
	pb.current = pb.total
	pb.Render()
	fmt.Println() // 换行
}

// InteractiveSelect 交互式选择
func InteractiveSelect(prompt string, options []string, defaultIndex int, uiOptions *UIOptions) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("没有可选项")
	}

	emoji := Emoji("🤔", uiOptions)
	fmt.Printf("%s%s\n", emoji, ColorizeBold(prompt, uiOptions))

	for i, option := range options {
		marker := "  "
		if i == defaultIndex {
			marker = Colorize("❯ ", ColorGreen, uiOptions)
			option = ColorizeBold(option, uiOptions)
		} else {
			marker = "  "
		}
		fmt.Printf("%s%d) %s\n", marker, i+1, option)
	}

	fmt.Printf("\n请输入选择 (1-%d，默认: %d): ", len(options), defaultIndex+1)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultIndex, nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("无效输入: %s", input)
	}

	if choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("选择超出范围: %d", choice)
	}

	return choice - 1, nil
}

// ConfirmAction 确认操作
func ConfirmAction(prompt string, defaultYes bool, uiOptions *UIOptions) bool {
	defaultText := "y/N"
	if defaultYes {
		defaultText = "Y/n"
	}

	emoji := Emoji("❓", uiOptions)
	coloredPrompt := ColorizeWarning(prompt, uiOptions)
	fmt.Printf("%s%s (%s): ", emoji, coloredPrompt, defaultText)

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultYes
	}

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "y" || input == "yes"
}

// ShowSpinner 显示旋转加载指示器
type Spinner struct {
	chars   []string
	delay   time.Duration
	message string
	stop    chan bool
	options *UIOptions
}

// NewSpinner 创建新的旋转器
func NewSpinner(message string, options *UIOptions) *Spinner {
	return &Spinner{
		chars:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		delay:   100 * time.Millisecond,
		message: message,
		stop:    make(chan bool),
		options: options,
	}
}

// Start 开始旋转
func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				char := s.chars[i%len(s.chars)]
				colored := Colorize(char, ColorBlue, s.options)
				fmt.Printf("\r%s %s", colored, s.message)
				time.Sleep(s.delay)
				i++
			}
		}
	}()
}

// Stop 停止旋转
func (s *Spinner) Stop() {
	s.stop <- true
	fmt.Printf("\r%s\n", strings.Repeat(" ", len(s.message)+10)) // 清除行
}

// ProgressCallback 进度回调适配器
func ProgressCallback(pb *ProgressBar) func(*types.ProgressInfo) {
	return func(info *types.ProgressInfo) {
		if info.Total > 0 {
			pb.total = info.Total
			pb.Update(info.Downloaded)
		} else {
			// 对于未知大小的下载，显示不确定进度
			fmt.Printf("\r下载中... %s (%s)",
				info.Status,
				formatBytesUI(info.Downloaded))
		}
	}
}

// 辅助函数
func formatBytesUI(bytes int64) string {
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

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// TablePrinter 表格打印器
type TablePrinter struct {
	headers []string
	rows    [][]string
	options *UIOptions
}

// NewTablePrinter 创建新的表格打印器
func NewTablePrinter(headers []string, options *UIOptions) *TablePrinter {
	return &TablePrinter{
		headers: headers,
		options: options,
	}
}

// AddRow 添加行
func (tp *TablePrinter) AddRow(row []string) {
	tp.rows = append(tp.rows, row)
}

// Print 打印表格
func (tp *TablePrinter) Print() {
	if len(tp.headers) == 0 {
		return
	}

	// 计算列宽
	colWidths := make([]int, len(tp.headers))
	for i, header := range tp.headers {
		colWidths[i] = len(header)
	}

	for _, row := range tp.rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// 打印表头
	for i, header := range tp.headers {
		colored := ColorizeBold(header, tp.options)
		fmt.Printf("%-*s", colWidths[i]+2, colored)
	}
	fmt.Println()

	// 打印分割线
	for i := range tp.headers {
		fmt.Printf("%-*s", colWidths[i]+2, strings.Repeat("-", colWidths[i]))
	}
	fmt.Println()

	// 打印行
	for _, row := range tp.rows {
		for i, cell := range row {
			if i < len(colWidths) {
				fmt.Printf("%-*s", colWidths[i]+2, cell)
			}
		}
		fmt.Println()
	}
}

// ShowBanner 显示横幅
func ShowBanner(title, version string, options *UIOptions) {
	banner := fmt.Sprintf(`
%s %s %s
%s
%s
`,
		Emoji(EmojiRocket, options),
		ColorizeBold(title, options),
		ColorizeInfo("v"+version, options),
		ColorizeInfo("通用命令行工具版本管理器", options),
		ColorizeDim("https://github.com/songzhibin97/vman", options),
	)

	fmt.Println(banner)
}

// ColorizeDim 暗色文本
func ColorizeDim(text string, options *UIOptions) string {
	return Colorize(text, ColorDim, options)
}
