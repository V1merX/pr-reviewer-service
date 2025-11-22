package team

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/repository"
)

var (
	ErrTeamNotFound = errors.New("team not found")
	ErrTeamExists   = errors.New("team already exists")
)

type Service struct {
	log  *slog.Logger
	repo repository.TeamRepository
}

func NewService(teamRepository repository.TeamRepository, logger *slog.Logger) *Service {
	return &Service{repo: teamRepository, log: logger}
}

func (s *Service) GetTeamByName(teamName string) (*api.Team, error) {
	if !s.repo.ExistTeamByName(teamName) {
		s.log.Error("GetTeamByName: team not found", "team_name", teamName)
		return nil, ErrTeamNotFound
	}
	t := s.repo.FindTeamByName(teamName)
	if t.TeamName == "" {
		s.log.Error("GetTeamByName: team not found", "team_name", teamName)
		return nil, ErrTeamNotFound
	}
	return &t, nil
}

func (s *Service) AddTeam(team *api.Team) error {
	if s.repo.ExistTeamByName(team.TeamName) {
		return ErrTeamExists
	}
	if err := s.repo.CreateTeam(*team); err != nil {
		s.log.Error("AddTeam: failed to create team", "team_name", team.TeamName, "err", err)
		return fmt.Errorf("create team: %w", err)
	}
	s.log.Info("AddTeam: team created", "team_name", team.TeamName)
	return nil
}
