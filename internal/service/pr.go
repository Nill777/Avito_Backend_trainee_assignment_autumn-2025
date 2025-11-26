package service

import (
	"context"
	"errors"
	"fmt"

	"reviewer/internal/domain"
	"reviewer/internal/repository"
)

func (s *Service) CreatePR(ctx context.Context, prID, title, authorID string) (*domain.PullRequest, error) {
	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return nil, fmt.Errorf("getting author: %w", err)
	}

	candidates, err := s.repo.GetActiveTeamMembers(ctx, author.TeamName)
	if err != nil {
		return nil, fmt.Errorf("getting candidates: %w", err)
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

	var createdPR *domain.PullRequest
	txRepo, ok := s.repo.(repository.Transactor)
	if !ok {
		return nil, errors.New("repository does not support transactions")
	}

	err = txRepo.RunInTx(ctx, func(ctxTx context.Context) error {
		prModel := &domain.PullRequest{
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

		tempPR, err := s.repo.GetPR(ctxTx, pr.ID)
		if err != nil {
			return err
		}
		createdPR = &tempPR
		return nil
	})

	if err != nil {
		return nil, err
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
