//go:build darwin

package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MaybePromptInstallFromDMG 在从磁盘映像（.dmg）内启动时提示用户是否安装到「应用程序」；
// 若用户确认，则复制 .app 到 /Applications 并启动新实例后退出当前进程。
func MaybePromptInstallFromDMG() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return
	}
	exe = filepath.Clean(exe)

	bundle := macOSAppBundlePath(exe)
	if bundle == "" {
		return
	}

	// 仅当 .app 位于已挂载的卷（通常为 .dmg）上时提示
	if !strings.HasPrefix(bundle, "/Volumes/") {
		return
	}

	bundle = filepath.Clean(bundle)

	if !installChoice() {
		return
	}

	dst := "/Applications/OpenOcta.app"
	if err := dittoWithAdmin(bundle, dst); err != nil {
		_ = displayAlert(
			"安装失败",
			fmt.Sprintf("无法复制到「应用程序」：%s\n\n请尝试将窗口中的 OpenOcta 图标拖入右侧「应用程序」文件夹。", err),
		)
		return
	}

	_ = exec.Command("/usr/bin/open", "-a", dst).Run()
	os.Exit(0)
}

func macOSAppBundlePath(exe string) string {
	dir := filepath.Dir(exe)
	for d := dir; d != "/" && d != "."; d = filepath.Dir(d) {
		base := filepath.Base(d)
		if strings.HasSuffix(strings.ToLower(base), ".app") {
			return d
		}
	}
	return ""
}

func installChoice() bool {
	script := `
try
	set r to button returned of (display dialog "OpenOcta 正从磁盘映像运行。" & return & return & "是否将 OpenOcta 安装到「应用程序」文件夹？" & return & return & "（安装后可在启动台或「应用程序」中打开。）" buttons {"稍后", "安装"} default button 2 with title "安装 OpenOcta" with icon note)
	if r is "安装" then
		"yes"
	else
		"no"
	end if
on error number -128
	"no"
end try
`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "yes"
}

func dittoWithAdmin(src, dst string) error {
	// AppleScript 字符串中的双引号需转义
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

func displayAlert(title, message string) error {
	t := appleScriptString(title)
	m := appleScriptString(message)
	script := fmt.Sprintf(`display alert %s message %s as critical`, t, m)
	return exec.Command("osascript", "-e", script).Run()
}
