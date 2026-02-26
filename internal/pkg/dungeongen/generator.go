package dungeongen

import (
	"math/rand"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// GenerationInput holds all inputs needed for dungeon generation.
type GenerationInput struct {
	Config           DungeonConfig
	Difficulty       DungeonDifficulty
	Theme            *ThemeDefinition
	ThemeTags        []string
	SecretRoomReveal bool
	TileMetadata     []*models.TileMetadata
	TileWalkability  map[string]*models.TileWalkability
	Creatures        []CreatureSummary
}

// DungeonResponse is the complete generated dungeon sent to the client.
type DungeonResponse struct {
	Seed             int64                      `json:"seed"`
	Size             DungeonSize                `json:"size"`
	ThemeName        string                     `json:"themeName"`
	Rooms            []DungeonRoom              `json:"rooms"`
	Connections      []RoomConnection           `json:"connections"`
	MainPathLength   int                        `json:"mainPathLength"`
	ExtractionPoints []ExtractionPoint          `json:"extractionPoints"`
	Composition      *MapComposition            `json:"composition"`
	Terrain          *BakedTerrain              `json:"terrain"`
	Encounters       map[string]*EncounterSetup `json:"encounters"`
	Loot             map[string]*LootTable      `json:"loot"`
	Traps            map[string][]TrapSetup     `json:"traps"`
	Secrets          map[string][]SecretSetup   `json:"secrets"`
	Narratives       map[string]LocalizedString `json:"narratives"`
}

// GenerateDungeon runs the full generation pipeline and returns a
// ready-to-serialize response. Every step is deterministic given the
// same seed, so identical inputs always produce the same dungeon.
func GenerateDungeon(input *GenerationInput) *DungeonResponse {
	rng := rand.New(rand.NewSource(input.Config.Seed))

	// B1: Generate abstract graph (rooms + connections)
	graph := GenerateGraph(input.Config, rng)

	// B2: Assign room types (combat, treasure, trap, etc.)
	AssignRoomTypes(graph, rng, input.Config.Size, input.SecretRoomReveal)

	// B3: Assign tiles (select tile + rotation for every node)
	assignments := AssignTiles(graph, input.TileMetadata, input.ThemeTags, rng)

	// B4: Compute physical layout (position tiles on 2D grid)
	composition := ComputeLayout(graph, assignments)

	// B5: Bake terrain (stamp walkability/occlusion, merge edges)
	terrain := BakeTerrain(composition, input.TileWalkability)

	// B6: Budget encounters (select monsters for combat rooms)
	encounters := BudgetEncounters(
		graph,
		input.Creatures,
		input.Config.PartyLevel,
		input.Config.PartySize,
		input.Difficulty,
		rng,
	)

	// B7: Populate rooms (loot, traps, secrets, narratives)
	population := PopulateRooms(
		graph,
		input.Theme,
		input.Config.PartyLevel,
		input.SecretRoomReveal,
		rng,
	)

	themeName := ""
	if input.Theme != nil {
		themeName = input.Theme.Theme
	}

	return &DungeonResponse{
		Seed:             input.Config.Seed,
		Size:             input.Config.Size,
		ThemeName:        themeName,
		Rooms:            graph.Rooms,
		Connections:      graph.Connections,
		MainPathLength:   graph.MainPathLength,
		ExtractionPoints: graph.ExtractionPoints,
		Composition:      composition,
		Terrain:          terrain,
		Encounters:       encounters,
		Loot:             population.Loot,
		Traps:            population.Traps,
		Secrets:          population.Secrets,
		Narratives:       population.Narratives,
	}
}
