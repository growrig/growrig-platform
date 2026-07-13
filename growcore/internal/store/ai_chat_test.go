package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

func TestAIChatsArePersistedScopedAndArchived(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now()
	if err := st.SaveGrow(domain.Grow{ID: "grow-1", Name: "Summer grow", Stages: []string{"growth"}, Stage: "growth", StartedAt: now, StageStarted: now}); err != nil {
		t.Fatal(err)
	}
	chat := domain.AIChat{ID: "chat-1", UserID: "user-1", GrowID: "grow-1", Title: "How is VPD?", CreatedAt: now, UpdatedAt: now}
	// Equal timestamps must retain insertion order. Message IDs are random in
	// production and therefore cannot be used as a chronological tie-breaker.
	userMessage := domain.AIChatMessage{ID: "z-user", ChatID: chat.ID, Role: "user", Content: "How is VPD?", CreatedAt: now}
	assistantMessage := domain.AIChatMessage{ID: "a-assistant", ChatID: chat.ID, Role: "assistant", Content: "VPD is stable.", CreatedAt: now}
	if err := st.SaveAIChatExchange(&chat, userMessage, assistantMessage); err != nil {
		t.Fatal(err)
	}
	chats, err := st.AIChats("user-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 1 || chats[0].MessageCount != 2 || chats[0].GrowName != "Summer grow" || chats[0].Preview != "VPD is stable." {
		t.Fatalf("unexpected chats: %#v", chats)
	}
	other, err := st.AIChats("user-2", nil)
	if err != nil || len(other) != 0 {
		t.Fatalf("other user's chats = %#v, %v", other, err)
	}
	loaded, ok, err := st.AIChat(chat.ID, "user-1")
	if err != nil || !ok || loaded.Title != chat.Title {
		t.Fatalf("loaded chat = %#v, %v, %v", loaded, ok, err)
	}
	messages, err := st.AIChatMessages(chat.ID)
	if err != nil || len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("messages = %#v, %v", messages, err)
	}
	if ok, err := st.SetAIChatArchived(chat.ID, "user-2", true); err != nil || ok {
		t.Fatalf("other user archived chat: %v, %v", ok, err)
	}
	if ok, err := st.SetAIChatArchived(chat.ID, "user-1", true); err != nil || !ok {
		t.Fatalf("archive failed: %v, %v", ok, err)
	}
	archived := true
	chats, err = st.AIChats("user-1", &archived)
	if err != nil || len(chats) != 1 || !chats[0].Archived {
		t.Fatalf("archived chats = %#v, %v", chats, err)
	}
}
