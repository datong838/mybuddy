package localagents

import (
	"regexp"
	"strings"
)

var mentionPattern = regexp.MustCompile(`(?i)@([a-z][a-z0-9_-]*)`)

// ParseMessage extracts @agent task segments from a user message.
// Returns nil when no valid external-agent mentions are present.
func ParseMessage(message string) []TaskSegment {
	text := strings.TrimSpace(message)
	if text == "" {
		return nil
	}
	aliasMap := aliasToID()
	matches := mentionPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	var segments []TaskSegment
	for i, loc := range matches {
		if len(loc) < 4 {
			continue
		}
		alias := normalizeAlias(text[loc[2]:loc[3]])
		agentID, ok := aliasMap[alias]
		if !ok {
			continue
		}
		taskStart := loc[1]
		var taskEnd int
		if i+1 < len(matches) {
			taskEnd = matches[i+1][0]
		} else {
			taskEnd = len(text)
		}
		task := strings.TrimSpace(text[taskStart:taskEnd])
		task = strings.TrimLeft(task, ":：")
		task = strings.TrimSpace(task)
		if task == "" {
			continue
		}
		segments = append(segments, TaskSegment{AgentID: agentID, Task: task})
	}
	if len(segments) == 0 {
		return nil
	}
	return segments
}

// ValidateSegments checks that every segment targets an installed agent.
func ValidateSegments(segments []TaskSegment, installed map[string]AgentProbeResult) (missing []string, ok bool) {
	for _, seg := range segments {
		if _, found := installed[seg.AgentID]; !found {
			missing = append(missing, seg.AgentID)
		}
	}
	return missing, len(missing) == 0
}

// CoversEntireMessage reports whether the message is only @-delegated tasks (no free text outside mentions).
func CoversEntireMessage(message string, segments []TaskSegment) bool {
	if len(segments) == 0 {
		return false
	}
	trimmed := strings.TrimSpace(message)
	rebuilt := strings.Builder{}
	for i, seg := range segments {
		if i > 0 {
			rebuilt.WriteByte(' ')
		}
		rebuilt.WriteString("@" + seg.AgentID + " " + seg.Task)
	}
	return strings.TrimSpace(rebuilt.String()) == trimmed
}
