package repository

import "github.com/V1merX/pr-reviewer-service/internal/api"

type UserRepository interface {
	FindUserByID(userID string) (*api.User, error)
	UpdateUserStatus(userID string, status bool) error
	GetAllUsers() ([]api.User, error)
}

type PullRequestRepository interface {
	CreatePR(pr api.PullRequest) error
	FindPRByID(prID string) (*api.PullRequest, error)
	UpdatePR(pr api.PullRequest) error
	FindPRsByReviewer(userID string) ([]api.PullRequest, error)
	GetAllPRs() ([]api.PullRequest, error)
}

type TeamRepository interface {
	CreateTeam(team api.Team) error
	UpdateTeam(team api.Team) error
	ExistTeamByName(name string) bool
	FindTeamByName(name string) api.Team
	FindTeamsByUser(userID string) ([]string, error)
	FindTeamMembersByName(teamName string) ([]api.TeamMember, error)
}
