//go:build !windows

package tools

import "os/exec"

func applyExecNoWindow(cmd *exec.Cmd) {}
