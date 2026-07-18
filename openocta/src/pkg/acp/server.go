package acp

// Server is the ACP server (Phase 6b: stub).
// Full implementation connects to Gateway via WebSocket and bridges
// stdio to AgentSideConnection (ndJSON over stdin/stdout).
type Server struct {
	opts ServerOptions
}

// NewServer creates a new ACP server.
func NewServer(opts ServerOptions) *Server {
	return &Server{opts: opts}
}

// Start starts the server (stub: no-op).
func (s *Server) Start() error {
	return nil
}

// Stop stops the server.
func (s *Server) Stop() error {
	return nil
}
