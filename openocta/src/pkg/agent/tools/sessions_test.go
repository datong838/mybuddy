package tools

import (
	"testing"
)

func TestSessionIDFromSessionKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"", "main"},
		{"main", "main"},
		{"agent:main:main", "main"},
		{"agent:main:employee:sre:run:uuid", "employee-sre-run-uuid"},
		{"agent:main:employee:sre:run:550e8400-e29b-41d4-a716-446655440000", "employee-sre-run-550e8400-e29b-41d4-a716-446655440000"},
		{"agent:main:employee:clickhouse", "employee-clickhouse"},
		{"agent:main:dm:alice", "dm-alice"},
		{"custom:550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		got := SessionIDFromSessionKey(tt.key)
		if got != tt.want {
			t.Errorf("SessionIDFromSessionKey(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}
