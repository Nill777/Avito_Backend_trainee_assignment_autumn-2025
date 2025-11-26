package service

import (
	"context"
	"errors"
	"fmt"

	"reviewer/internal/domain"
	"reviewer/internal/repository"
)

func (s *Service) CreateTeam(ctx context.Context, name string) (domain.Team, error) {
	return s.repo.CreateTeam(ctx, name)
}

func (s *Service) GetTeamByName(ctx context.Context, name string) (domain.Team, error) {
	team, err := s.repo.GetTeamByName(ctx, name)
	if err != nil {
		return domain.Team{}, err
	}
	users, err := s.repo.GetUsersByTeam(ctx, team.Name)
	if err != nil {
		return domain.Team{}, fmt.Errorf("getting team members: %w", err)
	}
	team.Members = make([]domain.TeamMember, len(users))
	for i, u := range users {
		team.Members[i] = domain.TeamMember{
			UserID:   u.ID,
			Username: u.Username,
			IsActive: u.IsActive,
		}
	}
	return team, nil
}

func (s *Service) DeactivateTeamAndRemoveReviews(ctx context.Context, teamName string) (*domain.DeactivationResult, error) {
	if _, err := s.repo.GetTeamByName(ctx, teamName); err != nil {
		return nil, err
	}

	result := &domain.DeactivationResult{
		DeactivatedUsers: []domain.User{},
		AffectedPRs:      []domain.PullRequestShort{},
	}

	txRepo, ok := s.repo.(repository.Transactor)
	if !ok {
		return nil, errors.New("repository does not support transactions")
	}

	err := txRepo.RunInTx(ctx, func(ctxTx context.Context) error {
		deactivatedUsers, err := s.repo.DeactivateTeamMembers(ctxTx, teamName)
		if err != nil {
			return fmt.Errorf("deactivating members: %w", err)
		}
		result.DeactivatedUsers = deactivatedUsers

		if len(deactivatedUsers) == 0 {
			return nil
		}

		deactivatedUserIDs := make([]string, len(deactivatedUsers))
		for i, u := range deactivatedUsers {
			deactivatedUserIDs[i] = u.ID
		}

		affectedPRs, err := s.repo.RemoveReviewersFromOpenPRs(ctxTx, deactivatedUserIDs)
		if err != nil {
			return fmt.Errorf("removing reviewers from open PRs: %w", err)
		}
		result.AffectedPRs = affectedPRs

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
