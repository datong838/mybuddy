package appupdate

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	return v
}

// CompareVersions compares current and latest semver strings (optional v prefix).
// Returns -1 if current < latest, 0 if equal, 1 if current > latest.
func CompareVersions(current, latest string) (int, error) {
	c, err := semver.NewVersion(normalizeVersion(current))
	if err != nil {
		return 0, fmt.Errorf("invalid current version %q: %w", current, err)
	}
	l, err := semver.NewVersion(normalizeVersion(latest))
	if err != nil {
		return 0, fmt.Errorf("invalid latest version %q: %w", latest, err)
	}
	return c.Compare(l), nil
}

// IsNewer reports whether latest is strictly newer than current.
func IsNewer(current, latest string) (bool, error) {
	cmp, err := CompareVersions(current, latest)
	if err != nil {
		return false, err
	}
	return cmp < 0, nil
}
