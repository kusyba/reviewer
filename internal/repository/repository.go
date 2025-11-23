package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"pr-reviewer-service/internal/models"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var (
	ErrTeamExists    = errors.New("team already exists")
	ErrPRExists      = errors.New("PR already exists")
	ErrPRMerged      = errors.New("PR is merged")
	ErrNotAssigned   = errors.New("reviewer not assigned")
	ErrNoCandidate   = errors.New("no active candidate available")
	ErrNotFound      = errors.New("resource not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTeam(team *models.Team) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingTeam string
	err = tx.QueryRow("SELECT team_name FROM teams WHERE team_name = $1", team.TeamName).Scan(&existingTeam)
	if err == nil {
		return ErrTeamExists
	} else if err != sql.ErrNoRows {
		return err
	}

	_, err = tx.Exec("INSERT INTO teams (team_name) VALUES ($1)", team.TeamName)
	if err != nil {
		return err
	}

	for _, member := range team.Members {
		_, err = tx.Exec(`
			INSERT INTO users (user_id, username, team_name, is_active) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) 
			DO UPDATE SET username = $2, team_name = $3, is_active = $4
		`, member.UserID, member.Username, team.TeamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetTeam(teamName string) (*models.Team, error) {
	var team models.Team
	team.TeamName = teamName

	rows, err := r.db.Query(`
		SELECT user_id, username, is_active 
		FROM users 
		WHERE team_name = $1
	`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member models.TeamMember
		err := rows.Scan(&member.UserID, &member.Username, &member.IsActive)
		if err != nil {
			return nil, err
		}
		team.Members = append(team.Members, member)
	}

	if len(team.Members) == 0 {
		return nil, ErrNotFound
	}

	return &team, nil
}

func (r *Repository) SetUserActive(userID string, isActive bool) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		UPDATE users SET is_active = $1 
		WHERE user_id = $2 
		RETURNING user_id, username, team_name, is_active
	`, isActive, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUser(userID string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2
	`, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) CreatePR(pr *models.PullRequest, reviewers []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingPR string
	err = tx.QueryRow("SELECT pull_request_id FROM pull_requests WHERE pull_request_id = $1", pr.PullRequestID).Scan(&existingPR)
	if err == nil {
		return ErrPRExists
	} else if err != sql.ErrNoRows {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, pr.CreatedAt)
	if err != nil {
		return err
	}

	for _, reviewer := range reviewers {
		_, err = tx.Exec(`
			INSERT INTO pr_reviewers (pr_id, reviewer_id) 
			VALUES ($1, $2)
		`, pr.PullRequestID, reviewer)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPR(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var mergedAt sql.NullTime

	err := r.db.QueryRow(`
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests 
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &pr.CreatedAt, &mergedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := r.db.Query(`
		SELECT reviewer_id 
		FROM pr_reviewers 
		WHERE pr_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewer string
		err := rows.Scan(&reviewer)
		if err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewer)
	}

	return &pr, nil
}

func (r *Repository) MergePR(prID string) (*models.PullRequest, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var pr models.PullRequest
	var mergedAt sql.NullTime

	err = tx.QueryRow(`
		SELECT status, merged_at 
		FROM pull_requests 
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.Status, &mergedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if pr.Status == "MERGED" {
		return r.GetPR(prID)
	}

	now := time.Now()
	_, err = tx.Exec(`
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = $1 
		WHERE pull_request_id = $2
	`, now, prID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetPR(prID)
}

func (r *Repository) ReassignReviewer(prID, oldUserID, newUserID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRow("SELECT status FROM pull_requests WHERE pull_request_id = $1", prID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if status == "MERGED" {
		return ErrPRMerged
	}

	var count int
	err = tx.QueryRow(`
		SELECT COUNT(*) 
		FROM pr_reviewers 
		WHERE pr_id = $1 AND reviewer_id = $2
	`, prID, oldUserID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotAssigned
	}

	_, err = tx.Exec(`
		UPDATE pr_reviewers 
		SET reviewer_id = $1 
		WHERE pr_id = $2 AND reviewer_id = $3
	`, newUserID, prID, oldUserID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetUserPRs(userID string) ([]models.PullRequestShort, error) {
	rows, err := r.db.Query(`
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_requests p
		JOIN pr_reviewers pr ON p.pull_request_id = pr.pr_id
		WHERE pr.reviewer_id = $1
		ORDER BY p.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []models.PullRequestShort
	for rows.Next() {
		var pr models.PullRequestShort
		err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status)
		if err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func (r *Repository) GetRandomActiveTeamMember(teamName string, excludeUserIDs []string) (*models.User, error) {
	if len(excludeUserIDs) == 0 {
		excludeUserIDs = []string{""}
	}

	placeholders := make([]string, len(excludeUserIDs))
	args := make([]interface{}, len(excludeUserIDs)+1)
	args[0] = teamName
	
	for i, id := range excludeUserIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT user_id, username, team_name, is_active 
		FROM users 
		WHERE team_name = $1 AND is_active = true 
		AND user_id NOT IN (%s)
		ORDER BY RANDOM() 
		LIMIT 1
	`, strings.Join(placeholders, ","))

	var user models.User
	err := r.db.QueryRow(query, args...).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoCandidate
		}
		return nil, err
	}
	return &user, nil
}
