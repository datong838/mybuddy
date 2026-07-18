//go:build !windows

package localbackend

import "os/exec"

func applyExecNoWindow(cmd *exec.Cmd) {}
