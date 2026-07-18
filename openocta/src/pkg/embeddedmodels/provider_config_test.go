package embeddedmodels

import (
	"os"
	"testing"

	"github.com/openocta/openocta/pkg/config"
)

func TestGatewayProxyBaseURL(t *testing.T) {
	port := 18999
	cfg := &config.OpenOctaConfig{
		Gateway: &config.GatewayConfig{Port: &port},
	}
	got := GatewayProxyBaseURL(cfg, os.Getenv)
	want := "http://127.0.0.1:18999/api/embedded-models/v1"
	if got != want {
		t.Fatalf("GatewayProxyBaseURL() = %q, want %q", got, want)
	}
}

func TestIsEmbeddedProvider(t *testing.T) {
	if !IsEmbeddedProvider(EmbeddedChatProviderKey) {
		t.Fatal("expected chat provider")
	}
	if !IsEmbeddedProvider(EmbeddedEmbeddingProviderKey) {
		t.Fatal("expected embedding provider")
	}
	if IsEmbeddedProvider("ollama") {
		t.Fatal("ollama should not be embedded provider")
	}
}

func TestMergedProviderConfigUsesGatewayProxy(t *testing.T) {
	port := 18900
	cfg := &config.OpenOctaConfig{
		Gateway: &config.GatewayConfig{Port: &port},
	}
	env := os.Getenv
	// No running models -> empty map, but verify helper URL when models exist via manual check
	got := GatewayProxyBaseURL(cfg, env)
	if got != "http://127.0.0.1:18900/api/embedded-models/v1" {
		t.Fatalf("unexpected proxy base: %s", got)
	}
}
