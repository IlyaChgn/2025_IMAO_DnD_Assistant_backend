package usecases

import "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"

type crTier int

const (
	tierLow  crTier = iota // CR 0–4
	tierMid                // CR 5–10
	tierHigh               // CR 11–16
	tierEpic               // CR 17+
)

func crToTier(cr int) crTier {
	switch {
	case cr <= 4:
		return tierLow
	case cr <= 10:
		return tierMid
	case cr <= 16:
		return tierHigh
	default:
		return tierEpic
	}
}

// rarityWeight stores cumulative weights (out of 100) for weighted random selection.
type rarityWeight struct {
	rarity models.ItemRarity
	cumul  int // cumulative percentage boundary
}

var tierRarityWeights = map[crTier][]rarityWeight{
	tierLow: {
		{models.ItemRarityCommon, 70},
		{models.ItemRarityUncommon, 95},
		{models.ItemRarityRare, 100},
	},
	tierMid: {
		{models.ItemRarityCommon, 30},
		{models.ItemRarityUncommon, 70},
		{models.ItemRarityRare, 95},
		{models.ItemRarityVeryRare, 100},
	},
	tierHigh: {
		{models.ItemRarityUncommon, 15},
		{models.ItemRarityRare, 55},
		{models.ItemRarityVeryRare, 90},
		{models.ItemRarityLegendary, 100},
	},
	tierEpic: {
		{models.ItemRarityRare, 15},
		{models.ItemRarityVeryRare, 50},
		{models.ItemRarityLegendary, 90},
		{models.ItemRarityArtifact, 100},
	},
}

type itemCountRange struct {
	min int
	max int
}

var tierItemCounts = map[crTier]itemCountRange{
	tierLow:  {1, 4},
	tierMid:  {2, 6},
	tierHigh: {3, 8},
	tierEpic: {4, 10},
}

type coinRange struct {
	min int
	max int
}

type tierCoins struct {
	cp coinRange
	sp coinRange
	gp coinRange
	pp coinRange
}

var tierCoinTables = map[crTier]tierCoins{
	tierLow:  {cp: coinRange{100, 600}, sp: coinRange{10, 60}, gp: coinRange{0, 20}, pp: coinRange{0, 0}},
	tierMid:  {cp: coinRange{0, 200}, sp: coinRange{50, 300}, gp: coinRange{20, 200}, pp: coinRange{0, 10}},
	tierHigh: {cp: coinRange{0, 0}, sp: coinRange{0, 0}, gp: coinRange{200, 2000}, pp: coinRange{10, 100}},
	tierEpic: {cp: coinRange{0, 0}, sp: coinRange{0, 0}, gp: coinRange{2000, 20000}, pp: coinRange{100, 1000}},
}
