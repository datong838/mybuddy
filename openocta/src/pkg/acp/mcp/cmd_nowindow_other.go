//go:build !windows

package mcp

import "os/exec"

func configureMCPCommand(cmd *exec.Cmd) {}
