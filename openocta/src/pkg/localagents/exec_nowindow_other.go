//go:build !windows

package localagents

import "os/exec"

func applyExecNoWindow(cmd *exec.Cmd) {}
