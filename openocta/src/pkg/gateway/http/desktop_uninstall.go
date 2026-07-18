package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/paths"
)

// DesktopQuit 由 Wails 主程序注入：在已安排延迟卸载后退出桌面宿主进程，以便清理脚本能够删到 .app 等文件。
// 仅在 OPENOCTA_RUN_MODE=desktop 时由 handleDesktopUninstall 调用。
var DesktopQuit func()

type uninstallRequestBody struct {
	Mode string `json:"mode"` // "program" | "full"
}

type uninstallResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// handleDesktopUninstallOptions handles CORS preflight for POST /api/desktop/uninstall.
func (s *Server) handleDesktopUninstallOptions(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization, X-Gateway-Token")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// handleDesktopUninstall removes user data and/or schedules removal of the install bundle/binary after exit.
// Requires gateway token. Allowed when OPENOCTA_RUN_MODE=desktop or OPENOCTA_ALLOW_UNINSTALL=1.
func (s *Server) handleDesktopUninstall(w http.ResponseWriter, r *http.Request) {
	setSiteProxyCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "仅支持 POST"})
		return
	}

	if !uninstallAllowed() {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(uninstallResponse{
			OK:      false,
			Message: "当前环境不允许通过 API 卸载",
			Detail:  "请在桌面应用中操作，或设置环境变量 OPENOCTA_ALLOW_UNINSTALL=1（仅限本机可信环境）",
		})
		return
	}

	var body uninstallRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "请求体无效", Detail: err.Error()})
		return
	}
	mode := strings.ToLower(strings.TrimSpace(body.Mode))
	if mode != "program" && mode != "full" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "mode 须为 program 或 full"})
		return
	}

	env := func(k string) string { return os.Getenv(k) }
	stateDir := filepath.Clean(paths.ResolveStateDir(env))

	exe, err := os.Executable()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "无法解析可执行文件路径", Detail: err.Error()})
		return
	}
	exe, _ = filepath.EvalSymlinks(exe)
	exe = filepath.Clean(exe)

	installRoot, rootKind := resolveInstallRoot(exe)
	var toRemove []string
	if mode == "full" {
		if stateDir != "" && stateDir != "." && stateDir != "/" {
			toRemove = append(toRemove, stateDir)
		}
	}
	if installRoot != "" {
		toRemove = append(toRemove, installRoot)
	} else {
		// 非 .app / 非标准安装目录时，至少尝试删除当前二进制
		if exe != "" && fileExists(exe) {
			toRemove = append(toRemove, exe)
		}
	}

	if len(toRemove) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "没有可删除的安装路径"})
		return
	}

	if err := scheduleDeferredRemoval(toRemove); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(uninstallResponse{OK: false, Message: "无法启动卸载任务", Detail: err.Error()})
		return
	}

	msg := "已安排卸载任务，数秒后删除选定内容。"
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")), "desktop") {
		msg += "桌面应用将自动退出。"
	} else {
		msg += "请结束本机上的 OpenOcta / 网关进程后再进行其他操作。"
	}
	if rootKind != "" {
		msg += fmt.Sprintf("（安装位置：%s）", rootKind)
	}
	_ = json.NewEncoder(w).Encode(uninstallResponse{OK: true, Message: msg})
	rc := http.NewResponseController(w)
	_ = rc.Flush()
	if DesktopQuit != nil && strings.EqualFold(strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")), "desktop") {
		go func() {
			time.Sleep(300 * time.Millisecond)
			DesktopQuit()
		}()
	}
}

func uninstallAllowed() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")), "desktop") {
		return true
	}
	if isTruthyEnv(func(k string) string { return os.Getenv(k) }, "OPENOCTA_ALLOW_UNINSTALL") {
		return true
	}
	return false
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// resolveInstallRoot returns a path to remove for the "program" part and a short description.
func resolveInstallRoot(exe string) (path string, kind string) {
	switch runtime.GOOS {
	case "darwin":
		// .../OpenOcta.app/Contents/MacOS/OpenOcta -> .../OpenOcta.app
		const marker = ".app/Contents/MacOS"
		if i := strings.Index(exe, marker); i > 0 {
			return filepath.Clean(exe[:i+4]), "macOS 应用程序包"
		}
		return exe, "可执行文件"
	case "windows":
		dir := filepath.Dir(exe)
		base := strings.ToLower(filepath.Base(exe))
		if base == "openocta.exe" {
			// 仅当 exe 直接位于名为 OpenOcta 的文件夹内时，才删除整个安装目录，避免误删上级目录（如单独放在某盘根目录）。
			if strings.EqualFold(filepath.Base(dir), "OpenOcta") {
				return dir, "Windows 安装目录"
			}
		}
		return exe, "可执行文件"
	default:
		// Linux 等：单文件安装
		return exe, "可执行文件"
	}
}

func scheduleDeferredRemoval(paths []string) error {
	// 去重、清洗
	seen := map[string]struct{}{}
	var uniq []string
	for _, p := range paths {
		p = filepath.Clean(p)
		if p == "" || p == "." || p == "/" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		uniq = append(uniq, p)
	}
	if len(uniq) == 0 {
		return fmt.Errorf("empty paths")
	}

	if runtime.GOOS == "windows" {
		return scheduleRemovalWindows(uniq)
	}
	return scheduleRemovalUnix(uniq)
}

func scheduleRemovalUnix(paths []string) error {
	var b strings.Builder
	b.WriteString("sleep 4; ")
	for _, p := range paths {
		b.WriteString(fmt.Sprintf("rm -rf %q; ", p))
	}
	cmd := exec.Command("/bin/sh", "-c", b.String())
	detachUninstallCmd(cmd)
	return cmd.Start()
}

func scheduleRemovalWindows(paths []string) error {
	// powershell -WindowStyle Hidden -Command "Start-Sleep -Seconds 4; Remove-Item -LiteralPath '...' -Recurse -Force -ErrorAction SilentlyContinue"
	var ps strings.Builder
	ps.WriteString("Start-Sleep -Seconds 4; ")
	for _, p := range paths {
		pp := strings.ReplaceAll(p, "'", "''")
		ps.WriteString(fmt.Sprintf("Remove-Item -LiteralPath '%s' -Recurse -Force -ErrorAction SilentlyContinue; ", pp))
	}
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps.String())
	detachUninstallCmd(cmd)
	return cmd.Start()
}
