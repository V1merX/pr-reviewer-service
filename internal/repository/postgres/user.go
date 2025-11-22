package postgres

import (
	"fmt"
	"log/slog"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/repository/models"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewUserRepository(db *sqlx.DB, logger *slog.Logger) *UserRepository {
	return &UserRepository{
		db:  db,
		log: logger,
	}
}

func (r *UserRepository) FindUserByID(userID string) (*api.User, error) {
	var u models.User
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`
	if err := r.db.Get(&u, query, userID); err != nil {
		return nil, fmt.Errorf("db: get user: %w", err)
	}
	user := api.User{
		UserId:   u.UserId,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
	return &user, nil
}

func (r *UserRepository) UpdateUserStatus(userID string, status bool) error {
	res, err := r.db.Exec("UPDATE users SET is_active = $1 WHERE user_id = $2", status, userID)
	if err != nil {
		return fmt.Errorf("db: update user status: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("db: user not found")
	}
	r.log.Info("UpdateUserStatus succeeded", "user", userID, "is_active", status)
	return nil
}

func (r *UserRepository) GetAllUsers() ([]api.User, error) {
	var dbUsers []models.User
	query := `SELECT user_id, username, team_name, is_active FROM users`
	if err := r.db.Select(&dbUsers, query); err != nil {
		return nil, fmt.Errorf("db: select users: %w", err)
	}
	users := make([]api.User, 0, len(dbUsers))
	for _, u := range dbUsers {
		users = append(users, api.User{
			UserId:   u.UserId,
			Username: u.Username,
			TeamName: u.TeamName,
			IsActive: u.IsActive,
		})
	}
	return users, nil
}
