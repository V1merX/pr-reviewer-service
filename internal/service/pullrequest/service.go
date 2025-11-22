package pullrequest

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/repository"
)

type Service struct {
	log                   *slog.Logger
	pullRequestRepository repository.PullRequestRepository
	teamRepository        repository.TeamRepository
	userRepository        repository.UserRepository
}

func NewService(
	log *slog.Logger,
	pullRequestRepository repository.PullRequestRepository,
	teamRepository repository.TeamRepository,
	userRepository repository.UserRepository,
) *Service {
	return &Service{
		log:                   log,
		pullRequestRepository: pullRequestRepository,
		teamRepository:        teamRepository,
		userRepository:        userRepository,
	}
}

func randomIndex(n int) (int, error) {
	maxN := big.NewInt(int64(n))
	num, err := rand.Int(rand.Reader, maxN)
	if err != nil {
		return 0, err
	}
	return int(num.Int64()), nil
}

var (
	ErrAuthorNotFound               = errors.New("author not found")
	ErrAuthorHasNoTeam              = errors.New("author has no team")
	ErrPRNotFound                   = errors.New("PR not found")
	ErrCannotReassignOnMergedPR     = errors.New("cannot reassign on merged PR")
	ErrReviewerNotAssigned          = errors.New("reviewer is not assigned to this PR")
	ErrNoReplacementCandidateInTeam = errors.New("no active replacement candidate in team")
	ErrTeamNotFound                 = errors.New("team not found")
)

func (s *Service) GetActiveTeamMembers(authorID string) ([]api.TeamMember, error) {
	author, err := s.userRepository.FindUserByID(authorID)
	if err != nil {
		return nil, ErrAuthorNotFound
	}

	if author.TeamName == "" {
		return nil, ErrAuthorHasNoTeam
	}

	members, err := s.teamRepository.FindTeamMembersByName(author.TeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	active := make([]api.TeamMember, 0, len(members))
	for _, m := range members {
		if m.IsActive && m.UserId != authorID {
			active = append(active, m)
		}
	}

	return active, nil
}

func (s *Service) SelectRandomReviewers(members []api.TeamMember, count int) []string {
	if len(members) == 0 || count <= 0 {
		return nil
	}
	if count > 2 {
		count = 2
	}
	if count > len(members) {
		count = len(members)
	}

	idxs := make([]int, len(members))
	for i := range idxs {
		idxs[i] = i
	}

	for i := len(idxs) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			break
		}
		j := int(jBig.Int64())
		idxs[i], idxs[j] = idxs[j], idxs[i]
	}

	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, members[idxs[i]].UserId)
	}
	return out
}

func (s *Service) CreatePR(pr *api.PullRequest) error {
	activeMembers, err := s.GetActiveTeamMembers(pr.AuthorId)
	if err != nil {
		return err
	}

	reviewers := s.SelectRandomReviewers(activeMembers, 2)
	pr.AssignedReviewers = reviewers
	pr.Status = api.PullRequestStatusOPEN
	now := time.Now()
	pr.CreatedAt = &now

	if err := s.pullRequestRepository.CreatePR(*pr); err != nil {
		s.log.Error("CreatePR failed", "pr_id", pr.PullRequestId, "author", pr.AuthorId, "err", err)
		return err
	}
	s.log.Info("PR created", "pr_id", pr.PullRequestId, "author", pr.AuthorId, "reviewers", pr.AssignedReviewers)
	return nil
}

func (s *Service) FindPRByID(prID string) (*api.PullRequest, error) {
	return s.pullRequestRepository.FindPRByID(prID)
}

func (s *Service) MergePR(prID string) (*api.PullRequest, error) {
	pr, err := s.pullRequestRepository.FindPRByID(prID)
	if err != nil {
		return nil, ErrPRNotFound
	}

	if pr.Status != api.PullRequestStatusMERGED {
		pr.Status = api.PullRequestStatusMERGED
		now := time.Now()
		pr.MergedAt = &now
		err = s.pullRequestRepository.UpdatePR(*pr)
		if err != nil {
			s.log.Error("MergePR: update failed", "pr_id", prID, "err", err)
			return nil, err
		}
		s.log.Info("PR merged", "pr_id", prID, "merged_at", pr.MergedAt)
	}

	return pr, nil
}

func (s *Service) ReassignReviewer(prID string, oldReviewerID string) (*api.PullRequest, *string, error) {
	pr, err := s.pullRequestRepository.FindPRByID(prID)
	if err != nil {
		return nil, nil, ErrPRNotFound
	}

	if pr.Status == api.PullRequestStatusMERGED {
		return nil, nil, ErrCannotReassignOnMergedPR
	}

	found := false
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == oldReviewerID {
			found = true
			break
		}
	}
	if !found {
		return nil, nil, ErrReviewerNotAssigned
	}

	oldReviewer, err := s.userRepository.FindUserByID(oldReviewerID)
	if err != nil {
		return nil, nil, fmt.Errorf("reviewer not found")
	}

	members, err := s.teamRepository.FindTeamMembersByName(oldReviewer.TeamName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get team members: %w", err)
	}

	var candidates []api.TeamMember
	for _, member := range members {
		if !member.IsActive {
			continue
		}
		if member.UserId == oldReviewerID {
			continue
		}
		alreadyReviewer := false
		for _, reviewer := range pr.AssignedReviewers {
			if reviewer == member.UserId {
				alreadyReviewer = true
				break
			}
		}
		if !alreadyReviewer {
			candidates = append(candidates, member)
		}
	}

	if len(candidates) == 0 {
		return nil, nil, ErrNoReplacementCandidateInTeam
	}

	idx, err := randomIndex(len(candidates))
	if err != nil {
		return nil, nil, err
	}
	newReviewer := candidates[idx].UserId

	newReviewers := []string{}
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer != oldReviewerID {
			newReviewers = append(newReviewers, reviewer)
		}
	}
	newReviewers = append(newReviewers, newReviewer)
	pr.AssignedReviewers = newReviewers

	err = s.pullRequestRepository.UpdatePR(*pr)
	if err != nil {
		s.log.Error("ReassignReviewer: update failed", "pr_id", prID, "old_reviewer", oldReviewerID, "err", err)
		return nil, nil, err
	}
	s.log.Info("Reviewer reassigned", "pr_id", prID, "from", oldReviewerID, "to", newReviewer)

	return pr, &newReviewer, nil
}

func (s *Service) FindPRsByReviewer(userID string) ([]api.PullRequest, error) {
	return s.pullRequestRepository.FindPRsByReviewer(userID)
}

func (s *Service) GetStatistics() (*api.Statistics, error) {
	stats := &api.Statistics{
		TotalAssignments: 0,
		ByUser:           make(map[string]int),
		ByStatus: struct {
			Open   int `json:"open"`
			Merged int `json:"merged"`
		}{},
	}

	prs, err := s.pullRequestRepository.GetAllPRs()
	if err != nil {
		s.log.Error("GetStatistics: failed to get PRs", "err", err)
		return stats, err
	}

	for _, pr := range prs {
		for _, reviewer := range pr.AssignedReviewers {
			stats.TotalAssignments++
			stats.ByUser[reviewer]++
		}

		switch pr.Status {
		case api.PullRequestStatusOPEN:
			stats.ByStatus.Open++
		case api.PullRequestStatusMERGED:
			stats.ByStatus.Merged++
		}
	}

	return stats, nil
}

func (s *Service) DeactivateUsersAndReassignPRs(teamName string, userIDs []string) (*api.BatchDeactivateResponse, error) {
	team := s.teamRepository.FindTeamByName(teamName)
	if team.TeamName == "" {
		s.log.Error("DeactivateUsersAndReassignPRs: team not found", "team", teamName)
		return nil, ErrTeamNotFound
	}

	response := &api.BatchDeactivateResponse{
		DeactivatedCount: 0,
		ReassignedCount:  0,
		Errors: []struct {
			UserID string `json:"user_id"`
			Error  string `json:"error"`
		}{},
	}

	userIDMap := make(map[string]bool)
	for _, userID := range userIDs {
		userIDMap[userID] = true
	}

	var activeReplacements []string
	allUsers, err := s.userRepository.GetAllUsers()
	if err != nil {
		s.log.Error("DeactivateUsersAndReassignPRs: failed to list users", "team", teamName, "err", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	for _, user := range allUsers {
		if user.TeamName == teamName && user.IsActive && !userIDMap[user.UserId] {
			activeReplacements = append(activeReplacements, user.UserId)
		}
	}

	if len(activeReplacements) == 0 {
		s.log.Error("DeactivateUsersAndReassignPRs: no active replacements", "team", teamName)
		return nil, ErrNoReplacementCandidateInTeam
	}

	for _, userID := range userIDs {
		err := s.userRepository.UpdateUserStatus(userID, false)
		if err != nil {
			s.log.Error("DeactivateUsersAndReassignPRs: failed to deactivate user", "team", teamName, "user", userID, "err", err)
			response.Errors = append(response.Errors, struct {
				UserID string `json:"user_id"`
				Error  string `json:"error"`
			}{UserID: userID, Error: "failed to deactivate"})
			continue
		}
		s.log.Info("User deactivated", "team", teamName, "user", userID)
		response.DeactivatedCount++

		prs, err := s.pullRequestRepository.FindPRsByReviewer(userID)
		if err != nil {
			response.Errors = append(response.Errors, struct {
				UserID string `json:"user_id"`
				Error  string `json:"error"`
			}{UserID: userID, Error: fmt.Sprintf("failed to list PRs: %v", err)})
			continue
		}
		for _, pr := range prs {
			if pr.Status != api.PullRequestStatusOPEN {
				continue
			}

			var newReviewers []string
			for _, rev := range pr.AssignedReviewers {
				if rev != userID {
					newReviewers = append(newReviewers, rev)
				}
			}

			if len(newReviewers) < 2 && len(activeReplacements) > 0 {
				replacementIndex, err := randomIndex(len(activeReplacements))
				if err != nil {
					response.Errors = append(response.Errors, struct {
						UserID string `json:"user_id"`
						Error  string `json:"error"`
					}{UserID: userID, Error: fmt.Sprintf("random selection failed: %v", err)})
					s.log.Error("DeactivateUsersAndReassignPRs: randomIndex failed", "team", teamName, "user", userID, "err", err)
					continue
				}
				replacement := activeReplacements[replacementIndex]
				newReviewers = append(newReviewers, replacement)
			}

			pr.AssignedReviewers = newReviewers
			err := s.pullRequestRepository.UpdatePR(pr)
			if err != nil {
				s.log.Error("DeactivateUsersAndReassignPRs: failed to update PR", "pr_id", pr.PullRequestId, "err", err)
				return nil, err
			}
			s.log.Info("PR reviewers updated after deactivation", "pr_id", pr.PullRequestId)
			response.ReassignedCount++
		}
	}
	s.log.Info("DeactivateUsersAndReassignPRs finished", "team", teamName, "deactivated", response.DeactivatedCount, "reassigned", response.ReassignedCount)
	return response, nil
}
