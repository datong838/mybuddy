//go:build windows

package appupdate

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func installPackageForPlatform(ctx context.Context, path string, report func(InstallProgress)) error {
	setInstallProgress(report, "install", 95, "正在运行安装程序…")
	cmd := exec.CommandContext(ctx, path, "/S")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装程序失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	setInstallProgress(report, "install", 98, "安装完成，应用即将退出…")
	scheduleDesktopQuit()
	return nil
}

var desktopQuit func()

// SetDesktopQuit registers a callback to exit the desktop host after install.
func SetDesktopQuit(fn func()) {
	desktopQuit = fn
}

func scheduleDesktopQuit() {
	if desktopQuit == nil {
		return
	}
	go func() {
		time.Sleep(1200 * time.Millisecond)
		desktopQuit()
	}()
}
