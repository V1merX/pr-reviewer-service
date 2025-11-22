CREATE TABLE IF NOT EXISTS pr_reviewers (
  pull_request_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id),
  PRIMARY KEY(pull_request_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pr_reviewers(user_id);