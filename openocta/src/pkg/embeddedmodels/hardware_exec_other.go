//go:build !windows

package embeddedmodels

import "os/exec"

func applyExecNoWindow(cmd *exec.Cmd) {}
