package seed

import (
	"context"
	_ "embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
)

//go:embed srd_spells.json
var srdSpellsJSON []byte

// SeedSpellDefinitions upserts SRD spell definitions into the database.
// Returns the number of spells successfully upserted.
func SeedSpellDefinitions(ctx context.Context, repo spellsinterfaces.SpellsRepository) (int, error) {
	var spells []models.SpellDefinition
	if err := json.Unmarshal(srdSpellsJSON, &spells); err != nil {
		return 0, err
	}

	upserted := 0

	for i := range spells {
		spells[i].SchemaVersion = 1

		if err := repo.UpsertSpell(ctx, &spells[i]); err != nil {
			return upserted, err
		}
		upserted++
	}

	return upserted, nil
}
