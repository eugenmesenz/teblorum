package service

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/teblorum/teblorum/internal/model"
	"github.com/teblorum/teblorum/internal/repo"
)

// ModerationService — бизнес-логика модерации.
type ModerationService struct {
	users    *repo.UserRepo
	sessions *repo.SessionRepo
}

func NewModerationService(db *sql.DB) *ModerationService {
	return &ModerationService{
		users:    repo.NewUserRepo(db),
		sessions: repo.NewSessionRepo(db),
	}
}

// BanUser банит пользователя.
func (s *ModerationService) BanUser(targetID, actorID int64, duration time.Duration) error {
	if targetID == actorID {
		return fmt.Errorf("cannot ban yourself: %w", model.ErrSelfAction)
	}

	actor, err := s.users.GetByID(actorID)
	if err != nil {
		return err
	}

	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}

	if !model.CanActOn(actor.Role, target.Role) {
		return fmt.Errorf("ban user: %w", model.ErrForbidden)
	}

	until := time.Now().Add(duration)

	if err := s.users.SetBan(targetID, until, actorID); err != nil {
		return fmt.Errorf("set ban: %w", err)
	}

	// Удаляем все сессии забаненного
	_ = s.sessions.DeleteByUserID(targetID)

	return nil
}

// UnbanUser снимает бан с пользователя.
func (s *ModerationService) UnbanUser(targetID, actorID int64) error {
	actor, err := s.users.GetByID(actorID)
	if err != nil {
		return err
	}

	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}

	if !model.CanActOn(actor.Role, target.Role) {
		return fmt.Errorf("unban user: %w", model.ErrForbidden)
	}

	return s.users.RemoveBan(targetID)
}

// PromoteToModerator назначает пользователя модератором (только root).
func (s *ModerationService) PromoteToModerator(targetID, actorID int64) error {
	if targetID == actorID {
		return fmt.Errorf("cannot promote yourself: %w", model.ErrSelfAction)
	}

	actor, err := s.users.GetByID(actorID)
	if err != nil {
		return err
	}

	if actor.Role != model.RoleRoot {
		return fmt.Errorf("promote: %w: only root can promote", model.ErrForbidden)
	}

	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}

	if target.Role != model.RoleUser {
		return fmt.Errorf("promote: %w: can only promote users", model.ErrValidation)
	}

	return s.users.UpdateRole(targetID, model.RoleModerator)
}

// DemoteFromModerator снимает роль модератора (только root).
func (s *ModerationService) DemoteFromModerator(targetID, actorID int64) error {
	if targetID == actorID {
		return fmt.Errorf("cannot demote yourself: %w", model.ErrSelfAction)
	}

	actor, err := s.users.GetByID(actorID)
	if err != nil {
		return err
	}

	if actor.Role != model.RoleRoot {
		return fmt.Errorf("demote: %w: only root can demote", model.ErrForbidden)
	}

	target, err := s.users.GetByID(targetID)
	if err != nil {
		return err
	}

	if target.Role != model.RoleModerator {
		return fmt.Errorf("demote: %w: can only demote moderators", model.ErrValidation)
	}

	return s.users.UpdateRole(targetID, model.RoleUser)
}