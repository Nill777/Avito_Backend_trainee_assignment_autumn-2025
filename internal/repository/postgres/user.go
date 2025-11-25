package postgres

import (
	"context"

	"reviewer/internal/domain"
)

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
