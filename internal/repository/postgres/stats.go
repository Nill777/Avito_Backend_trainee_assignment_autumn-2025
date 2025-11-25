package postgres

import (
	"context"

	"reviewer/internal/domain"
)

func (r *repositoryImpl) GetReviewerStats(ctx context.Context) ([]domain.UserAssignmentStats, error) {
	q := `SELECT user_id, COUNT(*) FROM pr_reviewers GROUP BY user_id ORDER BY user_id`
	rows, err := r.getQuerier(ctx).Query(ctx, q)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()

	stats := make([]domain.UserAssignmentStats, 0)
	for rows.Next() {
		var stat domain.UserAssignmentStats
		if err := rows.Scan(&stat.UserID, &stat.AssignmentCount); err != nil {
			return nil, r.handleError(err)
		}
		stats = append(stats, stat)
	}
	return stats, nil
}
