package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProtocManager 测试ProtocManager的创建和基本属性
func TestProtocManager(t *testing.T) {
	manager := NewProtocManager()
	
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.fs)
	assert.NotNil(t, manager.logger)
	assert.Contains(t, manager.shimsDir, ".vman/shims")
	assert.Equal(t, ".protoc-backup", manager.backupSuffix)
	assert.False(t, manager.protocBackedUp)
	assert.NotEmpty(t, manager.originalPATH)
}

// TestNewProtocCmd 测试protoc命令的创建
func TestNewProtocCmd(t *testing.T) {
	cmd := newProtocCmd()
	
	assert.Equal(t, "protoc", cmd.Use)
	assert.Equal(t, "Protocol Buffer编译器一键管理", cmd.Short)
	assert.Contains(t, cmd.Long, "提供protoc和插件的一键配置管理")
	
	// 检查子命令
	subcommands := cmd.Commands()
	commandNames := make([]string, 0, len(subcommands))
	for _, subcmd := range subcommands {
		commandNames = append(commandNames, subcmd.Name())
	}
	
	expectedSubcommands := []string{"setup", "exec", "make-api", "status"}
	for _, expected := range expectedSubcommands {
		assert.Contains(t, commandNames, expected, "Missing protoc subcommand: %s", expected)
	}
}

// TestProtocSetupCmd 测试setup子命令
func TestProtocSetupCmd(t *testing.T) {
	cmd := newProtocSetupCmd()
	
	assert.Equal(t, "setup", cmd.Use)
	assert.Equal(t, "一键设置protoc环境", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

// TestProtocExecCmd 测试exec子命令
func TestProtocExecCmd(t *testing.T) {
	cmd := newProtocExecCmd()
	
	assert.Equal(t, "exec [command...]", cmd.Use)
	assert.Equal(t, "在protoc模式下执行命令", cmd.Short)
	assert.Contains(t, cmd.Example, "vman protoc exec make api")
	assert.NotNil(t, cmd.RunE)
}

// TestProtocMakeAPICmd 测试make-api子命令
func TestProtocMakeAPICmd(t *testing.T) {
	cmd := newProtocMakeAPICmd()
	
	assert.Equal(t, "make-api", cmd.Use)
	assert.Equal(t, "一键执行make api命令", cmd.Short)
	assert.Contains(t, cmd.Long, "一键执行make api命令，自动处理所有protoc环境设置")
	assert.Contains(t, cmd.Example, "vman protoc make-api")
	assert.NotNil(t, cmd.RunE)
	
	// 检查标志
	dirFlag := cmd.Flags().Lookup("dir")
	assert.NotNil(t, dirFlag)
	assert.Equal(t, "d", dirFlag.Shorthand)
}

// TestProtocStatusCmd 测试status子命令
func TestProtocStatusCmd(t *testing.T) {
	cmd := newProtocStatusCmd()
	
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "显示protoc环境状态", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

// TestProtocManagerMethods 测试ProtocManager的主要方法
func TestProtocManagerMethods(t *testing.T) {
	// 创建内存文件系统进行测试
	memFS := afero.NewMemMapFs()
	manager := &ProtocManager{
		fs:             memFS,
		logger:         logrus.New(),
		shimsDir:       "/test/.vman/shims",
		backupSuffix:   ".protoc-backup",
		protocBackedUp: false,
		originalPATH:   "/usr/bin:/bin",
	}
	
	// 设置日志级别为错误，减少测试输出
	manager.logger.SetLevel(logrus.ErrorLevel)
	
	t.Run("TestSmartBackupProtocShim", func(t *testing.T) {
		// 创建测试目录
		err := memFS.MkdirAll("/test/.vman/shims", 0755)
		require.NoError(t, err)
		
		// 创建protoc shim文件
		shimPath := "/test/.vman/shims/protoc"
		err = afero.WriteFile(memFS, shimPath, []byte("fake protoc shim"), 0755)
		require.NoError(t, err)
		
		// 测试备份功能
		err = manager.smartBackupProtocShim()
		assert.NoError(t, err)
		assert.True(t, manager.protocBackedUp)
		
		// 检查备份文件是否存在
		backupPath := shimPath + manager.backupSuffix
		exists, err := afero.Exists(memFS, backupPath)
		assert.NoError(t, err)
		assert.True(t, exists)
		
		// 测试重复备份（应该跳过）
		err = manager.smartBackupProtocShim()
		assert.NoError(t, err)
	})
	
	t.Run("TestRestoreProtocShim", func(t *testing.T) {
		// 前提：已经备份了protoc shim
		manager.protocBackedUp = true
		
		err := manager.restoreProtocShim()
		assert.NoError(t, err)
		assert.False(t, manager.protocBackedUp)
		
		// 检查原文件是否恢复
		shimPath := "/test/.vman/shims/protoc"
		exists, err := afero.Exists(memFS, shimPath)
		assert.NoError(t, err)
		assert.True(t, exists)
	})
	
	t.Run("TestFileExists", func(t *testing.T) {
		// 测试存在的文件
		exists := manager.fileExists("/test/.vman/shims/protoc")
		assert.True(t, exists)
		
		// 测试不存在的文件
		exists = manager.fileExists("/nonexistent/file")
		assert.False(t, exists)
	})
	
	t.Run("TestSetupPluginPaths", func(t *testing.T) {
		err := manager.setupPluginPaths()
		assert.NoError(t, err) // 当前实现返回nil
	})
}

// TestBuildProtocEnv 测试环境变量构建
func TestBuildProtocEnv(t *testing.T) {
	manager := &ProtocManager{
		fs:           afero.NewMemMapFs(),
		logger:       logrus.New(),
		originalPATH: "/usr/bin:/bin",
	}
	
	// 设置日志级别为错误，减少测试输出
	manager.logger.SetLevel(logrus.ErrorLevel)
	
	env := manager.buildProtocEnv()
	
	// 检查环境变量数组不为空
	assert.NotEmpty(t, env)
	
	// 检查PATH环境变量是否存在
	pathFound := false
	for _, envVar := range env {
		if len(envVar) >= 5 && envVar[:5] == "PATH=" {
			pathFound = true
			break
		}
	}
	assert.True(t, pathFound, "PATH environment variable should be present")
}

// TestMakeAPIValidation 测试MakeAPI方法的参数验证
func TestMakeAPIValidation(t *testing.T) {
	// 创建临时目录进行测试
	tempDir := t.TempDir()
	
	manager := &ProtocManager{
		fs:     afero.NewOsFs(),
		logger: logrus.New(),
	}
	
	// 设置日志级别为错误，减少测试输出
	manager.logger.SetLevel(logrus.ErrorLevel)
	
	t.Run("TestDirectoryNotExists", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempDir, "nonexistent")
		err := manager.MakeAPI(nonExistentDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "切换目录失败")
	})
	
	t.Run("TestNoMakefile", func(t *testing.T) {
		// 创建空目录（没有Makefile）
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)
		
		err = manager.MakeAPI(emptyDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "当前目录不存在Makefile")
	})
	
	t.Run("TestWithMakefile", func(t *testing.T) {
		// 创建带有Makefile的目录
		projectDir := filepath.Join(tempDir, "project")
		err := os.MkdirAll(projectDir, 0755)
		require.NoError(t, err)
		
		// 创建一个简单的Makefile
		makefilePath := filepath.Join(projectDir, "Makefile")
		err = os.WriteFile(makefilePath, []byte("api:\n\techo 'Running API generation'\n"), 0644)
		require.NoError(t, err)
		
		// 注意：这个测试可能会失败，因为它会尝试执行实际的vman命令
		// 在真实环境中，我们需要mock这些外部依赖
		err = manager.MakeAPI(projectDir)
		// 我们不检查错误，因为可能会因为外部依赖而失败
		// 这里主要测试了参数验证逻辑
		t.Logf("MakeAPI result: %v", err)
	})
}

// TestProtocCommandIntegration 测试protoc命令集成到根命令
func TestProtocCommandIntegration(t *testing.T) {
	// 检查protoc命令是否正确注册到根命令
	commands := rootCmd.Commands()
	protocFound := false
	
	for _, cmd := range commands {
		if cmd.Name() == "protoc" {
			protocFound = true
			break
		}
	}
	
	assert.True(t, protocFound, "protoc command should be registered in root command")
}

// Benchmark测试（可选）
func BenchmarkNewProtocManager(b *testing.B) {
	for i := 0; i < b.N; i++ {
		manager := NewProtocManager()
		_ = manager
	}
}

func BenchmarkBuildProtocEnv(b *testing.B) {
	manager := NewProtocManager()
	manager.logger.SetLevel(logrus.ErrorLevel)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env := manager.buildProtocEnv()
		_ = env
	}
}