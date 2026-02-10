package usecases

import (
	"context"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptilesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
)

type mapTilesUsecases struct {
	repo maptilesinterfaces.MapTilesRepository
}

func NewMapTilesUsecases(repo maptilesinterfaces.MapTilesRepository) maptilesinterfaces.MapTilesUsecases {
	return &mapTilesUsecases{repo: repo}
}

func (uc *mapTilesUsecases) GetCategories(ctx context.Context, userID int) ([]*models.MapTileCategory, error) {
	l := logger.FromContext(ctx)
	if userID < 0 {
		l.UsecasesWarn(apperrors.InvalidUserIDError, userID, map[string]any{"userID": userID})
		return nil, apperrors.InvalidUserIDError
	}

	list, err := uc.repo.GetCategories(ctx, userID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"userID": strconv.Itoa(userID)})
		return nil, err
	}

	return list, nil
}

func (uc *mapTilesUsecases) GetWalkabilityByTileID(ctx context.Context, tileID string) (*models.TileWalkability, error) {
	l := logger.FromContext(ctx)
	if tileID == "" {
		l.UsecasesWarn(apperrors.InvalidTileIDError, 0, map[string]any{"tileID": tileID})
		return nil, apperrors.InvalidTileIDError
	}

	walkability, err := uc.repo.GetWalkabilityByTileID(ctx, tileID)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"tileID": tileID})
		return nil, err
	}

	return walkability, nil
}

func (uc *mapTilesUsecases) GetWalkabilityBySetID(ctx context.Context, setID string) ([]*models.TileWalkability, error) {
	l := logger.FromContext(ctx)
	if setID == "" {
		l.UsecasesWarn(apperrors.InvalidSetIDError, 0, map[string]any{"setID": setID})
		return nil, apperrors.InvalidSetIDError
	}

	walkabilities, err := uc.repo.GetWalkabilityBySetID(ctx, setID)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"setID": setID})
		return nil, err
	}

	return walkabilities, nil
}

func (uc *mapTilesUsecases) UpsertWalkability(ctx context.Context, walkability *models.TileWalkability) error {
	l := logger.FromContext(ctx)
	if walkability.TileID == "" {
		l.UsecasesWarn(apperrors.InvalidTileIDError, 0, map[string]any{"tileID": walkability.TileID})
		return apperrors.InvalidTileIDError
	}

	if err := uc.repo.UpsertWalkability(ctx, walkability); err != nil {
		l.UsecasesError(err, 0, map[string]any{"tileID": walkability.TileID})
		return err
	}

	return nil
}

func (uc *mapTilesUsecases) AddTile(ctx context.Context, categoryID string, tile *models.MapTile) error {
	l := logger.FromContext(ctx)
	if categoryID == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"categoryID": categoryID})
		return apperrors.InvalidIDErr
	}
	if tile.ID == "" || tile.Name == "" {
		l.UsecasesWarn(apperrors.InvalidTileIDError, 0, map[string]any{"tile": tile})
		return apperrors.InvalidTileIDError
	}

	if err := uc.repo.AddTile(ctx, categoryID, tile); err != nil {
		l.UsecasesError(err, 0, map[string]any{"categoryID": categoryID, "tileID": tile.ID})
		return err
	}

	return nil
}

func (uc *mapTilesUsecases) UpdateTile(ctx context.Context, categoryID string, tile *models.MapTile) error {
	l := logger.FromContext(ctx)
	if categoryID == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"categoryID": categoryID})
		return apperrors.InvalidIDErr
	}
	if tile.ID == "" {
		l.UsecasesWarn(apperrors.InvalidTileIDError, 0, map[string]any{"tile": tile})
		return apperrors.InvalidTileIDError
	}

	if err := uc.repo.UpdateTile(ctx, categoryID, tile); err != nil {
		l.UsecasesError(err, 0, map[string]any{"categoryID": categoryID, "tileID": tile.ID})
		return err
	}

	return nil
}

func (uc *mapTilesUsecases) DeleteTile(ctx context.Context, categoryID string, tileID string) error {
	l := logger.FromContext(ctx)
	if categoryID == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"categoryID": categoryID})
		return apperrors.InvalidIDErr
	}
	if tileID == "" {
		l.UsecasesWarn(apperrors.InvalidTileIDError, 0, map[string]any{"tileID": tileID})
		return apperrors.InvalidTileIDError
	}

	if err := uc.repo.DeleteTile(ctx, categoryID, tileID); err != nil {
		l.UsecasesError(err, 0, map[string]any{"categoryID": categoryID, "tileID": tileID})
		return err
	}

	return nil
}
