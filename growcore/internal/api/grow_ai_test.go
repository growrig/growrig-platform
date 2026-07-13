package api

import "testing"

func TestChatTitle(t *testing.T) {
	if got := chatTitle("  What   changed this week?  "); got != "What changed this week?" {
		t.Fatalf("title = %q", got)
	}
	long := "This is a deliberately long conversation title that should be shortened without breaking anything"
	if got := []rune(chatTitle(long)); len(got) != 58 || got[len(got)-1] != '…' {
		t.Fatalf("unexpected shortened title %q (%d runes)", string(got), len(got))
	}
}

func TestIntegrationChatContent(t *testing.T) {
	tests := []struct {
		name   string
		result any
		want   string
	}{
		{"ollama chat", map[string]any{"message": map[string]any{"content": "  healthy growth  "}}, "healthy growth"},
		{"ollama generate", map[string]any{"response": "check humidity"}, "check humidity"},
		{"openai compatible", map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "watch VPD"}}}}, "watch VPD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := integrationChatContent(tt.result)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("content = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIntegrationChatContentRejectsEmptyResponse(t *testing.T) {
	if _, err := integrationChatContent(map[string]any{"message": map[string]any{}}); err == nil {
		t.Fatal("expected empty response error")
	}
}
