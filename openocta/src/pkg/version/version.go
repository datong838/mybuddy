package version

import (
	"os"

	_ "github.com/openocta/openocta/embed" // ensure embedded .env is loaded before Version is set
)

// Version is the OpenOcta version. Set at build via -ldflags "-X .../version.Version=xxx".
// Fallback: OPENOCTA_BUNDLED_VERSION env (for local dev) or "0.0.1-dev".
var Version = "0.0.1-dev"

func init() {
	if v := os.Getenv("OPENOCTA_BUNDLED_VERSION"); v != "" && Version == "0.0.1-dev" {
		Version = v
	}
}
