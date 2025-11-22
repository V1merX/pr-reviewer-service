package pullrequest

import (
	"io"
	"log/slog"
	"testing"

	"github.com/V1merX/pr-reviewer-service/internal/api"
)

type fakeUserRepo struct {
	users map[string]api.User
}

func (f *fakeUserRepo) FindUserByID(userID string) (*api.User, error) {
	u, ok := f.users[userID]
	if !ok {
		return nil, repositoryError("not found")
	}
	return &u, nil
}
func (f *fakeUserRepo) UpdateUserStatus(userID string, status bool) error { return nil }
func (f *fakeUserRepo) GetAllUsers() ([]api.User, error)                  { return nil, nil }

type fakeTeamRepo struct {
	members map[string][]api.TeamMember
}

func (f *fakeTeamRepo) CreateTeam(team api.Team) error                  { return nil }
func (f *fakeTeamRepo) UpdateTeam(team api.Team) error                  { return nil }
func (f *fakeTeamRepo) ExistTeamByName(name string) bool                { return false }
func (f *fakeTeamRepo) FindTeamByName(name string) api.Team             { return api.Team{} }
func (f *fakeTeamRepo) FindTeamsByUser(userID string) ([]string, error) { return nil, nil }
func (f *fakeTeamRepo) FindTeamMembersByName(teamName string) ([]api.TeamMember, error) {
	return f.members[teamName], nil
}

type fakePRRepo struct {
	created       []api.PullRequest
	prsByReviewer map[string][]api.PullRequest
	updated       []api.PullRequest
}

func (f *fakePRRepo) CreatePR(pr api.PullRequest) error {
	f.created = append(f.created, pr)
	return nil
}

func (f *fakePRRepo) FindPRByID(prID string) (*api.PullRequest, error) {
	return nil, repositoryError("not found")
}

func (f *fakePRRepo) UpdatePR(pr api.PullRequest) error {
	f.updated = append(f.updated, pr)
	return nil
}

func (f *fakePRRepo) FindPRsByReviewer(userID string) ([]api.PullRequest, error) {
	return f.prsByReviewer[userID], nil
}
func (f *fakePRRepo) GetAllPRs() ([]api.PullRequest, error) { return nil, nil }

type repositoryError string

func (e repositoryError) Error() string { return string(e) }

func TestSelectRandomReviewers_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(logger, nil, nil, nil)

	cases := []struct {
		name    string
		members []api.TeamMember
		count   int
		wantLen int
	}{
		{"no members", nil, 2, 0},
		{"one member, request2", []api.TeamMember{{UserId: "u1", IsActive: true}}, 2, 1},
		{"three members, request2", []api.TeamMember{{UserId: "a"}, {UserId: "b"}, {UserId: "c"}}, 2, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.SelectRandomReviewers(tc.members, tc.count)
			if len(got) != tc.wantLen {
				t.Fatalf("want len %d got %d", tc.wantLen, len(got))
			}
		})
	}
}

func TestCreatePR_Success(t *testing.T) {
	urepo := &fakeUserRepo{users: map[string]api.User{"author": {UserId: "author", TeamName: "team1"}}}
	trepo := &fakeTeamRepo{members: map[string][]api.TeamMember{"team1": {{UserId: "u1", IsActive: true}, {UserId: "u2", IsActive: true}}}}
	prrepo := &fakePRRepo{}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(logger, prrepo, trepo, urepo)

	pr := &api.PullRequest{PullRequestId: "pr1", PullRequestName: "PR 1", AuthorId: "author"}
	if err := svc.CreatePR(pr); err != nil {
		t.Fatalf("CreatePR failed: %v", err)
	}
	if len(prrepo.created) != 1 {
		t.Fatalf("expected created PR recorded, got %d", len(prrepo.created))
	}
	created := prrepo.created[0]
	if created.PullRequestId != "pr1" {
		t.Fatalf("created id mismatch")
	}
	if len(created.AssignedReviewers) == 0 {
		t.Fatalf("expected assigned reviewers")
	}
}
