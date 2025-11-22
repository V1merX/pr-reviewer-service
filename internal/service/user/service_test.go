package user

import (
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/V1merX/pr-reviewer-service/internal/api"
)

type fakeUserRepoForTest struct {
	users        map[string]api.User
	updateErr    error
	updatedCalls []struct {
		UserID string
		Status bool
	}
}

func (f *fakeUserRepoForTest) FindUserByID(userID string) (*api.User, error) {
	u, ok := f.users[userID]
	if !ok {
		return nil, errors.New("not found")
	}
	return &u, nil
}

func (f *fakeUserRepoForTest) UpdateUserStatus(userID string, status bool) error {
	f.updatedCalls = append(f.updatedCalls, struct {
		UserID string
		Status bool
	}{UserID: userID, Status: status})
	return f.updateErr
}
func (f *fakeUserRepoForTest) GetAllUsers() ([]api.User, error) { return nil, nil }

func TestSetUserStatus_Table(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name    string
		repo    *fakeUserRepoForTest
		userID  string
		status  bool
		wantErr error
	}{
		{"user not found", &fakeUserRepoForTest{users: map[string]api.User{}}, "u1", false, ErrUserNotFound},
		{"update failed", &fakeUserRepoForTest{users: map[string]api.User{"u1": {UserId: "u1"}}, updateErr: errors.New("db")}, "u1", true, errors.New("update user status")},
		{"success", &fakeUserRepoForTest{users: map[string]api.User{"u1": {UserId: "u1", IsActive: false}}}, "u1", true, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewService(logger, tc.repo)
			u, err := svc.SetUserStatus(tc.userID, tc.status)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// for update failed case we expect wrapped error
				if tc.name == "update failed" {
					// expect error to wrap underlying repository error
					if !errors.Is(err, tc.repo.updateErr) {
						t.Fatalf("expected wrapped error to contain repo error, got: %v", err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u == nil {
				t.Fatalf("expected user returned")
			}
			if u.IsActive != tc.status {
				t.Fatalf("user active mismatch: want %v got %v", tc.status, u.IsActive)
			}
			// ensure UpdateUserStatus called
			if len(tc.repo.updatedCalls) == 0 {
				t.Fatalf("expected UpdateUserStatus to be called")
			}
		})
	}
	_ = logger
}
