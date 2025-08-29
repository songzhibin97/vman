package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/afero"
)

// ProtocManager protoc专用管理器
type ProtocManager struct {
	fs             afero.Fs
	logger         *logrus.Logger
	shimsDir       string
	backupSuffix   string
	protocBackedUp bool
	originalPATH   string
}

// NewProtocManager 创建protoc管理器
func NewProtocManager() *ProtocManager {
	homeDir, _ := os.UserHomeDir()
	return &ProtocManager{
		fs:             afero.NewOsFs(),
		logger:         logrus.New(),
		shimsDir:       filepath.Join(homeDir, ".vman", "shims"),
		backupSuffix:   ".protoc-backup",
		protocBackedUp: false,
		originalPATH:   os.Getenv("PATH"),
	}
}

// newProtocCmd 创建protoc命令
func newProtocCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protoc",
		Short: "Protocol Buffer编译器一键管理",
		Long:  `提供protoc和插件的一键配置管理，自动解决冲突`,
	}

	cmd.AddCommand(newProtocSetupCmd())
	cmd.AddCommand(newProtocExecCmd()) 
	cmd.AddCommand(newProtocMakeAPICmd())
	cmd.AddCommand(newProtocStatusCmd())

	return cmd
}

// newProtocSetupCmd 一键设置
func newProtocSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "一键设置protoc环境",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := NewProtocManager()
			return manager.Setup()
		},
	}
}

// newProtocExecCmd 执行命令
func newProtocExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec [command...]",
		Short: "在protoc模式下执行命令",
		Example: "vman protoc exec make api",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("请指定要执行的命令")
			}
			manager := NewProtocManager()
			return manager.ExecCommand(args)
		},
	}
}

// newProtocMakeAPICmd 一键make api命令
func newProtocMakeAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "make-api",
		Short: "一键执行make api命令",
		Long:  `一键执行make api命令，自动处理所有protoc环境设置`,
		Example: `  # 在当前目录执行make api
  vman protoc make-api
  
  # 指定目录执行
  vman protoc make-api --dir /path/to/project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := NewProtocManager()
			
			// 获取目录参数
			projectDir, _ := cmd.Flags().GetString("dir")
			if projectDir == "" {
				var err error
				projectDir, err = os.Getwd()
				if err != nil {
					return fmt.Errorf("获取当前目录失败: %w", err)
				}
			}
			
			return manager.MakeAPI(projectDir)
		},
	}
	
	// 添加标志
	cmd.Flags().StringP("dir", "d", "", "指定项目目录")
	
	return cmd
}
func newProtocStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "显示protoc环境状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := NewProtocManager()
			return manager.ShowStatus()
		},
	}
}

// Setup 一键设置protoc环境
func (pm *ProtocManager) Setup() error {
	pm.logger.Info("🚀 开始一键设置protoc环境...")

	// 1. 启用vman代理
	if err := pm.enableProxy(); err != nil {
		pm.logger.Warnf("启用代理失败: %v", err)
	}

	// 2. 智能备份并禁用protoc shim
	if err := pm.smartBackupProtocShim(); err != nil {
		return fmt.Errorf("智能备份protoc shim失败: %w", err)
	}

	// 3. 设置插件路径环境变量
	if err := pm.setupPluginPaths(); err != nil {
		return fmt.Errorf("设置插件路径失败: %w", err)
	}

	pm.logger.Info("✅ protoc环境设置完成！")
	pm.logger.Info("💡 现在可以使用: vman protoc exec make api")
	
	return nil
}

// ExecCommand 在protoc模式下执行命令
func (pm *ProtocManager) ExecCommand(args []string) error {
	pm.logger.Infof("📦 执行: %s", strings.Join(args, " "))

	// 临时设置环境
	env := pm.buildProtocEnv()
	
	// 执行命令
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env

	return cmd.Run()
}

// ShowStatus 显示状态
func (pm *ProtocManager) ShowStatus() error {
	fmt.Println("📊 protoc环境状态")
	fmt.Println("==================")

	// 检查工具版本
	tools := []string{"protoc-gen-go", "protoc-gen-go-grpc", "protoc-gen-go-http"}
	for _, tool := range tools {
		if path := pm.getToolPath(tool); path != "" {
			fmt.Printf("✅ %s: %s\n", tool, path)
		} else {
			fmt.Printf("❌ %s: 未找到\n", tool)
		}
	}

	return nil
}

// 辅助方法
func (pm *ProtocManager) enableProxy() error {
	cmd := exec.Command("vman", "proxy", "setup")
	return cmd.Run()
}

func (pm *ProtocManager) smartBackupProtocShim() error {
	shimPath := filepath.Join(pm.shimsDir, "protoc")
	backupPath := shimPath + pm.backupSuffix
	
	// 检查是否已经备份
	if pm.protocBackedUp {
		pm.logger.Debug("protoc shim已经备份，跳过")
		return nil
	}
	
	// 检查shim文件是否存在
	if _, err := pm.fs.Stat(shimPath); err == nil {
		// 检查是否已有备份
		if _, err := pm.fs.Stat(backupPath); err == nil {
			pm.logger.Debug("已存在protoc shim备份文件")
		} else {
			// 创建备份
			if err := pm.fs.Rename(shimPath, backupPath); err != nil {
				return fmt.Errorf("备份 protoc shim 失败: %w", err)
			}
			pm.logger.Debug("已备份 protoc shim")
		}
		pm.protocBackedUp = true
	}
	return nil
}

func (pm *ProtocManager) restoreProtocShim() error {
	if !pm.protocBackedUp {
		return nil
	}
	
	shimPath := filepath.Join(pm.shimsDir, "protoc")
	backupPath := shimPath + pm.backupSuffix
	
	if _, err := pm.fs.Stat(backupPath); err == nil {
		if err := pm.fs.Rename(backupPath, shimPath); err != nil {
			return fmt.Errorf("恢复 protoc shim 失败: %w", err)
		}
		pm.protocBackedUp = false
		pm.logger.Debug("已恢复 protoc shim")
	}
	return nil
}

func (pm *ProtocManager) setupPluginPaths() error {
	// 这里会在ExecCommand中动态设置
	return nil
}

func (pm *ProtocManager) buildProtocEnv() []string {
	env := os.Environ()
	
	// 获取vman工具路径
	paths := []string{pm.getToolBinDir("protoc-gen-go")}
	paths = append(paths, pm.getToolBinDir("protoc-gen-go-grpc"))
	paths = append(paths, pm.getToolBinDir("protoc-gen-go-http"))
	
	// 过滤空路径
	var validPaths []string
	for _, p := range paths {
		if p != "" {
			validPaths = append(validPaths, p)
		}
	}
	
	// 构建新的PATH
	if len(validPaths) > 0 {
		currentPath := os.Getenv("PATH")
		newPath := strings.Join(validPaths, ":") + ":" + currentPath
		
		// 更新PATH环境变量
		for i, envVar := range env {
			if strings.HasPrefix(envVar, "PATH=") {
				env[i] = "PATH=" + newPath
				break
			}
		}
	}
	
	return env
}

func (pm *ProtocManager) getToolPath(tool string) string {
	cmd := exec.Command("vman", "which", tool)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// MakeAPI 一键执行make api
func (pm *ProtocManager) MakeAPI(projectDir string) error {
	pm.logger.Infof("🚀 在目录 %s 中执行make api...", projectDir)
	
	// 保存当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}
	defer os.Chdir(currentDir)
	
	// 切换到目标目录
	if err := os.Chdir(projectDir); err != nil {
		return fmt.Errorf("切换目录失败: %w", err)
	}
	
	// 检查Makefile是否存在
	if !pm.fileExists("Makefile") {
		return fmt.Errorf("当前目录不存在Makefile")
	}
	
	// 一键设置环境
	if err := pm.Setup(); err != nil {
		return fmt.Errorf("设置环境失败: %w", err)
	}
	
	// 执行make api
	pm.logger.Info("🛠️ 正在执行make api...")
	return pm.ExecCommand([]string{"make", "api"})
}

func (pm *ProtocManager) fileExists(filename string) bool {
	_, err := pm.fs.Stat(filename)
	return err == nil
}

// getToolBinDir 获取工具的bin目录
func (pm *ProtocManager) getToolBinDir(tool string) string {
	toolPath := pm.getToolPath(tool)
	if toolPath == "" {
		return ""
	}
	return filepath.Dir(toolPath)
}

// init 注册protoc命令到根命令
func init() {
	// 添加protoc命令到根命令
	rootCmd.AddCommand(newProtocCmd())
}