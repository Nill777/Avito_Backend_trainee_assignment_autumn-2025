package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	testpg "reviewer/internal/tests/postgres"
)

func TestRepository_Stats(t *testing.T) {
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

	tName := "stats-repo"
	_, err = repo.CreateTeam(ctx, tName)
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, domain.User{ID: "s1", Username: "S1", TeamName: tName, IsActive: true})
	require.NoError(t, err)

	_, err = repo.CreatePR(ctx, &domain.PullRequest{ID: "p1", Title: "T", AuthorID: "s1", TeamName: tName, Status: "OPEN"})
	require.NoError(t, err)
	_, err = repo.CreatePR(ctx, &domain.PullRequest{ID: "p2", Title: "T", AuthorID: "s1", TeamName: tName, Status: "OPEN"})
	require.NoError(t, err)

	err = repo.AddReviewers(ctx, "p1", []string{"s1"})
	require.NoError(t, err)
	err = repo.AddReviewers(ctx, "p2", []string{"s1"})
	require.NoError(t, err)

	t.Run("GetReviewerStats", func(t *testing.T) {
		stats, err := repo.GetReviewerStats(ctx)

		require.NoError(t, err)
		assert.Len(t, stats, 1)
		assert.Equal(t, "s1", stats[0].UserID)
		assert.Equal(t, int64(2), stats[0].AssignmentCount)
	})
}
