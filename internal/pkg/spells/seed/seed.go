package seed

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	spellsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells"
)

// Spell data is split into per-level JSON files for easier maintenance.
// The glob pattern matches: srd_spells_level0.json .. srd_spells_level6plus.json
//
//go:embed srd_spells_*.json
var spellFiles embed.FS

// combinedSpellsJSON holds the merged JSON array of all per-level files.
// Built once at init; panics on malformed data (same guarantee as single-file embed).
var combinedSpellsJSON []byte

func init() {
	entries, err := spellFiles.ReadDir(".")
	if err != nil {
		panic("spells/seed: failed to read embedded directory: " + err.Error())
	}

	var all []json.RawMessage

	for _, entry := range entries {
		data, err := spellFiles.ReadFile(entry.Name())
		if err != nil {
			panic("spells/seed: failed to read " + entry.Name() + ": " + err.Error())
		}

		var batch []json.RawMessage
		if err := json.Unmarshal(data, &batch); err != nil {
			panic("spells/seed: failed to parse " + entry.Name() + ": " + err.Error())
		}

		all = append(all, batch...)
	}

	combinedSpellsJSON, err = json.Marshal(all)
	if err != nil {
		panic("spells/seed: failed to marshal combined spells: " + err.Error())
	}
}

// SRDSpellsJSON returns the combined SRD spells JSON data from all per-level files.
// Used by cmd/seed_spells to avoid duplicating the JSON files.
func SRDSpellsJSON() []byte {
	return combinedSpellsJSON
}

// SeedSpellDefinitions upserts SRD spell definitions into the database.
// Returns the number of spells successfully upserted.
func SeedSpellDefinitions(ctx context.Context, repo spellsinterfaces.SpellsRepository) (int, error) {
	var spells []models.SpellDefinition
	if err := json.Unmarshal(combinedSpellsJSON, &spells); err != nil {
		return 0, err
	}

	changed := 0

	for i := range spells {
		spells[i].SchemaVersion = 1

		modified, err := repo.UpsertSpell(ctx, &spells[i])
		if err != nil {
			return changed, err
		}
		if modified {
			changed++
		}
	}

	return changed, nil
}
