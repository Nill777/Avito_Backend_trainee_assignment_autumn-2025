package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"reviewer/internal/domain"
)

func (r *repositoryImpl) CreatePR(ctx context.Context, pr *domain.PullRequest) (*domain.PullRequest, error) {
	q := `INSERT INTO pull_requests (id, title, author_id, team_name, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	err := r.getQuerier(ctx).QueryRow(ctx, q, pr.ID, pr.Title, pr.AuthorID, pr.TeamName, pr.Status).
		Scan(&pr.ID, &pr.CreatedAt)
	if err != nil {
		return nil, r.handleError(err)
	}
	pr.Reviewers = []string{}
	return pr, nil
}

func (r *repositoryImpl) GetPR(ctx context.Context, id string) (domain.PullRequest, error) {
	return r.getPRInternal(ctx, id, false)
}

func (r *repositoryImpl) GetPRForUpdate(ctx context.Context, id string) (domain.PullRequest, error) {
	return r.getPRInternal(ctx, id, true)
}

func (r *repositoryImpl) getPRInternal(ctx context.Context, id string, forUpdate bool) (domain.PullRequest, error) {
	q := `SELECT id, title, author_id, team_name, status, created_at, merged_at FROM pull_requests WHERE id = $1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	var pr domain.PullRequest
	err := r.getQuerier(ctx).QueryRow(ctx, q, id).
		Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.TeamName, &pr.Status, &pr.CreatedAt, &pr.MergedAt)
	if err != nil {
		return domain.PullRequest{}, r.handleError(err)
	}

	reviewers, err := r.GetReviewerIDs(ctx, id)
	if err != nil {
		return domain.PullRequest{}, err
	}
	pr.Reviewers = reviewers
	return pr, nil
}

func (r *repositoryImpl) UpdatePRStatus(ctx context.Context, id string, status domain.PRStatus) (time.Time, error) {
	q := `UPDATE pull_requests 
	      SET status = $1, merged_at = CASE WHEN $1 = 'MERGED' THEN NOW() ELSE NULL END 
	      WHERE id = $2 
	      RETURNING COALESCE(merged_at, NOW())`

	var t time.Time
	err := r.getQuerier(ctx).QueryRow(ctx, q, status, id).Scan(&t)
	if err != nil {
		return time.Time{}, r.handleError(err)
	}
	return t, nil
}

func (r *repositoryImpl) AddReviewers(ctx context.Context, prID string, reviewerIDs []string) error {
	if len(reviewerIDs) == 0 {
		return nil
	}
	b := &pgx.Batch{}
	for _, userID := range reviewerIDs {
		b.Queue("INSERT INTO pr_reviewers (pr_id, user_id) VALUES ($1, $2)", prID, userID)
	}
	br := r.getQuerier(ctx).SendBatch(ctx, b)
	defer br.Close()
	for i := 0; i < len(reviewerIDs); i++ {
		_, err := br.Exec()
		if err != nil {
			return r.handleError(err)
		}
	}
	return nil
}

func (r *repositoryImpl) RemoveReviewer(ctx context.Context, prID, userID string) error {
	q := `DELETE FROM pr_reviewers WHERE pr_id = $1 AND user_id = $2`
	cmdTag, err := r.getQuerier(ctx).Exec(ctx, q, prID, userID)
	if err != nil {
		return r.handleError(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return domain.ErrNotAssigned
	}
	return nil
}

func (r *repositoryImpl) GetReviewerIDs(ctx context.Context, prID string) ([]string, error) {
	q := `SELECT user_id FROM pr_reviewers WHERE pr_id = $1 ORDER BY user_id`
	rows, err := r.getQuerier(ctx).Query(ctx, q, prID)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, r.handleError(err)
		}
		ids = append(ids, id)
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}

func (r *repositoryImpl) ListPRsByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	q := `
		SELECT pr.id, pr.title, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r ON pr.id = r.pr_id
		WHERE r.user_id = $1
		ORDER BY pr.created_at DESC
	`
	rows, err := r.getQuerier(ctx).Query(ctx, q, reviewerID)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()

	var prs []domain.PullRequestShort
	for rows.Next() {
		var pr domain.PullRequestShort
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status); err != nil {
			return nil, r.handleError(err)
		}
		prs = append(prs, pr)
	}
	if prs == nil {
		prs = []domain.PullRequestShort{}
	}
	return prs, nil
}
