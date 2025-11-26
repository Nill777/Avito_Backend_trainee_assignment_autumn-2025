package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	testpg "reviewer/internal/tests/postgres"
)

func TestRepository_Team(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	connStr, teardown, err := testpg.Setup(ctx)
	require.NoError(t, err)
	defer teardown()

	repo, err := New(ctx, connStr)
	require.NoError(t, err)
	defer repo.(interface{ Close() }).Close()

	t.Run("CreateTeam_Success", func(t *testing.T) {
		team, err := repo.CreateTeam(ctx, "backend")

		require.NoError(t, err)
		assert.Equal(t, "backend", team.Name)
	})

	t.Run("GetTeamByName_Success", func(t *testing.T) {
		_, err := repo.CreateTeam(ctx, "frontend")
		require.NoError(t, err)

		team, err := repo.GetTeamByName(ctx, "frontend")

		require.NoError(t, err)
		assert.Equal(t, "frontend", team.Name)
	})

	t.Run("GetTeamByName_NotFound", func(t *testing.T) {
		_, err := repo.GetTeamByName(ctx, "unknown")

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("ListTeams", func(t *testing.T) {
		_, _ = repo.CreateTeam(ctx, "list-t1")
		_, _ = repo.CreateTeam(ctx, "list-t2")

		teams, err := repo.ListTeams(ctx)

		require.NoError(t, err)
		found := 0
		for _, team := range teams {
			if team.Name == "list-t1" || team.Name == "list-t2" {
				found++
			}
		}
		assert.GreaterOrEqual(t, found, 2)
	})

	t.Run("DeactivateTeamMembers_Success", func(t *testing.T) {
		tName := "deact-repo"
		_, err := repo.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "u1", Username: "1", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "u2", Username: "2", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "u3", Username: "3", TeamName: tName, IsActive: false})
		require.NoError(t, err)

		users, err := repo.DeactivateTeamMembers(ctx, tName)

		require.NoError(t, err)
		assert.Len(t, users, 2)
		for _, u := range users {
			assert.False(t, u.IsActive)
		}

		u1, _ := repo.GetUser(ctx, "u1")
		assert.False(t, u1.IsActive)
	})
}
