package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mapsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maps"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"github.com/jackc/pgx/v5"
)

type mapsStorage struct {
	pool    serverrepo.PostgresPool
	metrics mymetrics.DBMetrics
}

func NewMapsStorage(pool serverrepo.PostgresPool, metrics mymetrics.DBMetrics) mapsinterfaces.MapsRepository {
	return &mapsStorage{
		pool:    pool,
		metrics: metrics,
	}
}

func (s *mapsStorage) CheckPermission(ctx context.Context, id string, userID int) bool {
	fnName := utils.GetFunctionName()

	var exists bool

	_, _ = dbcall.DBCall[bool](fnName, s.metrics, func() (bool, error) {
		line := s.pool.QueryRow(ctx, CheckMapPermissionQuery, id, userID)
		if err := line.Scan(&exists); err != nil {
			return false, err
		}

		return exists, nil
	})

	return exists
}

func (s *mapsStorage) CreateMap(ctx context.Context, userID int, name string, data []byte) (*models.MapFull, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var mapFull models.MapFull
	var dataJSON []byte

	_, err := dbcall.DBCall[*models.MapFull](fnName, s.metrics, func() (*models.MapFull, error) {
		line := s.pool.QueryRow(ctx, CreateMapQuery, userID, name, data)
		if err := line.Scan(&mapFull.ID, &mapFull.UserID, &mapFull.Name, &dataJSON,
			&mapFull.CreatedAt, &mapFull.UpdatedAt); err != nil {
			return nil, err
		}

		return &mapFull, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"userID": userID, "name": name})
		return nil, apperrors.TxError
	}

	if err := json.Unmarshal(dataJSON, &mapFull.Data); err != nil {
		l.RepoError(err, map[string]any{"userID": userID, "name": name})
		return nil, apperrors.ScanError
	}

	return &mapFull, nil
}

func (s *mapsStorage) GetMapByID(ctx context.Context, userID int, id string) (*models.MapFull, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var mapFull models.MapFull
	var dataJSON []byte

	_, err := dbcall.DBCall[*models.MapFull](fnName, s.metrics, func() (*models.MapFull, error) {
		line := s.pool.QueryRow(ctx, GetMapByIDQuery, id, userID)
		if err := line.Scan(&mapFull.ID, &mapFull.UserID, &mapFull.Name, &dataJSON,
			&mapFull.CreatedAt, &mapFull.UpdatedAt); err != nil {
			return nil, err
		}

		return &mapFull, nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.MapNotFoundError
		}
		l.RepoError(err, map[string]any{"id": id, "userID": userID})
		return nil, apperrors.ScanError
	}

	if err := json.Unmarshal(dataJSON, &mapFull.Data); err != nil {
		l.RepoError(err, map[string]any{"id": id, "userID": userID})
		return nil, apperrors.ScanError
	}

	return &mapFull, nil
}

func (s *mapsStorage) UpdateMap(ctx context.Context, userID int, id string, name *string, data []byte) (*models.MapFull, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var mapFull models.MapFull
	var dataJSON []byte

	_, err := dbcall.DBCall[*models.MapFull](fnName, s.metrics, func() (*models.MapFull, error) {
		line := s.pool.QueryRow(ctx, UpdateMapQuery, id, userID, name, data)
		if err := line.Scan(&mapFull.ID, &mapFull.UserID, &mapFull.Name, &dataJSON,
			&mapFull.CreatedAt, &mapFull.UpdatedAt); err != nil {
			return nil, err
		}

		return &mapFull, nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.MapNotFoundError
		}
		l.RepoError(err, map[string]any{"id": id, "userID": userID, "name": name})
		return nil, apperrors.TxError
	}

	if err := json.Unmarshal(dataJSON, &mapFull.Data); err != nil {
		l.RepoError(err, map[string]any{"id": id, "userID": userID, "name": name})
		return nil, apperrors.ScanError
	}

	return &mapFull, nil
}

func (s *mapsStorage) DeleteMap(ctx context.Context, userID int, id string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		result, err := s.pool.Exec(ctx, DeleteMapQuery, id, userID)
		if err != nil {
			return err
		}

		if result.RowsAffected() == 0 {
			return apperrors.MapNotFoundError
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.MapNotFoundError) {
			return apperrors.MapNotFoundError
		}
		l.RepoError(err, map[string]any{"id": id, "userID": userID})
		return apperrors.TxError
	}

	return nil
}

func (s *mapsStorage) ListMaps(ctx context.Context, userID int, start, size int) ([]models.MapMetadata, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	rows, err := dbcall.DBCall[pgx.Rows](fnName, s.metrics, func() (pgx.Rows, error) {
		return s.pool.Query(ctx, ListMapsQuery, userID, size, start)
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		l.RepoError(err, map[string]any{"userID": userID, "start": start, "size": size})
		return nil, apperrors.QueryError
	}
	if rows != nil {
		defer rows.Close()
	}

	list := make([]models.MapMetadata, 0)

	if rows == nil {
		return list, nil
	}

	for rows.Next() {
		var mapMeta models.MapMetadata

		if err := rows.Scan(&mapMeta.ID, &mapMeta.UserID, &mapMeta.Name,
			&mapMeta.CreatedAt, &mapMeta.UpdatedAt); err != nil {
			l.RepoError(err, map[string]any{"userID": userID, "start": start, "size": size})
			return nil, apperrors.ScanError
		}

		list = append(list, mapMeta)
	}

	return list, nil
}
