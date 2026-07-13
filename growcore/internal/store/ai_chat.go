package store

import (
	"database/sql"
	"time"

	"github.com/growrig/growrig/growcore/internal/domain"
)

const aiChatSelect = `SELECT c.id, c.user_id, c.grow_id, COALESCE(g.name, ''),
	c.environment_id, COALESCE(e.name, ''), c.title,
	c.instance_id, COALESCE(i.name, ''), c.archived, c.created, c.updated,
	(SELECT COUNT(*) FROM ai_chat_messages m WHERE m.chat_id=c.id),
	COALESCE((SELECT substr(m.content, 1, 180) FROM ai_chat_messages m
		WHERE m.chat_id=c.id ORDER BY m.created DESC, m.rowid DESC LIMIT 1), '')
	FROM ai_chats c
	LEFT JOIN grows g ON g.id=c.grow_id
	LEFT JOIN environments e ON e.id=c.environment_id
	LEFT JOIN integration_instances i ON i.id=c.instance_id`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAIChat(row rowScanner) (domain.AIChat, error) {
	var chat domain.AIChat
	var archived int
	var created, updated int64
	err := row.Scan(&chat.ID, &chat.UserID, &chat.GrowID, &chat.GrowName,
		&chat.EnvironmentID, &chat.EnvironmentName, &chat.Title,
		&chat.InstanceID, &chat.InstanceName, &archived, &created, &updated,
		&chat.MessageCount, &chat.Preview)
	chat.Archived = archived != 0
	chat.CreatedAt = time.UnixMilli(created)
	chat.UpdatedAt = time.UnixMilli(updated)
	return chat, err
}

func (s *Store) SaveAIChat(chat domain.AIChat) error {
	_, err := s.db.Exec(`INSERT INTO ai_chats
		(id, user_id, grow_id, environment_id, title, instance_id, archived, created, updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, chat.ID, chat.UserID, chat.GrowID, chat.EnvironmentID,
		chat.Title, chat.InstanceID, chat.Archived, chat.CreatedAt.UnixMilli(), chat.UpdatedAt.UnixMilli())
	return err
}

// AIChats lists conversations owned by userID. A nil archived filter returns both.
func (s *Store) AIChats(userID string, archived *bool) ([]domain.AIChat, error) {
	query := aiChatSelect + ` WHERE c.user_id=?`
	args := []any{userID}
	if archived != nil {
		query += ` AND c.archived=?`
		args = append(args, *archived)
	}
	query += ` ORDER BY c.updated DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chats := []domain.AIChat{}
	for rows.Next() {
		chat, err := scanAIChat(rows)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *Store) AIChat(id, userID string) (domain.AIChat, bool, error) {
	chat, err := scanAIChat(s.db.QueryRow(aiChatSelect+` WHERE c.id=? AND c.user_id=?`, id, userID))
	if err == sql.ErrNoRows {
		return domain.AIChat{}, false, nil
	}
	return chat, err == nil, err
}

func (s *Store) AIChatMessages(chatID string) ([]domain.AIChatMessage, error) {
	rows, err := s.db.Query(`SELECT id, role, content, created FROM ai_chat_messages
		WHERE chat_id=? ORDER BY created, rowid`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := []domain.AIChatMessage{}
	for rows.Next() {
		var message domain.AIChatMessage
		var created int64
		message.ChatID = chatID
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &created); err != nil {
			return nil, err
		}
		message.CreatedAt = time.UnixMilli(created)
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

// SaveAIChatExchange atomically creates an optional chat and appends one user
// turn plus its assistant response.
func (s *Store) SaveAIChatExchange(chat *domain.AIChat, userMessage, assistantMessage domain.AIChatMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if chat != nil {
		if _, err := tx.Exec(`INSERT INTO ai_chats
			(id, user_id, grow_id, environment_id, title, instance_id, archived, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, chat.ID, chat.UserID, chat.GrowID, chat.EnvironmentID,
			chat.Title, chat.InstanceID, chat.Archived, chat.CreatedAt.UnixMilli(), chat.UpdatedAt.UnixMilli()); err != nil {
			return err
		}
	}
	for _, message := range []domain.AIChatMessage{userMessage, assistantMessage} {
		if _, err := tx.Exec(`INSERT INTO ai_chat_messages (id, chat_id, role, content, created)
			VALUES (?, ?, ?, ?, ?)`, message.ID, message.ChatID, message.Role, message.Content, message.CreatedAt.UnixMilli()); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE ai_chats SET updated=? WHERE id=?`, assistantMessage.CreatedAt.UnixMilli(), assistantMessage.ChatID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetAIChatArchived(id, userID string, archived bool) (bool, error) {
	result, err := s.db.Exec(`UPDATE ai_chats SET archived=?, updated=? WHERE id=? AND user_id=?`, archived, time.Now().UnixMilli(), id, userID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
