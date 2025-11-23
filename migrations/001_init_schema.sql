CREATE TABLE IF NOT EXISTS teams(
    team_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users(
        user_id VARCHAR(255) PRIMARY KEY,
        user_name VARCHAR(255) NOT NULL,
        team_name REFERENCES teams(team_name) ON DELETE CASCADE,
        is_active BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pull_requests(
    pull_requests_id VARCHAR(255) PRIMARY KEY,
    pull_requests_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) REFERENCES users(user_id),
    status VARCHAR(255) DEFAULT 'OPEN',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    merged_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS PR_reviewers(
    pr_id VARCHAR(255) REFERENCES pull_requests(pull_requests_id) ON DELETE CASCADE,
    reviewer_id VARCHAR(255) REFERENCES users(user_id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pr_id,reviewer_id)
);

CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active);

CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);

CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id)