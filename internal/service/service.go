package service

import (
	"math/rand"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
	"time"
)

type Service struct {
	repo *repository.Repository
	rand *rand.Rand
}

func NewService(repo *repository.Repository) *Service {
	src := rand.NewSource(time.Now().UnixNano())
	return &Service{
		repo: repo,
		rand: rand.New(src),
	}
}

func (s *Service) CreateTeam(team *models.Team) error {
	return s.repo.CreateTeam(team)
}

func (s *Service) GetTeam(teamName string) (*models.Team, error) {
	return s.repo.GetTeam(teamName)
}

func (s *Service) SetUserActive(userID string, isActive bool) (*models.User, error) {
	return s.repo.SetUserActive(userID, isActive)
}

func (s *Service) CreatePR(req *models.CreatePRRequest) (*models.PullRequest, error) {
	author, err := s.repo.GetUser(req.AuthorID)
	if err != nil {
		return nil, err
	}

	teamMembers, err := s.repo.GetActiveTeamMembers(author.TeamName, req.AuthorID)
	if err != nil {
		return nil, err
	}

	var reviewers []string
	availableMembers := make([]models.User, len(teamMembers))
	copy(availableMembers, teamMembers)

	for i := 0; i < 2 && len(availableMembers) > 0; i++ {
		idx := s.rand.Intn(len(availableMembers))
		reviewers = append(reviewers, availableMembers[idx].UserID)
		availableMembers = append(availableMembers[:idx], availableMembers[idx+1:]...)
	}

	pr := &models.PullRequest{
		PullRequestID:    req.PullRequestID,
		PullRequestName:  req.PullRequestName,
		AuthorID:         req.AuthorID,
		Status:           "OPEN",
		AssignedReviewers: reviewers,
		CreatedAt:        time.Now(),
	}

	err = s.repo.CreatePR(pr, reviewers)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Service) MergePR(prID string) (*models.PullRequest, error) {
	return s.repo.MergePR(prID)
}

func (s *Service) ReassignReviewer(prID, oldUserID string) (*models.PullRequest, string, error) {
	pr, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == "MERGED" {
		return nil, "", repository.ErrPRMerged
	}

	found := false
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return nil, "", repository.ErrNotAssigned
	}

	oldUser, err := s.repo.GetUser(oldUserID)
	if err != nil {
		return nil, "", err
	}

	exclude := append(pr.AssignedReviewers, oldUserID)
	newUser, err := s.repo.GetRandomActiveTeamMember(oldUser.TeamName, exclude)
	if err != nil {
		return nil, "", err
	}

	err = s.repo.ReassignReviewer(prID, oldUserID, newUser.UserID)
	if err != nil {
		return nil, "", err
	}

	updatedPR, err := s.repo.GetPR(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newUser.UserID, nil
}

func (s *Service) GetUserPRs(userID string) (*models.UserPRsResponse, error) {
	_, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, err
	}

	prs, err := s.repo.GetUserPRs(userID)
	if err != nil {
		return nil, err
	}

	return &models.UserPRsResponse{
		UserID:       userID,
		PullRequests: prs,
	}, nil
}
