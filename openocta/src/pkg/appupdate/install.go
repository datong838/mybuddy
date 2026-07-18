package appupdate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type InstallProgress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

type installJob struct {
	mu         sync.Mutex
	running    bool
	done       bool
	err        string
	manualHint string
	progress   InstallProgress
	cancel     context.CancelFunc
}

var globalInstallJob installJob

func setInstallProgress(report func(InstallProgress), phase string, percent int, message string) {
	p := InstallProgress{Phase: phase, Percent: percent, Message: message}
	if report != nil {
		report(p)
	}
	globalInstallJob.mu.Lock()
	globalInstallJob.progress = p
	globalInstallJob.mu.Unlock()
}

// InstallStatus reports in-flight package install state.
func InstallStatus() map[string]interface{} {
	out := map[string]interface{}{
		"ok": true,
	}
	globalInstallJob.mu.Lock()
	defer globalInstallJob.mu.Unlock()
	out["installing"] = globalInstallJob.running
	out["installDone"] = globalInstallJob.done
	if globalInstallJob.err != "" {
		out["installError"] = globalInstallJob.err
	}
	if globalInstallJob.manualHint != "" {
		out["manualInstallHint"] = globalInstallJob.manualHint
	}
	if globalInstallJob.progress.Message != "" {
		out["progress"] = globalInstallJob.progress
	}
	return out
}

// StartInstallAsync downloads and installs the latest package for this platform.
func StartInstallAsync(latestVersion string) (started bool, err error) {
	if !InstallAllowed() {
		return false, fmt.Errorf("当前运行模式不支持自动安装")
	}
	target := ResolvePlatformTarget()
	if !target.AutoInstallSupported || target.AutoInstallURL == "" {
		if target.ManualInstallHint != "" {
			return false, fmt.Errorf("当前环境不支持自动安装，请参考手动命令\n\n%s", target.ManualInstallHint)
		}
		return false, fmt.Errorf("当前操作系统/架构暂不支持自动安装")
	}
	url := target.AutoInstallURL

	globalInstallJob.mu.Lock()
	if globalInstallJob.running {
		globalInstallJob.mu.Unlock()
		return false, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	globalInstallJob.running = true
	globalInstallJob.done = false
	globalInstallJob.err = ""
	globalInstallJob.manualHint = target.ManualInstallHint
	globalInstallJob.cancel = cancel
	globalInstallJob.progress = InstallProgress{Phase: "queued", Percent: 0, Message: "等待开始…"}
	globalInstallJob.mu.Unlock()

	go func() {
		installErr := runInstall(ctx, url, latestVersion, func(p InstallProgress) {
			globalInstallJob.mu.Lock()
			globalInstallJob.progress = p
			globalInstallJob.mu.Unlock()
		})
		globalInstallJob.mu.Lock()
		globalInstallJob.running = false
		globalInstallJob.done = true
		globalInstallJob.cancel = nil
		if installErr != nil {
			if errors.Is(installErr, context.Canceled) {
				globalInstallJob.err = ""
				globalInstallJob.progress = InstallProgress{Phase: "cancelled", Percent: 0, Message: "安装已取消"}
			} else {
				globalInstallJob.err = installErr.Error()
				if hint := manualHintFromError(installErr); hint != "" {
					globalInstallJob.manualHint = hint
				}
				globalInstallJob.progress = InstallProgress{
					Phase:   "error",
					Percent: globalInstallJob.progress.Percent,
					Message: installErr.Error(),
				}
			}
		}
		globalInstallJob.mu.Unlock()
	}()
	return true, nil
}

func runInstall(ctx context.Context, url, latestVersion string, report func(InstallProgress)) error {
	setInstallProgress(report, "download", 5, "正在下载更新包…")
	path, err := downloadPackage(ctx, url, latestVersion, report)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(path)
		_ = os.RemoveAll(filepath.Dir(path))
	}()

	setInstallProgress(report, "install", 92, "正在安装更新…")
	if err := installPackage(ctx, path, report); err != nil {
		return err
	}
	setInstallProgress(report, "done", 100, "安装完成，应用即将重启…")
	return nil
}

func downloadPackage(ctx context.Context, url, latestVersion string, report func(InstallProgress)) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("download HTTP %d", res.StatusCode)
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".pkg"
	}
	dir, err := os.MkdirTemp("", "openocta-update-*")
	if err != nil {
		return "", err
	}
	baseName := filepath.Base(strings.Split(url, "?")[0])
	if baseName == "" || baseName == "." || baseName == "/" {
		baseName = "OpenOcta-" + sanitizeFilename(latestVersion) + ext
	}
	dest := filepath.Join(dir, baseName)

	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	total := res.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastPct := 5
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		n, readErr := res.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				return "", wErr
			}
			downloaded += int64(n)
			if total > 0 {
				pct := 5 + int(float64(downloaded)/float64(total)*85)
				if pct > 90 {
					pct = 90
				}
				if pct >= lastPct+2 {
					lastPct = pct
					setInstallProgress(report, "download", pct, "正在下载更新包…")
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return "", readErr
		}
	}
	return dest, nil
}

func sanitizeFilename(v string) string {
	v = strings.TrimSpace(v)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(v)
}

func installPackage(ctx context.Context, path string, report func(InstallProgress)) error {
	return installPackageForPlatform(ctx, path, report)
}

func manualHintFromError(err error) string {
	if err == nil {
		return ""
	}
	target := ResolvePlatformTarget()
	return target.ManualInstallHint
}
