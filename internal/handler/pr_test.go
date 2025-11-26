package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
)

func TestHandler_PR(t *testing.T) {
	r, repo, teardown := setupIntegration(t)
	defer teardown()

	ctx := context.Background()

	createTeam := `{"team_name": "pr-api", "members": [
		{"user_id": "auth", "username": "A", "is_active": true},
		{"user_id": "r1", "username": "R1", "is_active": true},
		{"user_id": "r2", "username": "R2", "is_active": true}
	]}`
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewBufferString(createTeam)))

	t.Run("CreatePR_Success", func(t *testing.T) {
		body := `{"pull_request_id": "pr-1", "pull_request_name": "Test", "author_id": "auth"}`
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		pr := resp["pr"].(map[string]any)

		assert.Equal(t, "pr-1", pr["pull_request_id"])
		assert.Equal(t, "OPEN", pr["status"])
		reviewers := pr["assigned_reviewers"].([]any)
		assert.Len(t, reviewers, 2)
	})

	t.Run("MergePR_Success", func(t *testing.T) {
		body := `{"pull_request_id": "pr-1"}`
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		pr := resp["pr"].(map[string]any)

		assert.Equal(t, "MERGED", pr["status"])
		assert.NotNil(t, pr["mergedAt"])
	})

	t.Run("ReassignReviewer_Success", func(t *testing.T) {
		tName := "reassign-api"
		_, err := repo.CreateTeam(ctx, tName)
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "re_a", Username: "A", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "re_old", Username: "O", TeamName: tName, IsActive: true})
		require.NoError(t, err)
		_, err = repo.CreateUser(ctx, domain.User{ID: "re_new", Username: "N", TeamName: tName, IsActive: true})
		require.NoError(t, err)

		pr := &domain.PullRequest{ID: "pr-re", Title: "T", AuthorID: "re_a", TeamName: tName, Status: domain.PRStatusOpen}
		_, err = repo.CreatePR(ctx, pr)
		require.NoError(t, err)
		err = repo.AddReviewers(ctx, "pr-re", []string{"re_old"})
		require.NoError(t, err)

		reqBody := `{"pull_request_id": "pr-re", "old_reviewer_id": "re_old"}`
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		assert.Equal(t, "re_new", resp["replaced_by"])

		prMap := resp["pr"].(map[string]any)
		assert.Equal(t, "pr-re", prMap["pull_request_id"])

		reviewers := prMap["assigned_reviewers"].([]any)
		assert.Contains(t, reviewers, "re_new")
		assert.NotContains(t, reviewers, "re_old")
	})

	t.Run("ReassignReviewer_Fail_Merged", func(t *testing.T) {
		body := `{"pull_request_id": "pr-1", "old_user_id": "r1"}`
		req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var resp APIErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PR_MERGED", resp.Error.Code)
	})
}
