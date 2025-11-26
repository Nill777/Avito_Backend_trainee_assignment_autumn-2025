package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/repository/postgres"
	testpg "reviewer/internal/tests/postgres"
)

func TestService_Stats(t *testing.T) {
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

	t.Run("ReviewerStats_Integration", func(t *testing.T) {
		tName := "stats-svc"
		_, err = svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "u1", "U1", tName, true)
		require.NoError(t, err)

		pr1, err := svc.CreatePR(ctx, "pr1", "Title", "u1")
		require.NoError(t, err)
		pr2, err := svc.CreatePR(ctx, "pr2", "Title", "u1")
		require.NoError(t, err)

		err = repo.AddReviewers(ctx, pr1.ID, []string{"u1"})
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, pr2.ID, []string{"u1"})
		require.NoError(t, err)

		stats, err := svc.ReviewerStats(ctx)

		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, "u1", stats[0].UserID)
		assert.Equal(t, int64(2), stats[0].AssignmentCount)
	})
}
