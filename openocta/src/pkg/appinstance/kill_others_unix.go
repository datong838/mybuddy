//go:build !windows

package appinstance

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func parentPIDsToPreserve() []int {
	pp := os.Getppid()
	if pp <= 0 {
		return nil
	}
	return []int{pp}
}

func findOtherInstancePIDs(skip map[int]struct{}) []int {
	if runtime.GOOS == "linux" {
		return listByProc(skip)
	}
	return listByPs(skip)
}

func listByProc(skip map[int]struct{}) []int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if _, omit := skip[pid]; omit {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil || len(cmdline) == 0 {
			continue
		}
		parts := bytes.Split(cmdline, []byte{0})
		if len(parts) == 0 || len(parts[0]) == 0 {
			continue
		}
		if isOurProcessBase(string(parts[0])) {
			out = append(out, pid)
		}
	}
	return out
}

func listByPs(skip map[int]struct{}) []int {
	// BSD/macOS and other Unix without reliable /proc scanning for argv0.
	out, err := exec.Command("ps", "-ax", "-o", "pid=", "-o", "command=").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		if _, omit := skip[pid]; omit {
			continue
		}
		first := fields[1]
		if isOurProcessBase(first) {
			pids = append(pids, pid)
		}
	}
	return pids
}

func terminatePIDs(pids []int) {
	for _, pid := range pids {
		_ = unix.Kill(pid, unix.SIGTERM)
	}
	time.Sleep(300 * time.Millisecond)
	for _, pid := range pids {
		if err := unix.Kill(pid, 0); err != nil {
			continue
		}
		_ = unix.Kill(pid, unix.SIGKILL)
	}
}
