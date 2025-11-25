package repository

import (
	"context"
	"time"

	"reviewer/internal/domain"
)

type Repository interface {
	CreateTeam(ctx context.Context, name string) (domain.Team, error)
	GetTeamByName(ctx context.Context, name string) (domain.Team, error)
	ListTeams(ctx context.Context) ([]domain.Team, error)

	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUser(ctx context.Context, id string) (domain.User, error)
	GetActiveTeamMembers(ctx context.Context, teamName string) ([]domain.User, error)
	GetUsersByTeam(ctx context.Context, teamName string) ([]domain.User, error)
	UpdateUser(ctx context.Context, id string, isActive *bool) (domain.User, error)

	CreatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error)
	GetPR(ctx context.Context, id string) (domain.PullRequest, error)
	GetPRForUpdate(ctx context.Context, id string) (domain.PullRequest, error)
	UpdatePRStatus(ctx context.Context, id string, status domain.PRStatus) (time.Time, error)

	AddReviewers(ctx context.Context, prID string, reviewerIDs []string) error
	RemoveReviewer(ctx context.Context, prID, userID string) error
	ListPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error)

	GetReviewerStats(ctx context.Context) ([]domain.UserAssignmentStats, error)
}

type Transactor interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
