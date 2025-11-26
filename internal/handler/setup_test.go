package handler

import (
	"context"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"reviewer/internal/logger"
	"reviewer/internal/repository"
	"reviewer/internal/repository/postgres"
	"reviewer/internal/service"
	testpg "reviewer/internal/tests/postgres"
)

func setupIntegration(t *testing.T) (*chi.Mux, repository.Repository, func()) {
	if testing.Short() {
		t.Skip("skipping E2E test")
	}

	ctx := context.Background()

	connStr, teardown, err := testpg.Setup(ctx)
	require.NoError(t, err, "failed to setup postgres container")

	repo, err := postgres.New(ctx, connStr)
	require.NoError(t, err, "failed to connect to db")

	svc := service.New(repo)
	h := New(svc, logger.New())

	r := chi.NewRouter()
	h.RegisterRoutes(r)

	return r, repo, func() {
		repo.(interface{ Close() }).Close()
		teardown()
	}
}
