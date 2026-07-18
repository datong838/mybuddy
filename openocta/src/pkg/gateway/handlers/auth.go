package handlers

import (
	"os"
	"strings"
)

const envGatewayToken = "OPENOCTA_GATEWAY_TOKEN"

// GetExpectedGatewayToken returns the gateway token from config or env.
// When loadSnapshot is nil or returns no token, falls back to OPENOCTA_GATEWAY_TOKEN.
func GetExpectedGatewayToken(loadSnapshot func() (*ConfigSnapshot, error)) string {
	if loadSnapshot != nil {
		snap, err := loadSnapshot()
		if err == nil && snap != nil && snap.Config != nil &&
			snap.Config.Gateway != nil && snap.Config.Gateway.Auth != nil {
			t := strings.TrimSpace(snap.Config.Gateway.Auth.Token)
			if t != "" {
				return t
			}
		}
	}
	return strings.TrimSpace(os.Getenv(envGatewayToken))
}
