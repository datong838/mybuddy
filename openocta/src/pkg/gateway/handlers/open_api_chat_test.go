package handlers

import (
	"testing"

	"github.com/openocta/openocta/pkg/apikeys"
)

func TestOpenAPISessionKey(t *testing.T) {
	rec := &apikeys.Record{
		ID:          "key1",
		BindingMode: apikeys.BindingModeResources,
	}
	key, err := openAPISessionKey(rec, "conv-01")
	if err != nil {
		t.Fatalf("openAPISessionKey() error = %v", err)
	}
	want := "agent:main:open-api:key1:conv-01"
	if key != want {
		t.Fatalf("openAPISessionKey() = %q, want %q", key, want)
	}

	rec.BindingMode = apikeys.BindingModeEmployee
	rec.DigitalEmployeeID = "emp-a"
	key, err = openAPISessionKey(rec, "")
	if err != nil {
		t.Fatalf("employee openAPISessionKey() error = %v", err)
	}
	want = "agent:main:employee:emp-a:run:open-api:key1:default"
	if key != want {
		t.Fatalf("employee openAPISessionKey() = %q, want %q", key, want)
	}
}

func TestExtractOpenAPILastUserMessage(t *testing.T) {
	msgs := []map[string]interface{}{
		{"role": "system", "content": "ignore"},
		{"role": "user", "content": "hello"},
	}
	got, err := extractOpenAPILastUserMessage(msgs)
	if err != nil {
		t.Fatalf("extractOpenAPILastUserMessage() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("extractOpenAPILastUserMessage() = %q, want hello", got)
	}
}

func TestOpenAPIChatResources(t *testing.T) {
	rec := &apikeys.Record{
		BindingMode: apikeys.BindingModeResources,
		SkillKeys:   []string{"skill-a"},
		McpServers:  []string{"mcp-b"},
	}
	res := openAPIChatResources(rec)
	if res == nil || res["configured"] != true {
		t.Fatalf("openAPIChatResources() = %#v", res)
	}
	if len(res["skillKeys"].([]string)) != 1 {
		t.Fatalf("skillKeys = %#v", res["skillKeys"])
	}

	rec.BindingMode = apikeys.BindingModeEmployee
	if openAPIChatResources(rec) != nil {
		t.Fatalf("employee resources should be nil")
	}
}
