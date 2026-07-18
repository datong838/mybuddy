package localbackend

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/adk/filesystem"
)

// NormalizeShellCommand folds multiline command strings into a single executable line.
// Windows cmd.exe treats raw newlines as command terminators and silently drops the rest.
func NormalizeShellCommand(command string) string {
	s := strings.ReplaceAll(command, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if !strings.Contains(s, "\n") {
		return strings.TrimSpace(s)
	}
	if runtime.GOOS == "windows" {
		return collapseSpaces(strings.ReplaceAll(s, "\n", " "))
	}
	lines := strings.Split(s, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " && ")
}

func collapseSpaces(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

func normalizeExecuteRequest(input *filesystem.ExecuteRequest) (*filesystem.ExecuteRequest, error) {
	if input == nil || strings.TrimSpace(input.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}
	command := NormalizeShellCommand(input.Command)
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}
	if command == input.Command {
		return input, nil
	}
	return &filesystem.ExecuteRequest{
		Command:            command,
		RunInBackendGround: input.RunInBackendGround,
	}, nil
}
