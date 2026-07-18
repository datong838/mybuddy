package localbackend_test

import (
	"context"
	"io"
	"runtime"
	"strings"
	"testing"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk/filesystem"

	"github.com/openocta/openocta/pkg/agent/eino/localbackend"
)

func TestBackendExecuteEcho(t *testing.T) {
	ctx := context.Background()
	backend, err := localbackend.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	var cmd string
	switch runtime.GOOS {
	case "windows":
		cmd = "echo openocta-shell"
	default:
		cmd = "echo openocta-shell"
	}

	resp, err := backend.Execute(ctx, &filesystem.ExecuteRequest{Command: cmd})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp == nil {
		t.Fatal("nil response")
	}
	if resp.ExitCode == nil || *resp.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %v", resp.ExitCode)
	}
	if !strings.Contains(strings.ToLower(resp.Output), "openocta-shell") {
		t.Fatalf("unexpected output: %q", resp.Output)
	}
}

func TestBackendExecuteStreamingEcho(t *testing.T) {
	ctx := context.Background()
	backend, err := localbackend.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	sr, err := backend.ExecuteStreaming(ctx, &filesystem.ExecuteRequest{Command: "echo stream-ok"})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}

	var chunks []string
	for {
		chunk, recvErr := sr.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("Recv: %v", recvErr)
		}
		if chunk != nil && chunk.Output != "" {
			chunks = append(chunks, chunk.Output)
		}
	}
	out := strings.ToLower(strings.Join(chunks, ""))
	if !strings.Contains(out, "stream-ok") {
		t.Fatalf("unexpected stream output: %q", out)
	}
}

func TestBackendValidateCommand(t *testing.T) {
	ctx := context.Background()
	backend, err := localbackend.NewBackend(ctx, &localbk.Config{
		ValidateCommand: func(command string) error {
			if strings.Contains(command, "blocked") {
				return context.Canceled
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewBackend: %v", err)
	}

	_, err = backend.Execute(ctx, &filesystem.ExecuteRequest{Command: "echo blocked"})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
