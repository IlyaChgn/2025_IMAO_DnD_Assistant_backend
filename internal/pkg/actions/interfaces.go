package actions

import (
	"context"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type ActionsUsecases interface {
	ExecuteAction(ctx context.Context, encounterID string,
		req *models.ActionRequest, userID int) (*models.ActionResponse, error)
	GetActionLog(ctx context.Context, encounterID string, userID int,
		limit int, before time.Time) ([]*models.AuditLogEntry, error)
}

// AuditLogRepository handles persistence for the action_log MongoDB collection.
type AuditLogRepository interface {
	Insert(ctx context.Context, entry *models.AuditLogEntry) error
	GetByEncounterID(ctx context.Context, encounterID string, limit int, before time.Time) ([]*models.AuditLogEntry, error)
	EnsureIndexes(ctx context.Context) error
}
