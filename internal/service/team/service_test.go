package team

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/V1merX/pr-reviewer-service/internal/api"
)

type fakeTeamRepoForTest struct {
	teams     map[string]api.Team
	exist     bool
	createErr error
	created   []api.Team
}

func (f *fakeTeamRepoForTest) CreateTeam(team api.Team) error {
	f.created = append(f.created, team)
	return f.createErr
}
func (f *fakeTeamRepoForTest) UpdateTeam(team api.Team) error                  { return nil }
func (f *fakeTeamRepoForTest) ExistTeamByName(name string) bool                { return f.exist }
func (f *fakeTeamRepoForTest) FindTeamByName(name string) api.Team             { return f.teams[name] }
func (f *fakeTeamRepoForTest) FindTeamsByUser(userID string) ([]string, error) { return nil, nil }
func (f *fakeTeamRepoForTest) FindTeamMembersByName(teamName string) ([]api.TeamMember, error) {
	return nil, nil
}

func TestAddTeam_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name    string
		repo    *fakeTeamRepoForTest
		team    api.Team
		wantErr error
	}{
		{"exists", &fakeTeamRepoForTest{exist: true}, api.Team{TeamName: "t1"}, ErrTeamExists},
		{"create fails", &fakeTeamRepoForTest{exist: false, createErr: errors.New("db")}, api.Team{TeamName: "t2"}, errors.New("create team")},
		{"success", &fakeTeamRepoForTest{exist: false}, api.Team{TeamName: "t3"}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(tc.repo, logger)
			err := svc.AddTeam(&tc.team)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.name == "success" {
				if len(tc.repo.created) != 1 {
					t.Fatalf("expected CreateTeam to be called")
				}
			}
		})
	}
	_ = logger
}

func TestGetTeamByName_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := &fakeTeamRepoForTest{exist: true, teams: map[string]api.Team{"t1": {TeamName: "t1"}}}
	svc := NewService(repo, logger)

	_, err := svc.GetTeamByName("t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo2 := &fakeTeamRepoForTest{exist: false}
	svc2 := NewService(repo2, logger)
	_, err = svc2.GetTeamByName("tX")
	if err == nil {
		t.Fatalf("expected error for missing team")
	}
}
