package encounter

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_encounter.go -package=mocks

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

type EncounterRepository interface {
	GetEncountersListWithSearch(ctx context.Context, size, start, userID int,
		search *models.SearchParams) (*models.EncountersList, error)
	GetEncountersList(ctx context.Context, size, start, userID int) (*models.EncountersList, error)
	GetEncounterByID(ctx context.Context, id string) (*models.Encounter, error)
	SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, id string, userID int) error
	UpdateEncounter(ctx context.Context, data []byte, id string) error
	RemoveEncounter(ctx context.Context, id string) error
	CheckPermission(ctx context.Context, id string, userID int) bool
}

type EncounterUsecases interface {
	GetEncountersList(ctx context.Context, size, start, userID int,
		search *models.SearchParams) (*models.EncountersList, error)
	GetEncounterByID(ctx context.Context, id string, userID int) (*models.Encounter, error)
	SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error
	UpdateEncounter(ctx context.Context, data []byte, id string, userID int) error
	RemoveEncounter(ctx context.Context, id string, userID int) error
}
