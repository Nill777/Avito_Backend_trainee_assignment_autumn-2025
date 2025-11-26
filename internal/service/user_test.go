package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	"reviewer/internal/repository/postgres"
	testpg "reviewer/internal/tests/postgres"
)

func TestService_User(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ctx := context.Background()
	connStr, teardown, err := testpg.Setup(ctx)
	require.NoError(t, err)
	defer teardown()

	repo, err := postgres.New(ctx, connStr)
	require.NoError(t, err)
	defer repo.(interface{ Close() }).Close()
	svc := New(repo)

	tName := "user-svc-team"
	_, err = svc.CreateTeam(ctx, tName)
	require.NoError(t, err)

	t.Run("CreateUser_Success", func(t *testing.T) {
		u, err := svc.CreateUser(ctx, "u1", "User1", tName, true)

		require.NoError(t, err)
		assert.Equal(t, "u1", u.ID)
		assert.Equal(t, tName, u.TeamName)
	})

	t.Run("UpdateUser_Success", func(t *testing.T) {
		_, err := svc.CreateUser(ctx, "u2", "User2", tName, true)
		require.NoError(t, err)
		newActive := false

		updated, err := svc.UpdateUser(ctx, "u2", &newActive)

		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("UpdateUser_NotFound", func(t *testing.T) {
		isActive := true
		_, err := svc.UpdateUser(ctx, "unknown", &isActive)

		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("ListPRsByReviewer", func(t *testing.T) {
		tName := "list-pr-svc"
		_, err := repo.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "r_list", Username: "R", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "auth_list", Username: "A", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		pr := &domain.PullRequest{ID: "pr-list", Title: "T", AuthorID: "auth_list", TeamName: tName, Status: domain.PRStatusOpen}
		_, err = repo.CreatePR(ctx, pr)
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, pr.ID, []string{"r_list"})
		require.NoError(t, err)

		list, err := svc.ListPRsByReviewer(ctx, "r_list")

		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "pr-list", list[0].ID)
	})
}
