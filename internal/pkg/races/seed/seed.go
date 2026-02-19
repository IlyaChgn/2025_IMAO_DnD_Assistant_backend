package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	racesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/races"
)

//go:embed srd_races.json
var srdRacesJSON []byte

// SeedRaceDefinitions upserts SRD race definitions into the database.
// Returns the number of races successfully upserted.
func SeedRaceDefinitions(ctx context.Context, repo racesinterfaces.RacesRepository) (int, error) {
	var races []models.RaceDefinition
	if err := json.Unmarshal(srdRacesJSON, &races); err != nil {
		return 0, err
	}

	upserted := 0

	for i := range races {
		races[i].SchemaVersion = 1

		if err := repo.UpsertRace(ctx, &races[i]); err != nil {
			return upserted, err
		}
		upserted++
	}

	return upserted, nil
}
