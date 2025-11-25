package service

import (
	"context"
	"reviewer/internal/domain"
)

func (s *Service) CreateUser(ctx context.Context, id, username, teamName string, isActive bool) (domain.User, error) {
	user := domain.User{
		ID:       id,
		Username: username,
		TeamName: teamName,
		IsActive: isActive,
	}
	return s.repo.CreateUser(ctx, user)
}

func (s *Service) UpdateUser(ctx context.Context, id string, isActive *bool) (domain.User, error) {
	return s.repo.UpdateUser(ctx, id, isActive)
}

func (s *Service) ReviewerStats(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetReviewerStats(ctx)
}

func (s *Service) ListPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	return s.repo.ListPRsByReviewer(ctx, reviewerID)
}
