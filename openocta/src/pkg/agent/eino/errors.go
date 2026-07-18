package eino

import "errors"

var (
	ErrMissingModel        = errors.New("eino: model factory is required")
	ErrRuntimeClosed       = errors.New("eino: runtime is closed")
	ErrConcurrentExecution = errors.New("eino: concurrent execution on same session is not allowed")
)
