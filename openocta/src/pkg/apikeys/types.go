package apikeys

const (
	BindingModeResources = "resources"
	BindingModeEmployee  = "employee"
)

// DefaultAllowedPaths are pre-filled path prefixes for new API keys.
var DefaultAllowedPaths = []string{
	"/openocta/open/v1/ping",
	"/openocta/open/v1/completion",
}

// Record is the persisted API key (secret hash only).
type Record struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	KeyPrefix         string   `json:"keyPrefix"`
	KeyHash           string   `json:"keyHash"`
	KeyEnc            string   `json:"keyEnc,omitempty"`
	AllowedPaths      []string `json:"allowedPaths"`
	BindingMode       string   `json:"bindingMode"`
	AllowedModels     []string `json:"allowedModels,omitempty"`
	SkillKeys         []string `json:"skillKeys,omitempty"`
	McpServers        []string `json:"mcpServers,omitempty"`
	DigitalEmployeeID string   `json:"digitalEmployeeId,omitempty"`
	MonthlyTokenLimit *int64   `json:"monthlyTokenLimit,omitempty"`
	MonthlyTokensUsed int64    `json:"monthlyTokensUsed,omitempty"`
	UsageMonth        string   `json:"usageMonth,omitempty"`
	Enabled           bool     `json:"enabled"`
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
}

// PublicEntry is returned to the UI (no hash).
type PublicEntry struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	KeyPrefix         string   `json:"keyPrefix"`
	AllowedPaths      []string `json:"allowedPaths"`
	BindingMode       string   `json:"bindingMode"`
	AllowedModels     []string `json:"allowedModels,omitempty"`
	SkillKeys         []string `json:"skillKeys,omitempty"`
	McpServers        []string `json:"mcpServers,omitempty"`
	DigitalEmployeeID string   `json:"digitalEmployeeId,omitempty"`
	MonthlyTokenLimit *int64   `json:"monthlyTokenLimit,omitempty"`
	MonthlyTokensUsed int64    `json:"monthlyTokensUsed,omitempty"`
	UsageMonth        string   `json:"usageMonth,omitempty"`
	Enabled           bool     `json:"enabled"`
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
}

// CreateParams holds input for creating an API key.
type CreateParams struct {
	Name              string
	AllowedPaths      []string
	BindingMode       string
	AllowedModels     []string
	SkillKeys         []string
	McpServers        []string
	DigitalEmployeeID string
	MonthlyTokenLimit *int64
}

// CreateResult includes the plaintext secret for the admin UI.
type CreateResult struct {
	Entry  PublicEntry `json:"entry"`
	Secret string      `json:"secret"`
}

// SecretResult returns the stored plaintext secret for admin UI.
type SecretResult struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// RegenerateResult returns a newly generated secret.
type RegenerateResult struct {
	Entry  PublicEntry `json:"entry"`
	Secret string      `json:"secret"`
}

// UpdateParams holds patch fields for updating an API key.
type UpdateParams struct {
	Name              *string
	AllowedPaths      *[]string
	BindingMode       *string
	AllowedModels     *[]string
	SkillKeys         *[]string
	McpServers        *[]string
	DigitalEmployeeID *string
	MonthlyTokenLimit *int64
	Enabled           *bool
}
