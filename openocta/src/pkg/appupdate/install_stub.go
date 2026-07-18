//go:build !darwin && !windows && !linux

package appupdate

import (
	"context"
	"fmt"
)

func installPackageForPlatform(ctx context.Context, path string, report func(InstallProgress)) error {
	_ = ctx
	_ = path
	_ = report
	return fmt.Errorf("当前操作系统暂不支持自动安装")
}

// SetDesktopQuit is a no-op on unsupported platforms.
func SetDesktopQuit(fn func()) {}
