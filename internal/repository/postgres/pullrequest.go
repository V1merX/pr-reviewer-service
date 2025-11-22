package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/V1merX/pr-reviewer-service/internal/api"
	"github.com/jmoiron/sqlx"
)

type PullRequestRepository struct {
	db  *sqlx.DB
	log *slog.Logger
}

func NewPullRequestRepository(db *sqlx.DB, logger *slog.Logger) *PullRequestRepository {
	return &PullRequestRepository{
		db:  db,
		log: logger,
	}
}

const (
	qSelectPRByID        = `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at FROM pull_requests WHERE pull_request_id = $1`
	qSelectAllPRs        = `SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at FROM pull_requests ORDER BY created_at DESC`
	qSelectPRsByReviewer = `SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at FROM pull_requests pr WHERE pr.pull_request_id IN (SELECT pull_request_id FROM pr_reviewers WHERE user_id = $1) ORDER BY pr.created_at DESC`
	qSelectReviewers     = `SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1`
)

func (r *PullRequestRepository) withTx(fn func(*sqlx.Tx) error) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
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
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (r *PullRequestRepository) CreatePR(pr api.PullRequest) error {
	createdAt := time.Now()
	if pr.CreatedAt != nil {
		createdAt = *pr.CreatedAt
	}

	err := r.withTx(func(tx *sqlx.Tx) error {
		if _, err := tx.Exec(`INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) VALUES ($1, $2, $3, $4, $5)`, pr.PullRequestId, pr.PullRequestName, pr.AuthorId, pr.Status, createdAt); err != nil {
			return fmt.Errorf("insert pull_request: %w", err)
		}

		for _, reviewer := range pr.AssignedReviewers {
			if _, err := tx.Exec(`INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`, pr.PullRequestId, reviewer); err != nil {
				return fmt.Errorf("insert reviewer: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		r.log.Error("CreatePR failed", "pr_id", pr.PullRequestId, "err", err)
		return err
	}
	r.log.Info("CreatePR succeeded", "pr_id", pr.PullRequestId, "author", pr.AuthorId)
	return nil
}

func (r *PullRequestRepository) scanRowToPR(scanner interface{ Scan(dest ...any) error }) (api.PullRequest, error) {
	var pr api.PullRequest
	var createdAt time.Time
	var mergedAt *time.Time

	if err := scanner.Scan(&pr.PullRequestId, &pr.PullRequestName, &pr.AuthorId, &pr.Status, &createdAt, &mergedAt); err != nil {
		r.log.Error("scanRowToPR: scan failed", "err", err)
		return api.PullRequest{}, fmt.Errorf("scan pr: %w", err)
	}

	pr.CreatedAt = &createdAt
	pr.MergedAt = mergedAt

	reviewers := []string{}
	if err := r.db.Select(&reviewers, qSelectReviewers, pr.PullRequestId); err != nil {
		r.log.Error("scanRowToPR: select reviewers failed", "pr_id", pr.PullRequestId, "err", err)
		return api.PullRequest{}, fmt.Errorf("select reviewers: %w", err)
	}
	pr.AssignedReviewers = reviewers

	return pr, nil
}

func (r *PullRequestRepository) FindPRByID(prID string) (*api.PullRequest, error) {
	row := r.db.QueryRowx(qSelectPRByID, prID)
	pr, err := r.scanRowToPR(row)
	if err != nil {
		r.log.Error("FindPRByID failed", "pr_id", prID, "err", err)
		return nil, fmt.Errorf("find pr by id: %w", err)
	}
	return &pr, nil
}

func (r *PullRequestRepository) UpdatePR(pr api.PullRequest) error {
	err := r.withTx(func(tx *sqlx.Tx) error {
		var mergedAt any
		if pr.MergedAt != nil {
			mergedAt = pr.MergedAt
		}

		if _, err := tx.Exec(`UPDATE pull_requests SET status = $1, merged_at = $2 WHERE pull_request_id = $3`, pr.Status, mergedAt, pr.PullRequestId); err != nil {
			return fmt.Errorf("update pull_request: %w", err)
		}

		if _, err := tx.Exec(`DELETE FROM pr_reviewers WHERE pull_request_id = $1`, pr.PullRequestId); err != nil {
			return fmt.Errorf("delete old reviewers: %w", err)
		}

		for _, reviewer := range pr.AssignedReviewers {
			if _, err := tx.Exec(`INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)`, pr.PullRequestId, reviewer); err != nil {
				return fmt.Errorf("insert reviewer: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		r.log.Error("UpdatePR failed", "pr_id", pr.PullRequestId, "err", err)
		return err
	}
	r.log.Info("UpdatePR succeeded", "pr_id", pr.PullRequestId)
	return nil
}

func (r *PullRequestRepository) FindPRsByReviewer(userID string) ([]api.PullRequest, error) {
	rows, err := r.db.Queryx(qSelectPRsByReviewer, userID)
	if err != nil {
		return nil, fmt.Errorf("query prs by reviewer: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("rows close error: %v\n", err)
		}
	}()

	var results []api.PullRequest
	for rows.Next() {
		pr, err := r.scanRowToPR(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return results, nil
}

func (r *PullRequestRepository) GetAllPRs() ([]api.PullRequest, error) {
	rows, err := r.db.Queryx(qSelectAllPRs)
	if err != nil {
		return nil, fmt.Errorf("query all prs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			fmt.Printf("rows close error: %v\n", err)
		}
	}()

	var results []api.PullRequest
	for rows.Next() {
		pr, err := r.scanRowToPR(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return results, nil
}
