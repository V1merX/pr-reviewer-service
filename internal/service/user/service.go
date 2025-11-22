package user

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/repository"
)

var ErrUserNotFound = errors.New("user not found")

type Service struct {
	log            *slog.Logger
	userRepository repository.UserRepository
}

func NewService(log *slog.Logger, userRepository repository.UserRepository) *Service {
	return &Service{log: log, userRepository: userRepository}
}

func (s *Service) GetUserByID(userID string) (*api.User, error) {
	return s.userRepository.FindUserByID(userID)
}

func (s *Service) SetUserStatus(userID string, status bool) (*api.User, error) {
	u, err := s.userRepository.FindUserByID(userID)
	if err != nil {
		s.log.Error("SetUserStatus: user not found", "user_id", userID, "err", err)
		return nil, ErrUserNotFound
	}

	if err := s.userRepository.UpdateUserStatus(userID, status); err != nil {
		s.log.Error("SetUserStatus: failed to update user status", "user_id", userID, "status", status, "err", err)
		return nil, fmt.Errorf("update user status: %w", err)
	}

	u.IsActive = status
	s.log.Info("SetUserStatus: user status updated", "user_id", userID, "status", status)
	return u, nil
}
