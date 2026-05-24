package service

import (
	"database/sql"
	"fmt"
	"unicode/utf8"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// MessageService — бизнес-логика работы с личными сообщениями.
type MessageService struct {
	messages *repo.MessageRepo
	users    *repo.UserRepo
}

func NewMessageService(db *sql.DB) *MessageService {
	return &MessageService{
		messages: repo.NewMessageRepo(db),
		users:    repo.NewUserRepo(db),
	}
}

// Send отправляет личное сообщение.
func (s *MessageService) Send(fromID, toID int64, body string) (*model.Message, error) {
	if fromID == toID {
		return nil, fmt.Errorf("cannot send message to yourself: %w", model.ErrSelfAction)
	}

	if utf8.RuneCountInString(body) < 1 {
		return nil, fmt.Errorf("body: %w: cannot be empty", model.ErrValidation)
	}
	if utf8.RuneCountInString(body) > 5000 {
		return nil, fmt.Errorf("body: %w: too long (max 5000 characters)", model.ErrValidation)
	}

	// Проверяем существование получателя
	_, err := s.users.GetByID(toID)
	if err != nil {
		return nil, fmt.Errorf("recipient: %w", err)
	}

	return s.messages.Send(fromID, toID, body)
}

// GetConversations возвращает список диалогов пользователя.
func (s *MessageService) GetConversations(userID int64) ([]*model.ConversationSummary, error) {
	summaries, err := s.messages.GetConversations(userID)
	if err != nil {
		return nil, err
	}

	// Заполняем имена собеседников и unread count
	for _, summary := range summaries {
		otherUser, err := s.users.GetByID(summary.WithUser.ID)
		if err == nil {
			summary.WithUser = *otherUser
		}

		unread, err := s.messages.GetUnreadCountForUser(userID, summary.WithUser.ID)
		if err == nil {
			summary.UnreadCount = unread
		}
	}

	return summaries, nil
}

// GetConversation возвращает переписку с конкретным пользователем.
func (s *MessageService) GetConversation(userID, withUserID int64, page model.PaginationParams) ([]*model.Message, error) {
	if page.Limit < 1 || page.Limit > 50 {
		page.Limit = 50
	}
	if page.Page < 1 {
		page.Page = 1
	}

	// Отмечаем сообщения как прочитанные
	_ = s.messages.MarkRead(userID, withUserID)

	return s.messages.GetConversation(userID, withUserID, page)
}

// GetUnreadCount возвращает количество непрочитанных сообщений.
func (s *MessageService) GetUnreadCount(userID int64) (int, error) {
	return s.messages.GetUnreadCount(userID)
}
