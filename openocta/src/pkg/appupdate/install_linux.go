//go:build linux

package appupdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func installPackageForPlatform(ctx context.Context, path string, report func(InstallProgress)) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".deb":
		return installLinuxDeb(ctx, path, report)
	case ".rpm":
		return installLinuxRpm(ctx, path, report)
	default:
		return manualInstallError("当前环境仅支持 deb/rpm 自动安装")
	}
}

func installLinuxDeb(ctx context.Context, path string, report func(InstallProgress)) error {
	setInstallProgress(report, "install", 93, "正在停止 openocta 服务…")
	_ = runSudo(ctx, "systemctl", "stop", "openocta")

	setInstallProgress(report, "install", 95, "正在安装 deb 包（可能需要管理员权限）…")
	if err := runSudo(ctx, "dpkg", "-i", path); err != nil {
		return manualInstallError(fmt.Sprintf("deb 安装失败: %v", err))
	}

	setInstallProgress(report, "install", 98, "正在启动 openocta 服务…")
	_ = runSudo(ctx, "systemctl", "daemon-reload")
	_ = runSudo(ctx, "systemctl", "enable", "openocta")
	if err := runSudo(ctx, "systemctl", "start", "openocta"); err != nil {
		return manualInstallError(fmt.Sprintf("服务启动失败: %v", err))
	}
	scheduleServiceExit()
	return nil
}

func installLinuxRpm(ctx context.Context, path string, report func(InstallProgress)) error {
	setInstallProgress(report, "install", 93, "正在停止 openocta 服务…")
	_ = runSudo(ctx, "systemctl", "stop", "openocta")

	setInstallProgress(report, "install", 95, "正在安装 rpm 包（可能需要管理员权限）…")
	if err := runSudo(ctx, "rpm", "-Uvh", path); err != nil {
		return manualInstallError(fmt.Sprintf("rpm 安装失败: %v", err))
	}

	setInstallProgress(report, "install", 98, "正在启动 openocta 服务…")
	_ = runSudo(ctx, "systemctl", "daemon-reload")
	_ = runSudo(ctx, "systemctl", "enable", "openocta")
	if err := runSudo(ctx, "systemctl", "start", "openocta"); err != nil {
		return manualInstallError(fmt.Sprintf("服务启动失败: %v", err))
	}
	scheduleServiceExit()
	return nil
}

func runSudo(ctx context.Context, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("未找到 sudo: %w", err)
	}
	cmdArgs := append([]string{"-n"}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	// 非交互环境无法输入密码时，提示用户改用手动命令
	if strings.Contains(string(out), "password") || strings.Contains(err.Error(), "password") {
		return fmt.Errorf("需要 sudo 密码")
	}
	return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
}

type manualInstallErr struct {
	msg string
}

func (e *manualInstallErr) Error() string { return e.msg }

func manualInstallError(msg string) error {
	target := ResolvePlatformTarget()
	full := strings.TrimSpace(msg)
	if target.ManualInstallHint != "" {
		full = full + "\n\n" + target.ManualInstallHint
	}
	return &manualInstallErr{msg: full}
}

func scheduleServiceExit() {
	go func() {
		time.Sleep(800 * time.Millisecond)
		os.Exit(0)
	}()
}

// SetDesktopQuit is unused on Linux service installs.
func SetDesktopQuit(fn func()) {}
