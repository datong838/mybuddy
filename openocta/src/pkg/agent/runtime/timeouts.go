package runtime

import (
	"os"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/config"
)

// DefaultAgentRunDuration returns the configured agent run timeout.
func DefaultAgentRunDuration(env func(string) string, cfg *config.OpenOctaConfig) time.Duration {
	if env == nil {
		env = os.Getenv
	}
	if v := strings.TrimSpace(env("OPENOCTA_AGENT_RUN_TIMEOUT")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	if cfg != nil && cfg.Agents != nil && cfg.Agents.Defaults != nil && cfg.Agents.Defaults.TimeoutSeconds != nil {
		if sec := *cfg.Agents.Defaults.TimeoutSeconds; sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return 10 * time.Minute
}
