package handler

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"reviewer/internal/domain"
	"reviewer/internal/logger"
	"reviewer/internal/repository"
	"reviewer/internal/repository/postgres"
	"reviewer/internal/service"
	testpg "reviewer/internal/tests/postgres"
)

const (
	BenchUsersCount = 100
	BenchPRsCount   = 1000
)

func setupBenchmark(b *testing.B) (*chi.Mux, repository.Repository, func()) {
	ctx := context.Background()
	connStr, teardown, err := testpg.Setup(ctx)
	require.NoError(b, err)

	repo, err := postgres.New(ctx, connStr)
	require.NoError(b, err)

	svc := service.New(repo)
	h := New(svc, logger.New())

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	return r, repo, func() {
		repo.(interface{ Close() }).Close()
		teardown()
	}
}

func prepareBenchData(ctx context.Context, repo repository.Repository, teamName string) {
	_, _ = repo.CreateTeam(ctx, teamName)

	userIDs := make([]string, BenchUsersCount)
	for i := 0; i < BenchUsersCount; i++ {
		uid := fmt.Sprintf("%s-u-%d", teamName, i)
		userIDs[i] = uid
		_, _ = repo.CreateUser(ctx, domain.User{
			ID:       uid,
			Username: fmt.Sprintf("User%d", i),
			TeamName: teamName,
			IsActive: true,
		})
	}

	for i := 0; i < BenchPRsCount; i++ {
		prID := fmt.Sprintf("%s-pr-%d", teamName, i)
		authorID := userIDs[rand.Intn(len(userIDs))]
		reviewerID := userIDs[rand.Intn(len(userIDs))]

		if authorID == reviewerID {
			reviewerID = userIDs[(rand.Intn(len(userIDs))+1)%len(userIDs)]
		}

		_, _ = repo.CreatePR(ctx, &domain.PullRequest{
			ID:       prID,
			Title:    "Bench PR",
			AuthorID: authorID,
			TeamName: teamName,
			Status:   domain.PRStatusOpen,
		})

		_ = repo.AddReviewers(ctx, prID, []string{reviewerID})
	}
}

func BenchmarkDeactivateTeam(b *testing.B) {
	r, repo, teardown := setupBenchmark(b)
	defer teardown()
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		teamName := fmt.Sprintf("bench-team-%d", i)
		prepareBenchData(ctx, repo, teamName)

		reqBody := fmt.Sprintf(`{"team_name": %q}`, teamName)
		req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewBufferString(reqBody))
		w := httptest.NewRecorder()

		b.StartTimer()
		r.ServeHTTP(w, req)
		b.StopTimer()

		if w.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	}
}
