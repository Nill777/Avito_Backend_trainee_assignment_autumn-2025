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

func TestService_Team(t *testing.T) {
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

	t.Run("CreateTeam_Success", func(t *testing.T) {
		team, err := svc.CreateTeam(ctx, "svc-team-1")

		require.NoError(t, err)
		assert.Equal(t, "svc-team-1", team.Name)
	})

	t.Run("GetTeamByName_WithMembers", func(t *testing.T) {
		tName := "get-svc-team"
		_, err = svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "gu1", "U1", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "gu2", "U2", tName, false)
		require.NoError(t, err)

		team, err := svc.GetTeamByName(ctx, tName)

		require.NoError(t, err)
		assert.Equal(t, tName, team.Name)
		assert.Len(t, team.Members, 2)
	})

	t.Run("DeactivateTeamAndRemoveReviews_Transaction", func(t *testing.T) {
		tName := "deact-team"
		_, err = svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "d1", "D1", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "d2", "D2", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "d3", "D3", tName, true)
		require.NoError(t, err)

		_, err := svc.CreatePR(ctx, "pr-deact", "Title", "d1")
		require.NoError(t, err)

		res, err := svc.DeactivateTeamAndRemoveReviews(ctx, tName)

		require.NoError(t, err)
		assert.Len(t, res.DeactivatedUsers, 3)
		assert.Len(t, res.AffectedPRs, 1)
		assert.Equal(t, "pr-deact", res.AffectedPRs[0].ID)
		u2, _ := repo.GetUser(ctx, "d2")
		assert.False(t, u2.IsActive)
		updatedPR, _ := repo.GetPR(ctx, "pr-deact")
		assert.NotContains(t, updatedPR.Reviewers, "d2")
	})

	t.Run("DeactivateTeam_NotFound", func(t *testing.T) {
		_, err := svc.DeactivateTeamAndRemoveReviews(ctx, "unknown-team")

		require.Error(t, err)
		assert.Equal(t, domain.ErrNotFound, err)
	})
}
