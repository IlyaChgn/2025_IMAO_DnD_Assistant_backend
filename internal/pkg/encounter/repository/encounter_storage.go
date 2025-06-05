package repository

import (
	"context"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
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
	fnName := utils.GetFunctionName()

	var hasPermission bool

	hasPermission, _ = dbcall.DBCall[bool](fnName, s.metrics, func() (bool, error) {
		line := s.pool.QueryRow(ctx, CheckPermissionQuery, id, userID)
		if err := line.Scan(&hasPermission); err != nil {
			return false, nil
		}

		return hasPermission, nil
	})

	return hasPermission
}
