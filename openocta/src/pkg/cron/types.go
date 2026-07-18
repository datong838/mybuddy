// Package cron provides cron job scheduling and storage.
package cron

// CronSchedule defines when a job runs.
type CronSchedule struct {
	Kind     string `json:"kind"` // "at" | "every" | "cron"
	At       string `json:"at,omitempty"`
	EveryMs  int64  `json:"everyMs,omitempty"`
	AnchorMs *int64 `json:"anchorMs,omitempty"`
	Expr     string `json:"expr,omitempty"`
	Tz       string `json:"tz,omitempty"`
}

// CronDelivery defines how to deliver job output (aligns with openclaw CronDelivery).
type CronDelivery struct {
	Mode       string `json:"mode"`              // "none" | "announce" | "webhook"
	Channel    string `json:"channel,omitempty"` // channel id or "last"
	To         string `json:"to,omitempty"`      // chat/user id for announce; URL for webhook
	BestEffort bool   `json:"bestEffort,omitempty"`
}

// CronPayload is the job payload.
type CronPayload struct {
	Kind    string `json:"kind"` // "systemEvent" | "agentTurn"
	Text    string `json:"text,omitempty"`
	Message string `json:"message,omitempty"`
}

// CronRunConfig holds per-run agent resources for isolated cron jobs.
type CronRunConfig struct {
	ModelRef    string                 `json:"modelRef,omitempty"`
	SkillKeys   []string               `json:"skillKeys,omitempty"`
	McpServers  []string               `json:"mcpServers,omitempty"`
	ExtraParams map[string]interface{} `json:"extraParams,omitempty"`
}

// CronJobState holds runtime state.
type CronJobState struct {
	NextRunAtMs       *int64 `json:"nextRunAtMs,omitempty"`
	RunningAtMs       *int64 `json:"runningAtMs,omitempty"`
	LastRunAtMs       *int64 `json:"lastRunAtMs,omitempty"`
	LastStatus        string `json:"lastStatus,omitempty"`
	LastError         string `json:"lastError,omitempty"`
	LastDurationMs    *int64 `json:"lastDurationMs,omitempty"`
	ConsecutiveErrors int    `json:"consecutiveErrors,omitempty"`
}

// CronJob is a scheduled job.
type CronJob struct {
	ID      string `json:"id"`
	AgentID string `json:"agentId,omitempty"`
	// DigitalEmployeeID 若设置，则该任务运行时会使用数字员工会话（agent:main:employee:<id>）。
	// 仅保存员工 id；展示名由前端通过 employees.list 解析。
	DigitalEmployeeID string       `json:"digitalEmployeeId,omitempty"`
	Name              string       `json:"name"`
	Description       string       `json:"description,omitempty"`
	Enabled           bool         `json:"enabled"`
	DeleteAfterRun    bool         `json:"deleteAfterRun,omitempty"`
	CreatedAtMs       int64        `json:"createdAtMs"`
	UpdatedAtMs       int64        `json:"updatedAtMs"`
	Schedule          CronSchedule `json:"schedule"`
	SessionTarget     string       `json:"sessionTarget"`
	// SessionKey 可选，用于定时调度时复用同一会话。格式：agent:main:cron:<jobId>。
	// 手动触发时始终生成新 sessionKey（agent:main:cron:<jobId>:run:<sessionId>），忽略此字段。
	SessionKey string         `json:"sessionKey,omitempty"`
	WakeMode   string         `json:"wakeMode"`
	Payload    CronPayload    `json:"payload"`
	Delivery   *CronDelivery  `json:"delivery,omitempty"`
	RunConfig  *CronRunConfig `json:"runConfig,omitempty"`
	State      CronJobState   `json:"state"`
}

// StoreFile is the persistent store format.
type StoreFile struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}
