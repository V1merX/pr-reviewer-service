package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/V1merX/pr-reviewer-service/internal/repository/models"
	"github.com/jmoiron/sqlx"
)

type TeamRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewTeamRepository(db *sqlx.DB, logger *slog.Logger) *TeamRepository {
	return &TeamRepository{
		db:  db,
		log: logger,
	}
}

const (
	qInsertTeam        = `INSERT INTO teams (team_name) VALUES ($1)`
	qUpsertUser        = `INSERT INTO users (user_id, username, team_name, is_active) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id) DO UPDATE SET username = EXCLUDED.username, team_name = EXCLUDED.team_name, is_active = EXCLUDED.is_active`
	qExistsTeam        = `SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`
	qSelectTeamsByUser = `SELECT DISTINCT team_name FROM users WHERE user_id = $1`
	qSelectTeamMembers = `SELECT u.user_id as "user_id", u.username, u.is_active FROM users u WHERE u.team_name = $1 ORDER BY u.user_id`
)

func (r *TeamRepository) withTx(fn func(*sqlx.Tx) error) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Printf("tx rollback error: %v\n", err)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

func (r *TeamRepository) CreateTeam(team api.Team) error {
	err := r.withTx(func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(qInsertTeam, team.TeamName); err != nil {
			return fmt.Errorf("db: insert team: %w", err)
		}

		for _, m := range team.Members {
			if _, err := tx.Exec(qUpsertUser, m.UserId, m.Username, team.TeamName, m.IsActive); err != nil {
				return fmt.Errorf("db: upsert user %s: %w", m.UserId, err)
			}
		}
		return nil
	})
	if err != nil {
		r.log.Error("CreateTeam failed", "team", team.TeamName, "err", err)
		return err
	}
	r.log.Info("CreateTeam succeeded", "team", team.TeamName, "members", len(team.Members))
	return nil
}

func (r *TeamRepository) UpdateTeam(team api.Team) error { return r.CreateTeam(team) }

func (r *TeamRepository) ExistTeamByName(name string) bool {
	var ok bool
	if err := r.db.QueryRow(qExistsTeam, name).Scan(&ok); err != nil {
		return false
	}
	return ok
}

func (r *TeamRepository) FindTeamByName(name string) api.Team {
	team := api.Team{TeamName: name}
	members, err := r.FindTeamMembersByName(name)
	if err != nil {
		return api.Team{}
	}
	team.Members = members
	return team
}

func (r *TeamRepository) FindTeamsByUser(userID string) ([]string, error) {
	var teams []string
	if err := r.db.Select(&teams, qSelectTeamsByUser, userID); err != nil {
		return nil, fmt.Errorf("db: select teams by user: %w", err)
	}
	return teams, nil
}

func (r *TeamRepository) FindTeamMembersByName(teamName string) ([]api.TeamMember, error) {
	var dbMembers []models.TeamMember
	if err := r.db.Select(&dbMembers, qSelectTeamMembers, teamName); err != nil {
		return nil, fmt.Errorf("db: select team members: %w", err)
	}

	members := make([]api.TeamMember, 0, len(dbMembers))
	for _, m := range dbMembers {
		members = append(members, api.TeamMember{
			UserId:   m.UserId,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	return members, nil
}
