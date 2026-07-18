package init

import (
	"os"

	"github.com/openocta/openocta/pkg/bundled"
)

// InitBundled installs shipped peekaboo and inner_skills on supported desktop platforms.
func InitBundled() error {
	env := func(k string) string { return os.Getenv(k) }
	return bundled.InstallAll(env)
}
