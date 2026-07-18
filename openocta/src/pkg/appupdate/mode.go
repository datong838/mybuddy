package appupdate

import (
	"os"
	"runtime"
	"strings"
)

// InstallAllowed reports whether this gateway process may run package install.
func InstallAllowed() bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")))
	switch mode {
	case "desktop":
		return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
	case "service":
		return runtime.GOOS == "linux"
	default:
		return false
	}
}

// DesktopMode reports whether the gateway runs inside the Wails desktop shell.
func DesktopMode() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("OPENOCTA_RUN_MODE")), "desktop")
}
