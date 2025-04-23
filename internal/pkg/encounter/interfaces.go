package encounter

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type EncounterRepository interface {
	GetEncountersListWithSearch(ctx context.Context, size, start, userID int,
		search *models.SearchParams) (*models.EncountersList, error)
	GetEncountersList(ctx context.Context, size, start, userID int) (*models.EncountersList, error)
	GetEncounterByID(ctx context.Context, id int) (*models.Encounter, error)
	SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error
	UpdateEncounter(ctx context.Context, data []byte, id int) error
	RemoveEncounter(ctx context.Context, id int) error
	CheckPermission(ctx context.Context, id, userID int) bool
}

type EncounterUsecases interface {
	GetEncountersList(ctx context.Context, size, start, userID int,
		search *models.SearchParams) (*models.EncountersList, error)
	GetEncounterByID(ctx context.Context, id, userID int) (*models.Encounter, error)
	SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error
	UpdateEncounter(ctx context.Context, data []byte, id, userID int) error
	RemoveEncounter(ctx context.Context, id, userID int) error
}
