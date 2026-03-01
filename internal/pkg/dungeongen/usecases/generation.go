package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiary "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/dungeongen"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	maptiles "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/maptiles"
)

// Standard D&D 5e CR → XP mapping.
var crXPTable = map[string]int{
	"0":   10,
	"1/8": 25,
	"1/4": 50,
	"1/2": 100,
	"1":   200,
	"2":   450,
	"3":   700,
	"4":   1100,
	"5":   1800,
	"6":   2300,
	"7":   2900,
	"8":   3900,
	"9":   5000,
	"10":  5900,
	"11":  7200,
	"12":  8400,
	"13":  10000,
	"14":  11500,
	"15":  13000,
	"16":  15000,
	"17":  18000,
	"18":  20000,
	"19":  22000,
	"20":  25000,
}

var (
	ErrInvalidSize       = errors.New("invalid dungeon size")
	ErrInvalidPartyLevel = errors.New("party level must be between 1 and 10")
	ErrInvalidPartySize  = errors.New("party size must be between 1 and 8")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidTheme      = errors.New("unknown theme")
	ErrNoTileMetadata    = errors.New("no tile metadata available")
)

type dungeonGenUsecases struct {
	tileMetaRepo dungeongen.TileMetadataRepository
	mapTilesRepo maptiles.MapTilesRepository
	bestiaryRepo bestiary.BestiaryRepository
	defaultSetID string
}

// NewDungeonGenUsecases creates a new dungeon generation usecases instance.
func NewDungeonGenUsecases(
	tileMetaRepo dungeongen.TileMetadataRepository,
	mapTilesRepo maptiles.MapTilesRepository,
	bestiaryRepo bestiary.BestiaryRepository,
	defaultSetID string,
) dungeongen.DungeonGenUsecases {
	return &dungeonGenUsecases{
		tileMetaRepo: tileMetaRepo,
		mapTilesRepo: mapTilesRepo,
		bestiaryRepo: bestiaryRepo,
		defaultSetID: defaultSetID,
	}
}

func (uc *dungeonGenUsecases) GenerateDungeon(ctx context.Context, req *dungeongen.GenerateRequest) (*dungeongen.DungeonResponse, error) {
	l := logger.FromContext(ctx)

	// Validate and normalize input
	size, err := validateSize(req.Size)
	if err != nil {
		return nil, err
	}

	difficulty, err := validateDifficulty(req.Difficulty)
	if err != nil {
		return nil, err
	}

	if req.PartyLevel < 1 || req.PartyLevel > 10 {
		return nil, ErrInvalidPartyLevel
	}

	partySize := req.PartySize
	if partySize < 1 || partySize > 8 {
		return nil, ErrInvalidPartySize
	}

	themeName := req.Theme
	if themeName == "" {
		themeName = "catacombs"
	}
	theme, ok := dungeongen.DefaultThemes[themeName]
	if !ok {
		return nil, ErrInvalidTheme
	}

	seed := req.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	// Fetch tile metadata
	tileMetadata, err := uc.tileMetaRepo.GetAll(ctx)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"step": "fetch_tile_metadata"})
		return nil, err
	}
	if len(tileMetadata) == 0 {
		return nil, ErrNoTileMetadata
	}

	// Fetch walkability data
	walkList, err := uc.mapTilesRepo.GetWalkabilityBySetID(ctx, uc.defaultSetID)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"step": "fetch_walkability", "setID": uc.defaultSetID})
		return nil, err
	}
	walkMap := make(map[string]*models.TileWalkability, len(walkList))
	for _, w := range walkList {
		walkMap[w.TileID] = w
	}

	// Fetch creatures for encounter budgeting
	creatures := fetchCreatureSummaries(ctx, uc.bestiaryRepo, l)

	// Determine theme tags from the theme name
	themeTags := dungeongen.ThemeTagsForCategory(themeName)

	input := &dungeongen.GenerationInput{
		Config: dungeongen.DungeonConfig{
			Seed:       seed,
			Size:       size,
			PartyLevel: req.PartyLevel,
			PartySize:  partySize,
		},
		Difficulty:       difficulty,
		Theme:            theme,
		ThemeTags:        themeTags,
		SecretRoomReveal: false,
		TileMetadata:     tileMetadata,
		TileWalkability:  walkMap,
		Creatures:        creatures,
	}

	return dungeongen.GenerateDungeon(input), nil
}

func validateSize(s string) (dungeongen.DungeonSize, error) {
	switch dungeongen.DungeonSize(s) {
	case dungeongen.SizeShort, dungeongen.SizeMedium, dungeongen.SizeLong:
		return dungeongen.DungeonSize(s), nil
	case "":
		return dungeongen.SizeMedium, nil
	default:
		return "", ErrInvalidSize
	}
}

func validateDifficulty(d string) (dungeongen.DungeonDifficulty, error) {
	switch dungeongen.DungeonDifficulty(d) {
	case dungeongen.DifficultyEasy, dungeongen.DifficultyMedium, dungeongen.DifficultyHard, dungeongen.DifficultyDeadly:
		return dungeongen.DungeonDifficulty(d), nil
	case "":
		return dungeongen.DifficultyMedium, nil
	default:
		return "", ErrInvalidDifficulty
	}
}

// fetchCreatureSummaries loads bestiary creatures and converts them to
// lightweight summaries for encounter budgeting. Errors are logged but
// not fatal — the dungeon is generated without encounters if needed.
func fetchCreatureSummaries(ctx context.Context, repo bestiary.BestiaryRepository, l logger.Logger) []dungeongen.CreatureSummary {
	bestiaryCreatures, err := repo.GetCreaturesList(ctx, 1000, 0, nil, models.FilterParams{}, models.SearchParams{})
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"step": "fetch_creatures"})
		return nil
	}

	summaries := make([]dungeongen.CreatureSummary, 0, len(bestiaryCreatures))
	for _, c := range bestiaryCreatures {
		xp, ok := crXPTable[c.ChallengeRating]
		if !ok {
			continue
		}
		summaries = append(summaries, dungeongen.CreatureSummary{
			ID:           c.ID.Hex(),
			Name:         c.Name.Eng,
			CR:           c.ChallengeRating,
			XP:           xp,
			CreatureType: c.Type.Name,
		})
	}

	return summaries
}
