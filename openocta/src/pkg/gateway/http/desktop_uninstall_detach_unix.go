//go:build !windows

package http

import (
	"os/exec"
	"syscall"
)

func detachUninstallCmd(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
