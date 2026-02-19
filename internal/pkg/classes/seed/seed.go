package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	classesinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/classes"
)

//go:embed srd_classes.json
var srdClassesJSON []byte

// SeedClassDefinitions upserts SRD class definitions into the database.
// Returns the number of classes successfully upserted.
func SeedClassDefinitions(ctx context.Context, repo classesinterfaces.ClassesRepository) (int, error) {
	var classes []models.ClassDefinition
	if err := json.Unmarshal(srdClassesJSON, &classes); err != nil {
		return 0, err
	}

	upserted := 0

	for i := range classes {
		classes[i].SchemaVersion = 1

		if err := repo.UpsertClass(ctx, &classes[i]); err != nil {
			return upserted, err
		}
		upserted++
	}

	return upserted, nil
}
