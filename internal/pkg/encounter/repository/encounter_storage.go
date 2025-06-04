package repository

import (
	"context"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
)

type encounterStorage struct {
	pool    serverrepo.PostgresPool
	metrics *mymetrics.DBMetrics
}

func NewEncounterStorage(pool serverrepo.PostgresPool, metrics *mymetrics.DBMetrics) encounterinterfaces.EncounterRepository {
	return &encounterStorage{
		pool:    pool,
		metrics: metrics,
	}
}

func (s *encounterStorage) CheckPermission(ctx context.Context, id string, userID int) bool {
	var hasPermission bool

	line := s.pool.QueryRow(ctx, CheckPermissionQuery, id, userID)
	if err := line.Scan(&hasPermission); err != nil {
		return false
	}

	return hasPermission
}
