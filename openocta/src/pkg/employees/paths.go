package employees

import (
	"path/filepath"

	"github.com/openocta/openocta/pkg/paths"
)

// ResolveEmployeesDir 返回数字员工模板所在的根目录。
// 默认路径基于 ResolveStateDir：
// - Linux/macOS: ~/.openocta/employees
// - Windows: %APPDATA%\openocta\employees
// 也支持通过 OPENOCTA_STATE_DIR 等环境变量间接控制。
func ResolveEmployeesDir(env func(string) string) string {
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "employees")
}
