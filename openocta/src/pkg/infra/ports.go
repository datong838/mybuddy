// Package infra provides port utilities for the CLI.
package infra

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PortProcess holds a PID and optional command.
type PortProcess struct {
	PID     int
	Command string
}

// ListPortListeners returns PIDs listening on the given port.
// Uses lsof -nP -iTCP:port -sTCP:LISTEN -FpFc.
func ListPortListeners(port int) ([]PortProcess, error) {
	cmd := exec.Command("lsof", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-FpFc")
	out, err := cmd.CombinedOutput()
	outStr := string(out)
	if err != nil {
		// lsof exits 1 when no listeners
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof: %w", err)
	}
	return parseLsofOutput(outStr), nil
}

// ForceFreePort sends SIGTERM to listeners on the port and waits briefly.
func ForceFreePort(port int) ([]PortProcess, error) {
	listeners, err := ListPortListeners(port)
	if err != nil {
		return nil, err
	}
	for _, p := range listeners {
		proc, _ := os.FindProcess(p.PID)
		if proc != nil {
			_ = proc.Signal(syscall.SIGTERM)
		}
	}
	if len(listeners) > 0 {
		time.Sleep(500 * time.Millisecond)
	}
	return listeners, nil
}

func parseLsofOutput(output string) []PortProcess {
	var result []PortProcess
	var current PortProcess
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			if current.PID > 0 {
				result = append(result, current)
			}
			pid, _ := strconv.Atoi(line[1:])
			current = PortProcess{PID: pid}
		case 'c':
			current.Command = line[1:]
		}
	}
	if current.PID > 0 {
		result = append(result, current)
	}
	return result
}
