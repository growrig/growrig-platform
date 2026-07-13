package domain

import "time"

// AIChat is a persisted, user-owned conversation scoped to one grow.
type AIChat struct {
	ID           string          `json:"id"`
	UserID       string          `json:"-"`
	GrowID       string          `json:"growId"`
	GrowName     string          `json:"growName"`
	Title        string          `json:"title"`
	InstanceID   string          `json:"-"`
	InstanceName string          `json:"instanceName"`
	Archived     bool            `json:"archived"`
	MessageCount int             `json:"messageCount"`
	Preview      string          `json:"preview"`
	Messages     []AIChatMessage `json:"messages,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// AIChatMessage is one immutable user or assistant turn in a conversation.
type AIChatMessage struct {
	ID        string    `json:"id"`
	ChatID    string    `json:"-"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}
