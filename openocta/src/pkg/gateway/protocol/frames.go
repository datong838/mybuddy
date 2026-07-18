// Package protocol defines the Gateway WebSocket frame formats and handshake.
// Compatible with dist/protocol.schema.json (TypeBox → JSON Schema).
package protocol

// PROTOCOL_VERSION is the current Gateway protocol version.
const PROTOCOL_VERSION = 3

// RequestFrame is a client → server request.
type RequestFrame struct {
	Type   string      `json:"type"` // "req"
	ID     string      `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}

// ResponseFrame is a server → client response.
type ResponseFrame struct {
	Type    string      `json:"type"` // "res"
	ID      string      `json:"id"`
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload,omitempty"`
	Error   *ErrorShape `json:"error,omitempty"`
}

// EventFrame is a server → client event.
type EventFrame struct {
	Type         string        `json:"type"` // "event"
	Event        string        `json:"event"`
	Payload      interface{}   `json:"payload,omitempty"`
	Seq          *int64        `json:"seq,omitempty"`
	StateVersion *StateVersion `json:"stateVersion,omitempty"`
}

// GatewayFrame is a discriminated union: req | res | event.
type GatewayFrame struct {
	Type         string        `json:"type"`
	ID           string        `json:"id,omitempty"`
	Method       string        `json:"method,omitempty"`
	Params       interface{}   `json:"params,omitempty"`
	OK           bool          `json:"ok,omitempty"`
	Payload      interface{}   `json:"payload,omitempty"`
	Error        *ErrorShape   `json:"error,omitempty"`
	Event        string        `json:"event,omitempty"`
	Seq          *int64        `json:"seq,omitempty"`
	StateVersion *StateVersion `json:"stateVersion,omitempty"`
}

// ErrorShape is the canonical error payload.
type ErrorShape struct {
	Code         string      `json:"code"`
	Message      string      `json:"message"`
	Details      interface{} `json:"details,omitempty"`
	Retryable    *bool       `json:"retryable,omitempty"`
	RetryAfterMs *int64      `json:"retryAfterMs,omitempty"`
}

// StateVersion tracks presence/health version.
type StateVersion struct {
	Presence int `json:"presence"`
	Health   int `json:"health"`
}

// ConnectParams is the client handshake payload.
type ConnectParams struct {
	MinProtocol int               `json:"minProtocol"`
	MaxProtocol int               `json:"maxProtocol"`
	Client      ConnectClientInfo `json:"client"`
	Caps        []string          `json:"caps,omitempty"`
	Commands    []string          `json:"commands,omitempty"`
	Permissions map[string]bool   `json:"permissions,omitempty"`
	PathEnv     string            `json:"pathEnv,omitempty"`
	Role        string            `json:"role,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
	Device      *ConnectDevice    `json:"device,omitempty"`
	Auth        *ConnectAuth      `json:"auth,omitempty"`
	Locale      string            `json:"locale,omitempty"`
	UserAgent   string            `json:"userAgent,omitempty"`
}

// ConnectClientInfo identifies the client.
type ConnectClientInfo struct {
	ID              string `json:"id"`
	DisplayName     string `json:"displayName,omitempty"`
	Version         string `json:"version"`
	Platform        string `json:"platform"`
	DeviceFamily    string `json:"deviceFamily,omitempty"`
	ModelIdentifier string `json:"modelIdentifier,omitempty"`
	Mode            string `json:"mode"`
	InstanceID      string `json:"instanceId,omitempty"`
}

// ConnectDevice is device pairing info.
type ConnectDevice struct {
	ID        string `json:"id"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"`
	SignedAt  int64  `json:"signedAt"`
	Nonce     string `json:"nonce,omitempty"`
}

// ConnectAuth is token/password auth.
type ConnectAuth struct {
	Token    string `json:"token,omitempty"`
	Password string `json:"password,omitempty"`
}

// HelloOk is the server handshake response.
type HelloOk struct {
	Type          string        `json:"type"` // "hello-ok"
	Protocol      int           `json:"protocol"`
	Server        HelloServer   `json:"server"`
	Features      HelloFeatures `json:"features"`
	Snapshot      Snapshot      `json:"snapshot"`
	CanvasHostURL string        `json:"canvasHostUrl,omitempty"`
	Auth          *HelloAuth    `json:"auth,omitempty"`
	Policy        HelloPolicy   `json:"policy"`
}

// HelloServer identifies the server.
type HelloServer struct {
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
	Host    string `json:"host,omitempty"`
	ConnID  string `json:"connId"`
}

// HelloFeatures lists supported methods and events.
type HelloFeatures struct {
	Methods []string `json:"methods"`
	Events  []string `json:"events"`
}

// HelloAuth is device token info.
type HelloAuth struct {
	DeviceToken string   `json:"deviceToken"`
	Role        string   `json:"role"`
	Scopes      []string `json:"scopes"`
	IssuedAtMs  *int64   `json:"issuedAtMs,omitempty"`
}

// HelloPolicy defines connection limits.
type HelloPolicy struct {
	MaxPayload       int `json:"maxPayload"`
	MaxBufferedBytes int `json:"maxBufferedBytes"`
	TickIntervalMs   int `json:"tickIntervalMs"`
}

// Snapshot is the initial state sent in hello-ok.
type Snapshot struct {
	Presence        []PresenceEntry  `json:"presence"`
	Health          interface{}      `json:"health"`
	StateVersion    StateVersion     `json:"stateVersion"`
	UptimeMs        int64            `json:"uptimeMs"`
	ConfigPath      string           `json:"configPath,omitempty"`
	StateDir        string           `json:"stateDir,omitempty"`
	SessionDefaults *SessionDefaults `json:"sessionDefaults,omitempty"`
}

// PresenceEntry is a connected client entry.
type PresenceEntry struct {
	Host             string   `json:"host,omitempty"`
	IP               string   `json:"ip,omitempty"`
	Version          string   `json:"version,omitempty"`
	Platform         string   `json:"platform,omitempty"`
	DeviceFamily     string   `json:"deviceFamily,omitempty"`
	ModelIdentifier  string   `json:"modelIdentifier,omitempty"`
	Mode             string   `json:"mode,omitempty"`
	LastInputSeconds *int64   `json:"lastInputSeconds,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Text             string   `json:"text,omitempty"`
	Ts               int64    `json:"ts"`
	DeviceID         string   `json:"deviceId,omitempty"`
	Roles            []string `json:"roles,omitempty"`
	Scopes           []string `json:"scopes,omitempty"`
	InstanceID       string   `json:"instanceId,omitempty"`
}

// SessionDefaults holds default session config.
type SessionDefaults struct {
	DefaultAgentID string `json:"defaultAgentId"`
	MainKey        string `json:"mainKey"`
	MainSessionKey string `json:"mainSessionKey"`
	Scope          string `json:"scope,omitempty"`
}

// TickEvent is a periodic tick.
type TickEvent struct {
	Ts int64 `json:"ts"`
}

// ShutdownEvent is sent when the server is shutting down.
type ShutdownEvent struct {
	Reason            string `json:"reason"`
	RestartExpectedMs *int64 `json:"restartExpectedMs,omitempty"`
}
