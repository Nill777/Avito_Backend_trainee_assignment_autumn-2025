package service

import (
	"context"

	"reviewer/internal/domain"
)

func (s *Service) ReviewerStats(ctx context.Context) ([]domain.UserAssignmentStats, error) {
	return s.repo.GetReviewerStats(ctx)
}
