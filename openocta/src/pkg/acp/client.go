package acp

// Client is the ACP client (Phase 6b: stub).
// Connects to Gateway to send agent requests and receive events.
type Client struct {
	GatewayURL             string
	Token                  string
	Password               string
	DefaultSessionKey      string
	DefaultSessionLabel    string
	RequireExistingSession bool
}

// NewClient creates a new ACP client.
func NewClient(opts struct {
	GatewayURL string
	Token      string
	Password   string
}) *Client {
	return &Client{
		GatewayURL: opts.GatewayURL,
		Token:      opts.Token,
		Password:   opts.Password,
	}
}

// Connect establishes connection to Gateway (stub).
func (c *Client) Connect() error {
	return nil
}

// Stop closes the connection.
func (c *Client) Stop() error {
	return nil
}
