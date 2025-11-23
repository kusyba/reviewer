package models

import "time"

type TeamMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type Team struct {
	TeamName string       `json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type PullRequest struct {
	PullRequestID    string    `json:"pull_request_id"`
	PullRequestName  string    `json:"pull_request_name"`
	AuthorID         string    `json:"author_id"`
	Status           string    `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt        time.Time `json:"createdAt,omitempty"`
	MergedAt         *time.Time `json:"mergedAt,omitempty"`
}

type PullRequestShort struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type SetActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type UserPRsResponse struct {
	UserID        string               `json:"user_id"`
	PullRequests  []PullRequestShort   `json:"pull_requests"`
}
