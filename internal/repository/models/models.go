package models

import "time"

type User struct {
	UserId   string `db:"user_id"`
	Username string `db:"username"`
	TeamName string `db:"team_name"`
	IsActive bool   `db:"is_active"`
}

type TeamMember struct {
	UserId   string `db:"user_id"`
	Username string `db:"username"`
	IsActive bool   `db:"is_active"`
}

type PullRequest struct {
	PullRequestId   string     `db:"pull_request_id"`
	PullRequestName string     `db:"pull_request_name"`
	AuthorId        string     `db:"author_id"`
	Status          string     `db:"status"`
	CreatedAt       time.Time  `db:"created_at"`
	MergedAt        *time.Time `db:"merged_at"`
}
