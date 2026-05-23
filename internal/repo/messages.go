package repo

import (
	"database/sql"
	"fmt"

	"github.com/teblorum/teblorum/internal/model"
)

// MessageRepo — репозиторий личных сообщений.
type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Send(fromID, toID int64, body string) (*model.Message, error) {
	// Определяем thread_id: ищем существующий диалог
	var threadID sql.NullInt64
	err := r.db.QueryRow(`
		SELECT MIN(m.thread_id) FROM messages m
		WHERE (m.from_user_id = ? AND m.to_user_id = ?)
		   OR (m.from_user_id = ? AND m.to_user_id = ?)
	`, fromID, toID, toID, fromID).Scan(&threadID)

	if err != nil || !threadID.Valid {
		// Первое сообщение — thread_id = NULL, получим его после вставки
	}

	result, err := r.db.Exec(`
		INSERT INTO messages (from_user_id, to_user_id, thread_id, body, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
	`, fromID, toID, threadID, body)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Если это первое сообщение, устанавливаем thread_id = id
	if !threadID.Valid {
		_, err = r.db.Exec("UPDATE messages SET thread_id = ? WHERE id = ?", id, id)
		if err != nil {
			return nil, fmt.Errorf("set thread_id: %w", err)
		}
	}

	return r.GetByID(id)
}

func (r *MessageRepo) GetByID(id int64) (*model.Message, error) {
	row := r.db.QueryRow(`
		SELECT id, from_user_id, to_user_id, thread_id, body, read_at, created_at
		FROM messages WHERE id = ?
	`, id)
	msg := &model.Message{}
	var threadID, readAt sql.NullInt64
	err := row.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &threadID, &msg.Body, &readAt, &msg.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message: %w", model.ErrNotFound)
		}
		return nil, err
	}
	if threadID.Valid {
		msg.ThreadID = threadID
	}
	return msg, nil
}

// GetConversations возвращает список диалогов для пользователя.
func (r *MessageRepo) GetConversations(userID int64) ([]*model.ConversationSummary, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.from_user_id, m.to_user_id, m.thread_id, m.body, m.read_at, m.created_at
		FROM messages m
		WHERE m.id IN (
			SELECT MAX(m2.id) FROM messages m2
			WHERE m2.from_user_id = ? OR m2.to_user_id = ?
			GROUP BY COALESCE(m2.thread_id, m2.id)
		)
		ORDER BY m.created_at DESC
	`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*model.ConversationSummary
	for rows.Next() {
		msg := &model.Message{}
		var threadID, readAt sql.NullInt64
		err := rows.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &threadID, &msg.Body, &readAt, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		if threadID.Valid {
			msg.ThreadID = threadID
		}
		// Определяем собеседника
		otherID := msg.ToUserID
		if msg.ToUserID == userID {
			otherID = msg.FromUserID
		}
		summaries = append(summaries, &model.ConversationSummary{
			LastMessage: *msg,
			WithUser:    model.User{ID: otherID},
		})
	}
	return summaries, rows.Err()
}

func (r *MessageRepo) GetConversation(userID, withUserID int64, page model.PaginationParams) ([]*model.Message, error) {
	rows, err := r.db.Query(`
		SELECT id, from_user_id, to_user_id, thread_id, body, read_at, created_at
		FROM messages
		WHERE (from_user_id = ? AND to_user_id = ?)
		   OR (from_user_id = ? AND to_user_id = ?)
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, withUserID, withUserID, userID, page.Limit, page.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (r *MessageRepo) MarkRead(userID, fromUserID int64) error {
	_, err := r.db.Exec(`
		UPDATE messages SET read_at = datetime('now')
		WHERE to_user_id = ? AND from_user_id = ? AND read_at IS NULL
	`, userID, fromUserID)
	return err
}

func (r *MessageRepo) GetUnreadCount(userID int64) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM messages WHERE to_user_id = ? AND read_at IS NULL
	`, userID).Scan(&count)
	return count, err
}

func scanMessages(rows *sql.Rows) ([]*model.Message, error) {
	var msgs []*model.Message
	for rows.Next() {
		msg := &model.Message{}
		var threadID, readAt sql.NullInt64
		err := rows.Scan(&msg.ID, &msg.FromUserID, &msg.ToUserID, &threadID, &msg.Body, &readAt, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		if threadID.Valid {
			msg.ThreadID = threadID
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}
