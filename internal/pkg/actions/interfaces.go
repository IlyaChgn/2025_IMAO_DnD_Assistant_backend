package actions

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type ActionsUsecases interface {
	ExecuteAction(ctx context.Context, encounterID string,
		req *models.ActionRequest, userID int) (*models.ActionResponse, error)
}
