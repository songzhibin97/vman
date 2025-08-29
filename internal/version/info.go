package version

import (
	"fmt"
	"runtime"
)

// 版本信息变量，在编译时由ldflags设置
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// Info 版本信息结构
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion 获取版本信息
func GetVersion() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersionString 获取版本字符串
func GetVersionString() string {
	return fmt.Sprintf("vman %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}

// GetFullVersionString 获取完整版本信息字符串
func GetFullVersionString() string {
	return fmt.Sprintf(`vman version info:
  Version:    %s
  Commit:     %s
  Build Time: %s
  Go Version: %s
  Platform:   %s/%s`,
		Version,
		Commit,
		BuildTime,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}
