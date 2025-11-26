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

func TestService_PR(t *testing.T) {
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

	t.Run("CreatePR_AutoAssignment", func(t *testing.T) {
		tName := "pr-svc-team"
		_, err := svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "s_auth", "Author", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "s_r1", "Rev1", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "s_r2", "Rev2", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "s_r3", "Rev3", tName, true)
		require.NoError(t, err)

		pr, err := svc.CreatePR(ctx, "pr-1", "Test", "s_auth")

		require.NoError(t, err)
		assert.Len(t, pr.Reviewers, 2)
		assert.NotContains(t, pr.Reviewers, "s_auth")
	})

	t.Run("CreatePR_NotEnoughCandidates", func(t *testing.T) {
		tName := "small-team"
		_, err := svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "sm_a", "Author", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "sm_r", "Rev1", tName, true)
		require.NoError(t, err)

		pr, err := svc.CreatePR(ctx, "pr-small", "Test", "sm_a")

		require.NoError(t, err)
		require.Len(t, pr.Reviewers, 1)
		assert.Equal(t, "sm_r", pr.Reviewers[0])
	})

	t.Run("MergePR_Idempotency", func(t *testing.T) {
		tName := "merge-team"
		_, err := svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "m_u1", "M1", tName, true)
		require.NoError(t, err)
		pr, _ := svc.CreatePR(ctx, "pr-merge", "M", "m_u1")

		merged1, err := svc.MergePR(ctx, pr.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.PRStatusMerged, merged1.Status)
		assert.NotNil(t, merged1.MergedAt)
		firstTime := *merged1.MergedAt

		merged2, err := svc.MergePR(ctx, pr.ID)

		require.NoError(t, err)
		assert.Equal(t, domain.PRStatusMerged, merged2.Status)
		assert.Equal(t, firstTime.Unix(), merged2.MergedAt.Unix())
	})

	t.Run("ReassignReviewer_Success", func(t *testing.T) {
		tName := "reassign-team"
		_, err := svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "re_a", "Author", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "re_old", "Old", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "re_new", "New", tName, true)
		require.NoError(t, err)

		prIn := &domain.PullRequest{
			ID: "pr-re", Title: "Test", AuthorID: "re_a", TeamName: tName, Status: domain.PRStatusOpen,
		}
		_, err = repo.CreatePR(ctx, prIn)
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, "pr-re", []string{"re_old"})
		require.NoError(t, err)

		updatedPR, newReviewer, err := svc.ReassignReviewer(ctx, "pr-re", "re_old")

		require.NoError(t, err)
		assert.Equal(t, "re_new", newReviewer.ID)
		assert.Contains(t, updatedPR.Reviewers, "re_new", "New reviewer should be present")
		assert.NotContains(t, updatedPR.Reviewers, "re_old", "Old reviewer should be gone")
	})

	t.Run("ReassignReviewer_NoCandidates", func(t *testing.T) {
		tName := "no-cand-team"
		_, err := svc.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "nc_a", "Author", tName, true)
		require.NoError(t, err)
		_, err = svc.CreateUser(ctx, "nc_r", "Rev", tName, true)
		require.NoError(t, err)
		pr, err := svc.CreatePR(ctx, "pr-nc", "Test", "nc_a")
		require.NoError(t, err)

		_, _, err = svc.ReassignReviewer(ctx, pr.ID, "nc_r")

		require.ErrorIs(t, err, domain.ErrNoCandidates)
	})
}
