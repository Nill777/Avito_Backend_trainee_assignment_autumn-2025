package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"reviewer/internal/domain"
	"reviewer/internal/repository"
)

type txKey struct{}

type repositoryImpl struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connString string) (repository.Repository, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &repositoryImpl{pool: pool}, nil
}

func (r *repositoryImpl) Close() { r.pool.Close() }

func (r *repositoryImpl) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

func (r *repositoryImpl) getQuerier(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return r.pool
}

func (r *repositoryImpl) CreateTeam(ctx context.Context, name string) (domain.Team, error) {
	q := `INSERT INTO teams (name) VALUES ($1) RETURNING name`
	var t domain.Team
	err := r.getQuerier(ctx).QueryRow(ctx, q, name).Scan(&t.Name)
	return t, r.handleError(err)
}

func (r *repositoryImpl) GetTeamByName(ctx context.Context, name string) (domain.Team, error) {
	q := `SELECT name FROM teams WHERE name = $1`
	var t domain.Team
	err := r.getQuerier(ctx).QueryRow(ctx, q, name).Scan(&t.Name)
	return t, r.handleError(err)
}

func (r *repositoryImpl) ListTeams(ctx context.Context) ([]domain.Team, error) {
	q := `SELECT name FROM teams ORDER BY name`
	rows, err := r.getQuerier(ctx).Query(ctx, q)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()
	var teams []domain.Team
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.Name); err != nil {
			return nil, r.handleError(err)
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (r *repositoryImpl) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	q := `INSERT INTO users (id, username, team_name, is_active) VALUES ($1, $2, $3, $4) RETURNING id, username, team_name, is_active`
	var u domain.User
	err := r.getQuerier(ctx).QueryRow(ctx, q, user.ID, user.Username, user.TeamName, user.IsActive).
		Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	return u, r.handleError(err)
}

func (r *repositoryImpl) GetUser(ctx context.Context, id string) (domain.User, error) {
	q := `SELECT id, username, team_name, is_active FROM users WHERE id = $1`
	var u domain.User
	err := r.getQuerier(ctx).QueryRow(ctx, q, id).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	return u, r.handleError(err)
}

func (r *repositoryImpl) GetActiveTeamMembers(ctx context.Context, teamName string) ([]domain.User, error) {
	q := `SELECT id, username, team_name, is_active FROM users WHERE team_name = $1 AND is_active = true`
	rows, err := r.getQuerier(ctx).Query(ctx, q, teamName)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, r.handleError(err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repositoryImpl) GetUsersByTeam(ctx context.Context, teamName string) ([]domain.User, error) {
	q := `SELECT id, username, team_name, is_active FROM users WHERE team_name = $1 ORDER BY id`
	rows, err := r.getQuerier(ctx).Query(ctx, q, teamName)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()
	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, r.handleError(err)
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *repositoryImpl) UpdateUser(ctx context.Context, id string, isActive *bool) (domain.User, error) {
	if isActive == nil {
		return r.GetUser(ctx, id)
	}
	q := `UPDATE users SET is_active = $1 WHERE id = $2 RETURNING id, username, team_name, is_active`
	var u domain.User
	err := r.getQuerier(ctx).QueryRow(ctx, q, *isActive, id).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	return u, r.handleError(err)
}

func (r *repositoryImpl) CreatePR(ctx context.Context, pr domain.PullRequest) (domain.PullRequest, error) {
	q := `INSERT INTO pull_requests (id, title, author_id, team_name, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	err := r.getQuerier(ctx).QueryRow(ctx, q, pr.ID, pr.Title, pr.AuthorID, pr.TeamName, pr.Status).
		Scan(&pr.ID, &pr.CreatedAt)
	if err != nil {
		return domain.PullRequest{}, r.handleError(err)
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

func (r *repositoryImpl) GetReviewerStats(ctx context.Context) (map[string]int64, error) {
	q := `SELECT user_id, COUNT(*) FROM pr_reviewers GROUP BY user_id`
	rows, err := r.getQuerier(ctx).Query(ctx, q)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()
	stats := make(map[string]int64)
	for rows.Next() {
		var userID string
		var count int64
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, r.handleError(err)
		}
		stats[userID] = count
	}
	return stats, nil
}

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
}

func (r *repositoryImpl) handleError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			return domain.ErrConflict
		}
		if pgErr.Code == pgerrcode.ForeignKeyViolation {
			return domain.ErrNotFound
		}
	}
	return err
}
