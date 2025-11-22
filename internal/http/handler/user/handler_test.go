package user

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	psvc "github.com/V1merX/pr-reviewer-service/internal/service/pullrequest"
)

// Заглушки для сервисов
type fakeUserSvc struct{}

func (f *fakeUserSvc) GetUserByID(userID string) (*api.User, error) {
	return &api.User{UserId: userID}, nil
}

func (f *fakeUserSvc) SetUserStatus(userID string, status bool) (*api.User, error) {
	return &api.User{UserId: userID, IsActive: status}, nil
}

type fakePRSvc struct {
	prs              map[string][]api.PullRequest
	deactivateResult *api.BatchDeactivateResponse
	deactivateErr    error
}

func (f *fakePRSvc) GetActiveTeamMembers(authorID string) ([]api.TeamMember, error) {
	return nil, nil
}

func (f *fakePRSvc) SelectRandomReviewers(members []api.TeamMember, count int) []string {
	return nil
}

func (f *fakePRSvc) CreatePR(pr *api.PullRequest) error {
	return nil
}

func (f *fakePRSvc) FindPRByID(prID string) (*api.PullRequest, error) {
	return nil, nil
}

func (f *fakePRSvc) MergePR(prID string) (*api.PullRequest, error) {
	return nil, nil
}

func (f *fakePRSvc) ReassignReviewer(prID, oldReviewerID string) (*api.PullRequest, *string, error) {
	return nil, nil, nil
}

func (f *fakePRSvc) FindPRsByReviewer(userID string) ([]api.PullRequest, error) {
	return f.prs[userID], nil
}

func (f *fakePRSvc) GetStatistics() (*api.Statistics, error) {
	return nil, nil
}

func (f *fakePRSvc) DeactivateUsersAndReassignPRs(teamName string, userIDs []string) (*api.BatchDeactivateResponse, error) {
	return f.deactivateResult, f.deactivateErr
}

func TestGetUsersGetReview_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prsvc := &fakePRSvc{prs: map[string][]api.PullRequest{"u1": {{PullRequestId: "p1", PullRequestName: "P1", AuthorId: "a1", Status: api.PullRequestStatusOPEN}}}}
	h := New(&fakeUserSvc{}, prsvc)

	cases := []struct {
		name       string
		userID     string
		wantStatus int
	}{
		{"missing user id", "", http.StatusBadRequest},
		{"existing user id", "u1", http.StatusOK},
		{"no prs", "nouser", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id="+tc.userID, nil)
			w := httptest.NewRecorder()
			params := api.GetUsersGetReviewParams{UserId: tc.userID}
			h.GetUsersGetReview(w, req, params)
			res := w.Result()
			if res.StatusCode != tc.wantStatus {
				t.Fatalf("want status %d got %d", tc.wantStatus, res.StatusCode)
			}
		})
	}
	_ = logger
}

func TestPostUsersDeactivateBatch_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	prsvc := &fakePRSvc{deactivateResult: &api.BatchDeactivateResponse{DeactivatedCount: 1, ReassignedCount: 1}}
	h := New(&fakeUserSvc{}, prsvc)

	reqBody := api.BatchDeactivateRequest{TeamName: "team1", UserIds: []string{"u1"}}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/users/deactivateBatch", bytes.NewReader(b))
	w := httptest.NewRecorder()
	h.PostUsersDeactivateBatch(w, req)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Result().StatusCode)
	}

	prsvc2 := &fakePRSvc{deactivateErr: psvc.ErrTeamNotFound}
	h2 := New(&fakeUserSvc{}, prsvc2)
	req2 := httptest.NewRequest(http.MethodPost, "/users/deactivateBatch", bytes.NewReader(b))
	w2 := httptest.NewRecorder()
	h2.PostUsersDeactivateBatch(w2, req2)
	if w2.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w2.Result().StatusCode)
	}
	_ = logger
}
