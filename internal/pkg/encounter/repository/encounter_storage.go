package repository

import (
	"context"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository"
)

type encounterStorage struct {
	pool serverrepo.PostgresPool
}

func NewEncounterStorage(pool serverrepo.PostgresPool) encounterinterfaces.EncounterRepository {
	return &encounterStorage{
		pool: pool,
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
