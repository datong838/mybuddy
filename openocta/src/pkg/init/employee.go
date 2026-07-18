package init

import (
	"os"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/employees"
)

// InitEmployee 在项目启动时确保 ~/.openocta/employees 目录存在。
// 后端不再 embed 默认数字员工，用户需自行创建或从市场安装。
func InitEmployee(_ *config.OpenOctaConfig) error {
	env := func(k string) string { return os.Getenv(k) }
	root := employees.ResolveEmployeesDir(env)
	return os.MkdirAll(root, 0o755)
}
