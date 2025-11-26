package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	testpg "reviewer/internal/tests/postgres"
)

func TestRepository_PR(t *testing.T) {
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

	tName := "pr-repo"
	_, err = repo.CreateTeam(ctx, tName)
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, domain.User{ID: "auth", Username: "A", TeamName: tName, IsActive: true})
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, domain.User{ID: "rev1", Username: "R1", TeamName: tName, IsActive: true})
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, domain.User{ID: "rev2", Username: "R2", TeamName: tName, IsActive: true})
	require.NoError(t, err)

	t.Run("CreatePR_Success", func(t *testing.T) {
		in := &domain.PullRequest{
			ID: "pr-1", Title: "T", AuthorID: "auth", TeamName: tName, Status: domain.PRStatusOpen,
		}

		out, err := repo.CreatePR(ctx, in)

		require.NoError(t, err)
		assert.Equal(t, "pr-1", out.ID)
		assert.Empty(t, out.Reviewers)
	})

	t.Run("Reviewers_AddAndRemove", func(t *testing.T) {
		prID := "pr-revs"
		_, err := repo.CreatePR(ctx, &domain.PullRequest{ID: prID, Title: "T", AuthorID: "auth", TeamName: tName, Status: domain.PRStatusOpen})
		require.NoError(t, err)

		err = repo.AddReviewers(ctx, prID, []string{"rev1", "rev2"})
		require.NoError(t, err)

		pr, _ := repo.GetPR(ctx, prID)
		assert.Len(t, pr.Reviewers, 2)

		err = repo.RemoveReviewer(ctx, prID, "rev1")
		require.NoError(t, err)

		pr, _ = repo.GetPR(ctx, prID)
		assert.Len(t, pr.Reviewers, 1)
		assert.Equal(t, "rev2", pr.Reviewers[0])
	})

	t.Run("UpdatePRStatus_ReturnsTime", func(t *testing.T) {
		prID := "pr-status"
		_, err := repo.CreatePR(ctx, &domain.PullRequest{ID: prID, Title: "T", AuthorID: "auth", TeamName: tName, Status: domain.PRStatusOpen})
		require.NoError(t, err)

		mergedAt, err := repo.UpdatePRStatus(ctx, prID, domain.PRStatusMerged)

		require.NoError(t, err)
		assert.False(t, mergedAt.IsZero())

		pr, _ := repo.GetPR(ctx, prID)
		assert.Equal(t, domain.PRStatusMerged, pr.Status)
	})

	t.Run("ListPRsByReviewer", func(t *testing.T) {
		isolatedTeam := "list-team-iso"
		_, err := repo.CreateTeam(ctx, isolatedTeam)
		require.NoError(t, err)

		listPrID := "pr-list-1"
		listRevID := "rev-list-1"

		_, err = repo.CreateUser(ctx, domain.User{ID: listRevID, Username: "ListRev", TeamName: isolatedTeam, IsActive: true})
		require.NoError(t, err)

		authID := "auth-list-1"
		_, err = repo.CreateUser(ctx, domain.User{ID: authID, Username: "A", TeamName: isolatedTeam, IsActive: true})
		require.NoError(t, err)

		_, err = repo.CreatePR(ctx, &domain.PullRequest{ID: listPrID, Title: "T", AuthorID: authID, TeamName: isolatedTeam, Status: domain.PRStatusOpen})
		require.NoError(t, err)

		err = repo.AddReviewers(ctx, listPrID, []string{listRevID})
		require.NoError(t, err)

		list, err := repo.ListPRsByReviewer(ctx, listRevID)

		require.NoError(t, err)
		require.Len(t, list, 1)
		assert.Equal(t, listPrID, list[0].ID)
	})

	t.Run("RemoveReviewersFromOpenPRs", func(t *testing.T) {
		_, err := repo.CreatePR(ctx, &domain.PullRequest{ID: "pr-open", Title: "O", AuthorID: "auth", TeamName: tName, Status: domain.PRStatusOpen})
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, "pr-open", []string{"rev1"})
		require.NoError(t, err)

		_, err = repo.CreatePR(ctx, &domain.PullRequest{ID: "pr-closed", Title: "C", AuthorID: "auth", TeamName: tName, Status: domain.PRStatusMerged})
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, "pr-closed", []string{"rev1"})
		require.NoError(t, err)

		affected, err := repo.RemoveReviewersFromOpenPRs(ctx, []string{"rev1"})

		require.NoError(t, err)
		assert.Len(t, affected, 1)
		assert.Equal(t, "pr-open", affected[0].ID)

		prClosed, _ := repo.GetPR(ctx, "pr-closed")
		assert.NotEmpty(t, prClosed.Reviewers)
	})
}
