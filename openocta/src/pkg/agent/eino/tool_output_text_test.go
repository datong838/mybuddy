package eino

import "testing"

func TestIsLeakedToolOutputText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		text string
		want bool
	}{
		{"command exited with non-zero code 127\n[stderr]:\n/bin/sh: stock: command not found", true},
		{"Output too large (8359). Full output saved to: /tmp/out", true},
		{"<persisted-output>raw</persisted-output>", true},
		{`{"matches":["browser"]}`, true},
		{"杭州今天的天气情况如下：\n\n**当前天气**", false},
		{"不客气！如果还有其他问题随时问我 😊", false},
	}
	for _, tc := range cases {
		if got := IsLeakedToolOutputText(tc.text); got != tc.want {
			t.Fatalf("IsLeakedToolOutputText(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}
