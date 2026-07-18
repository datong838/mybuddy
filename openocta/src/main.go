// OpenOcta Wails desktop app: starts gateway in-process and loads http://127.0.0.1:18900 in webview.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/appinstance"
	"github.com/openocta/openocta/pkg/appupdate"
	"github.com/openocta/openocta/pkg/desktop"
	gatewayhttp "github.com/openocta/openocta/pkg/gateway/http"
	"github.com/openocta/openocta/pkg/paths"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:wails-bootstrap
var bootstrapFS embed.FS

func writeStartupLog(msg string) {
	env := func(k string) string { return os.Getenv(k) }
	stateDir := paths.ResolveStateDir(env)
	_ = os.MkdirAll(stateDir, 0700)
	logPath := filepath.Join(stateDir, "desktop-startup.log")
	_ = os.WriteFile(logPath, []byte(msg+"\n"), 0600)
	// macOS: 弹窗提示用户查看日志
	if runtime.GOOS == "darwin" {
		esc := escapeForAppleScript(msg)
		_ = exec.Command("osascript", "-e", fmt.Sprintf("display alert \"OpenOcta 启动失败\" message \"%s\" & return & \"详见: %s\" as critical", esc, escapeForAppleScript(logPath))).Run()
	}
}

func escapeForAppleScript(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "\"", "\\\"")
}

func main() {
	// 捕获 panic 并写入日志
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v", r)
			writeStartupLog(msg)
			os.Exit(1)
		}
	}()

	// 从 .dmg 首次启动时提示安装到「应用程序」（macOS）；若已安装并重启则直接退出本实例
	desktop.MaybePromptInstallFromDMG()

	appinstance.KillOtherOpenOctaProcesses()

	// Start gateway in goroutine before creating window
	srv, err := desktop.StartGateway()
	if err != nil {
		msg := fmt.Sprintf("Failed to start gateway: %v", err)
		writeStartupLog(msg)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := desktop.WaitForHealthy(ctx, 20*time.Second); err != nil {
		msg := fmt.Sprintf("Gateway not ready: %v", err)
		writeStartupLog(msg)
		_ = srv.Shutdown(context.Background())
		os.Exit(1)
	}

	// Bootstrap FS root must contain index.html
	assets, err := fs.Sub(bootstrapFS, "wails-bootstrap")
	if err != nil {
		msg := fmt.Sprintf("Failed to load assets: %v", err)
		writeStartupLog(msg)
		_ = srv.Shutdown(context.Background())
		os.Exit(1)
	}

	var wailsAppCtx context.Context
	gatewayhttp.DesktopQuit = func() {
		if wailsAppCtx != nil {
			wailsruntime.Quit(wailsAppCtx)
		}
	}
	appupdate.SetDesktopQuit(gatewayhttp.DesktopQuit)

	appOptions := &options.App{
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			wailsAppCtx = ctx
		},
		OnShutdown: func(ctx context.Context) {
			defer func() { _ = recover() }()
			_ = srv.Shutdown(ctx)
		},
		Title:     "OpenOcta",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
	}
	// Windows 使用系统原生标题栏，确保窗口有最小化/最大化/关闭按钮，可拖动和调整大小
	if runtime.GOOS == "windows" {
		appOptions.Frameless = false
		appOptions.Windows = &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		}
	}
	runErr := wails.Run(appOptions)
	if runErr != nil {
		msg := fmt.Sprintf("Wails error: %v", runErr)
		writeStartupLog(msg)
		_ = srv.Shutdown(context.Background())
		os.Exit(1)
	}
}
