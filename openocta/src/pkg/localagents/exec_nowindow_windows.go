//go:build windows

package localagents

import (
	"os/exec"
	"syscall"
)

func applyExecNoWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
