package stream

import "testing"

func TestIsLeakedAssistantText(t *testing.T) {
	leaked := `(Empty response: {'content':[{'type':'thinking','thinking':'hi'}],'stop_reason':'end_turn'})`
	if !IsLeakedAssistantText(leaked) {
		t.Fatal("expected leaked payload to be detected")
	}
	if IsLeakedAssistantText("你好！有什么可以帮你的？") {
		t.Fatal("normal greeting should not be filtered")
	}
}
