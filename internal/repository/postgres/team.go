package postgres

import (
	"context"

	"reviewer/internal/domain"
)

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

func (r *repositoryImpl) DeactivateTeamMembers(ctx context.Context, teamName string) ([]domain.User, error) {
	q := `UPDATE users SET is_active = false WHERE team_name = $1 AND is_active = true RETURNING id, username, team_name, is_active`
	rows, err := r.getQuerier(ctx).Query(ctx, q, teamName)
	if err != nil {
		return nil, r.handleError(err)
	}
	defer rows.Close()

	deactivatedUsers := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, r.handleError(err)
		}
		deactivatedUsers = append(deactivatedUsers, u)
	}
	return deactivatedUsers, nil
}
