package appupdate

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    int
	}{
		{"v1.0.0", "v1.1.0", -1},
		{"1.0.0", "1.0.0", 0},
		{"v2.0.0", "v1.9.9", 1},
	}
	for _, tc := range tests {
		got, err := CompareVersions(tc.current, tc.latest)
		if err != nil {
			t.Fatalf("CompareVersions(%q, %q): %v", tc.current, tc.latest, err)
		}
		if got != tc.want {
			t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	ok, err := IsNewer("v1.0.0", "v1.0.1")
	if err != nil || !ok {
		t.Fatalf("expected newer, got ok=%v err=%v", ok, err)
	}
	ok, err = IsNewer("v1.0.1", "v1.0.0")
	if err != nil || ok {
		t.Fatalf("expected not newer, got ok=%v err=%v", ok, err)
	}
}
