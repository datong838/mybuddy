package handlers

import (
	"context"

	"github.com/openocta/openocta/pkg/appupdate"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// UpdateRunHandler handles "update.run".
func UpdateRunHandler(opts HandlerOpts) error {
	if !appupdate.InstallAllowed() {
		result := map[string]interface{}{
			"status":     "skipped",
			"mode":       "unsupported",
			"reason":     "当前运行模式不支持自动安装",
			"steps":      []interface{}{},
			"durationMs": 0,
		}
		opts.Respond(true, map[string]interface{}{
			"ok":      true,
			"result":  result,
			"restart": nil,
			"sentinel": map[string]interface{}{
				"path":    nil,
				"payload": nil,
			},
		}, nil, nil)
		return nil
	}

	versionParam, _ := opts.Params["version"].(string)
	if versionParam == "" {
		check := appupdate.Check(context.Background(), appupdate.CheckOptions{Force: true, Record: false})
		versionParam = check.LatestVersion
	}
	started, err := appupdate.StartInstallAsync(versionParam)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: err.Error(),
		}, nil)
		return nil
	}
	status := "started"
	if !started {
		status = "already_running"
	}
	opts.Respond(true, map[string]interface{}{
		"ok": true,
		"result": map[string]interface{}{
			"status":     status,
			"mode":       installModeLabel(),
			"version":    versionParam,
			"install":    appupdate.InstallStatus(),
			"durationMs": 0,
		},
		"restart": nil,
		"sentinel": map[string]interface{}{
			"path":    nil,
			"payload": nil,
		},
	}, nil, nil)
	return nil
}

func installModeLabel() string {
	if appupdate.DesktopMode() {
		return "desktop"
	}
	if appupdate.InstallAllowed() {
		return "service"
	}
	return "unsupported"
}
