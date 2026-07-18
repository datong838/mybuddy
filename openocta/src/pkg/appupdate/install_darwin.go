//go:build darwin

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
	mountPoint, err := attachDMG(ctx, path)
	if err != nil {
		return err
	}
	defer detachDMG(mountPoint)

	appBundle, err := findAppBundle(mountPoint)
	if err != nil {
		return err
	}

	dst := "/Applications/OpenOcta.app"
	setInstallProgress(report, "install", 95, "正在复制到「应用程序」…")
	if err := dittoWithAdmin(appBundle, dst); err != nil {
		return err
	}

	setInstallProgress(report, "install", 98, "正在启动新版本…")
	_ = exec.Command("/usr/bin/open", "-a", dst).Run()
	scheduleDesktopQuit()
	return nil
}

func attachDMG(ctx context.Context, dmgPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", dmgPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("挂载 DMG 失败: %w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "/Volumes/") {
			continue
		}
		fields := strings.Fields(line)
		for i := len(fields) - 1; i >= 0; i-- {
			if strings.HasPrefix(fields[i], "/Volumes/") {
				return fields[i], nil
			}
		}
	}
	return "", fmt.Errorf("无法解析 DMG 挂载点")
}

func detachDMG(mountPoint string) {
	if mountPoint == "" {
		return
	}
	_ = exec.Command("hdiutil", "detach", mountPoint, "-quiet").Run()
}

func findAppBundle(root string) (string, error) {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || !info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".app") {
			found = path
			return filepath.SkipDir
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("在更新包中未找到 OpenOcta.app")
	}
	return found, nil
}

func dittoWithAdmin(src, dst string) error {
	srcAS := appleScriptString(src)
	dstAS := appleScriptString(dst)
	script := fmt.Sprintf(`
set src to %s
set dst to %s
do shell script "/usr/bin/ditto " & quoted form of src & " " & quoted form of dst with administrator privileges
`, srcAS, dstAS)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func appleScriptString(s string) string {
	esc := strings.ReplaceAll(s, `\`, `\\`)
	esc = strings.ReplaceAll(esc, `"`, `\"`)
	return `"` + esc + `"`
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
		time.Sleep(800 * time.Millisecond)
		desktopQuit()
	}()
}
