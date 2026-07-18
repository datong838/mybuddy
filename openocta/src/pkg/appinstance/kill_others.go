// Package appinstance handles single-instance style cleanup for packaged OpenOcta builds.
package appinstance

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KillOtherOpenOctaProcesses terminates other running processes whose executable basename
// matches packaged OpenOcta binaries (openocta, openocta-launcher, OpenOcta), excluding
// this process and its parent (so a gateway child does not kill openocta-launcher on Windows).
// Used so exe/dmg/rpm upgrades can restart without file locks or port conflicts.
//
// Set OPENOCTA_SKIP_SINGLETON_KILL=1 to disable (e.g. local debugging).
func KillOtherOpenOctaProcesses() {
	if strings.TrimSpace(os.Getenv("OPENOCTA_SKIP_SINGLETON_KILL")) == "1" {
		return
	}
	self := os.Getpid()
	skip := map[int]struct{}{self: {}}
	for _, p := range parentPIDsToPreserve() {
		if p > 0 {
			skip[p] = struct{}{}
		}
	}
	pids := findOtherInstancePIDs(skip)
	if len(pids) == 0 {
		return
	}
	terminatePIDs(pids)
	time.Sleep(250 * time.Millisecond)
}

func isOurProcessBase(argv0 string) bool {
	b := strings.TrimSuffix(strings.ToLower(filepath.Base(strings.TrimSpace(argv0))), ".exe")
	return b == "openocta" || b == "openocta-launcher"
}
