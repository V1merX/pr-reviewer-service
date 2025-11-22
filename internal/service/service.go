package service

import "github.com/V1merX/pr-reviewer-service/internal/api"

type UserService interface {
	GetUserByID(userID string) (*api.User, error)
	SetUserStatus(userID string, status bool) (*api.User, error)
}

type PullRequestService interface {
	GetActiveTeamMembers(authorID string) ([]api.TeamMember, error)
	SelectRandomReviewers(members []api.TeamMember, count int) []string
	CreatePR(pr *api.PullRequest) error
	FindPRByID(prID string) (*api.PullRequest, error)
	MergePR(prID string) (*api.PullRequest, error)
	FindPRsByReviewer(userID string) ([]api.PullRequest, error)
	ReassignReviewer(prID string, oldReviewerID string) (*api.PullRequest, *string, error)
	GetStatistics() (*api.Statistics, error)
	DeactivateUsersAndReassignPRs(teamName string, userIDs []string) (*api.BatchDeactivateResponse, error)
}

type TeamService interface {
	GetTeamByName(teamName string) (*api.Team, error)
	AddTeam(team *api.Team) error
}
