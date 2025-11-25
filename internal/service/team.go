package service

import (
	"context"
	"fmt"
	"reviewer/internal/domain"
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
