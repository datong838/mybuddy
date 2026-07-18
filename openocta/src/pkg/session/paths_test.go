package session

import (
	"testing"
)

func TestSanitizeForSessionID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "main"},
		{"  ", "main"},
		{"main", "main"},
		{"employee:sre:run:uuid", "employee-sre-run-uuid"},
		{"agent:main:employee:sre:run:uuid", "agent-main-employee-sre-run-uuid"},
		{"simple-id", "simple-id"},
		{"simple_id", "simple_id"},
		{"UPPER", "upper"},
		{"a:b:c", "a-b-c"},
		{":::", "main"},
	}
	for _, tt := range tests {
		got := SanitizeForSessionID(tt.in)
		if got != tt.want {
			t.Errorf("SanitizeForSessionID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
