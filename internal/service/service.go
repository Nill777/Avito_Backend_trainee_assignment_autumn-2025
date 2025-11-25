package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"reviewer/internal/domain"
	"reviewer/internal/repository"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

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

func (s *Service) CreatePR(ctx context.Context, prID, title, authorID string) (domain.PullRequest, error) {
	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("getting author: %w", err)
	}

	candidates, err := s.repo.GetActiveTeamMembers(ctx, author.TeamName)
	if err != nil {
		return domain.PullRequest{}, fmt.Errorf("getting candidates: %w", err)
	}

	validCandidates := make([]domain.User, 0, len(candidates))
	for _, u := range candidates {
		if u.ID != author.ID {
			validCandidates = append(validCandidates, u)
		}
	}

	selectedUsers := s.pickRandomReviewers(validCandidates, 2)
	selectedIDs := make([]string, len(selectedUsers))
	for i, u := range selectedUsers {
		selectedIDs[i] = u.ID
	}

	var createdPR domain.PullRequest
	txRepo, ok := s.repo.(repository.Transactor)
	if !ok {
		return domain.PullRequest{}, errors.New("repository does not support transactions")
	}

	err = txRepo.RunInTx(ctx, func(ctxTx context.Context) error {
		prModel := domain.PullRequest{
			ID:       prID,
			Title:    title,
			AuthorID: author.ID,
			TeamName: author.TeamName,
			Status:   domain.PRStatusOpen,
		}

		pr, err := s.repo.CreatePR(ctxTx, prModel)
		if err != nil {
			return err
		}

		if err := s.repo.AddReviewers(ctxTx, pr.ID, selectedIDs); err != nil {
			return err
		}

		createdPR, err = s.repo.GetPR(ctxTx, pr.ID)
		return err
	})

	if err != nil {
		return domain.PullRequest{}, err
	}

	return createdPR, nil
}

func (s *Service) MergePR(ctx context.Context, prID string) (domain.PullRequest, error) {
	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, err
	}

	if pr.Status == domain.PRStatusMerged {
		return pr, nil
	}

	mergedAt, err := s.repo.UpdatePRStatus(ctx, prID, domain.PRStatusMerged)
	if err != nil {
		return domain.PullRequest{}, err
	}

	pr.Status = domain.PRStatusMerged
	pr.MergedAt = &mergedAt

	return pr, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (domain.PullRequest, domain.User, error) {
	var resultPR domain.PullRequest
	var newReviewer domain.User

	txRepo, ok := s.repo.(repository.Transactor)
	if !ok {
		return domain.PullRequest{}, domain.User{}, errors.New("repository does not support transactions")
	}

	err := txRepo.RunInTx(ctx, func(ctxTx context.Context) error {
		pr, err := s.repo.GetPRForUpdate(ctxTx, prID)
		if err != nil {
			return err
		}

		if pr.Status == domain.PRStatusMerged {
			return domain.ErrPRMerged
		}

		isAssigned := false
		currentReviewerIDs := make(map[string]bool)
		for _, rID := range pr.Reviewers {
			currentReviewerIDs[rID] = true
			if rID == oldReviewerID {
				isAssigned = true
			}
		}
		if !isAssigned {
			return domain.ErrNotAssigned
		}

		candidates, err := s.repo.GetActiveTeamMembers(ctxTx, pr.TeamName)
		if err != nil {
			return err
		}

		possibleReplacements := make([]domain.User, 0)
		for _, c := range candidates {
			if c.ID == pr.AuthorID {
				continue
			}
			if c.ID == oldReviewerID {
				continue
			}
			if _, exists := currentReviewerIDs[c.ID]; exists {
				continue
			}
			possibleReplacements = append(possibleReplacements, c)
		}

		if len(possibleReplacements) == 0 {
			return domain.ErrNoCandidates
		}

		selected := s.pickRandomReviewers(possibleReplacements, 1)
		newReviewer = selected[0]

		if err := s.repo.RemoveReviewer(ctxTx, prID, oldReviewerID); err != nil {
			return err
		}
		if err := s.repo.AddReviewers(ctxTx, prID, []string{newReviewer.ID}); err != nil {
			return err
		}

		resultPR, err = s.repo.GetPR(ctxTx, prID)
		return err
	})

	return resultPR, newReviewer, err
}

func (s *Service) ListPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	return s.repo.ListPRsByReviewer(ctx, reviewerID)
}

func (s *Service) ReviewerStats(ctx context.Context) (map[string]int64, error) {
	return s.repo.GetReviewerStats(ctx)
}

func (s *Service) pickRandomReviewers(users []domain.User, n int) []domain.User {
	if len(users) == 0 || n <= 0 {
		return []domain.User{}
	}
	shuffled := make([]domain.User, len(users))
	copy(shuffled, users)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	if n > len(shuffled) {
		n = len(shuffled)
	}
	return shuffled[:n]
}
