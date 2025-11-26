package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	testpg "reviewer/internal/tests/postgres"
)

func TestRepository_User(t *testing.T) {
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

	teamName := "user-repo"
	_, err = repo.CreateTeam(ctx, teamName)
	require.NoError(t, err)

	t.Run("CreateAndGetUser", func(t *testing.T) {
		in := domain.User{ID: "u1", Username: "Alice", TeamName: teamName, IsActive: true}

		created, err := repo.CreateUser(ctx, in)
		require.NoError(t, err)

		got, errGet := repo.GetUser(ctx, "u1")

		require.NoError(t, errGet)
		assert.Equal(t, in.ID, created.ID)
		assert.Equal(t, in.Username, got.Username)
	})

	t.Run("GetUsersByTeam", func(t *testing.T) {
		tName := "users-list-team"
		_, err := repo.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "ul1", Username: "U1", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "ul2", Username: "U2", TeamName: tName, IsActive: false})
		require.NoError(t, err)

		users, err := repo.GetUsersByTeam(ctx, tName)

		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("UpdateUser_Activity", func(t *testing.T) {
		_, err := repo.CreateUser(ctx, domain.User{ID: "u2", Username: "Bob", TeamName: teamName, IsActive: true})
		require.NoError(t, err)
		newActive := false

		updated, err := repo.UpdateUser(ctx, "u2", &newActive)

		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("GetActiveTeamMembers", func(t *testing.T) {
		tName := "active-check"
		_, err := repo.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "a1", Username: "A", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "a2", Username: "B", TeamName: tName, IsActive: false})
		require.NoError(t, err)

		users, err := repo.GetActiveTeamMembers(ctx, tName)

		require.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, "a1", users[0].ID)
	})
}
