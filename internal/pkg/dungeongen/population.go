package dungeongen

import "math/rand"

// TrapType describes the kind of trap.
type TrapType string

const (
	TrapPit          TrapType = "pit"
	TrapDart         TrapType = "dart"
	TrapPoisonGas    TrapType = "poison_gas"
	TrapFallingRocks TrapType = "falling_rocks"
	TrapAlarm        TrapType = "alarm"
	TrapGlyph        TrapType = "glyph"
)

// TrapSetup describes a single trap placed in a room.
type TrapSetup struct {
	TrapType    TrapType `json:"trapType"`
	DetectionDC int      `json:"detectionDC"`
	DisarmDC    int      `json:"disarmDC"`
	Damage      string   `json:"damage"` // dice expression or "" for alarm
	Position    [2]int   `json:"position"`
}

// SecretSetup describes a hidden feature in a room.
type SecretSetup struct {
	SecretType  string `json:"secretType"` // "hidden_passage"
	DetectionDC int    `json:"detectionDC"`
	Position    [2]int `json:"position"`
}

// LootItem describes a single item in a loot table.
type LootItem struct {
	Name   LocalizedString `json:"name"`
	Count  int             `json:"count"`
	Rarity string          `json:"rarity"` // common, uncommon, rare, very_rare, legendary
}

// LootTable describes loot found in a room.
type LootTable struct {
	Gold      int        `json:"gold"`
	Items     []LootItem `json:"items"`
	Container string     `json:"container"` // "chest", "scattered", "hidden"
}

// LocalizedString holds text in multiple languages.
type LocalizedString struct {
	En string `json:"en"`
	Ru string `json:"ru"`
}

// PopulationResult holds all room-level content assignments.
type PopulationResult struct {
	Loot       map[string]*LootTable      `json:"loot"`
	Traps      map[string][]TrapSetup     `json:"traps"`
	Secrets    map[string][]SecretSetup   `json:"secrets"`
	Narratives map[string]LocalizedString `json:"narratives"`
}

// ThemeDefinition describes a dungeon theme with creature types, traps, and narratives.
type ThemeDefinition struct {
	Theme                  string                         `json:"theme"`
	PrimaryCreatureTypes   []string                       `json:"primaryCreatureTypes"`
	SecondaryCreatureTypes []string                       `json:"secondaryCreatureTypes"`
	TrapTypes              []TrapType                     `json:"trapTypes"`
	TrapDCRange            [2]int                         `json:"trapDCRange"` // [min, max]
	Narratives             map[RoomType][]LocalizedString `json:"narratives"`
}

// trapDamage maps trap type + tier (0=levels 1-4, 1=levels 5+) to damage dice.
var trapDamage = map[TrapType][2]string{
	TrapPit:          {"1d6", "2d6"},
	TrapDart:         {"1d4+2", "2d4+4"},
	TrapPoisonGas:    {"1d8", "2d8"},
	TrapFallingRocks: {"2d6", "3d6"},
	TrapAlarm:        {"", ""},
	TrapGlyph:        {"2d6", "3d8"},
}

// partyTier returns 0 for levels 1-4, 1 for levels 5+.
func partyTier(level int) int {
	if level <= 4 {
		return 0
	}
	return 1
}

// PopulateRooms assigns loot, traps, secrets, and narratives to all rooms.
func PopulateRooms(
	graph *DungeonGraph,
	theme *ThemeDefinition,
	partyLevel int,
	secretRoomReveal bool,
	rng *rand.Rand,
) *PopulationResult {
	result := &PopulationResult{
		Loot:       make(map[string]*LootTable),
		Traps:      make(map[string][]TrapSetup),
		Secrets:    make(map[string][]SecretSetup),
		Narratives: make(map[string]LocalizedString),
	}

	tier := partyTier(partyLevel)

	for _, room := range graph.Rooms {
		// Loot
		if loot := generateLoot(room, partyLevel, rng); loot != nil {
			result.Loot[room.ID] = loot
		}

		// Traps
		if traps := generateTraps(room, theme, tier, rng); len(traps) > 0 {
			result.Traps[room.ID] = traps
		}

		// Secrets
		if secrets := generateSecrets(room, secretRoomReveal, rng); len(secrets) > 0 {
			result.Secrets[room.ID] = secrets
		}

		// Narratives
		if theme != nil {
			if narrs, ok := theme.Narratives[room.Type]; ok && len(narrs) > 0 {
				result.Narratives[room.ID] = narrs[rng.Intn(len(narrs))]
			}
		}
	}

	return result
}

// generateLoot creates loot for treasure and boss rooms.
func generateLoot(room DungeonRoom, partyLevel int, rng *rand.Rand) *LootTable {
	switch room.Type {
	case RoomTreasure:
		gold := (partyLevel*10 + rng.Intn(partyLevel*20+1))
		return &LootTable{
			Gold:      gold,
			Items:     generateLootItems(partyLevel, rng),
			Container: "chest",
		}
	case RoomBoss:
		gold := (partyLevel*20 + rng.Intn(partyLevel*40+1))
		return &LootTable{
			Gold:      gold,
			Items:     generateLootItems(partyLevel+2, rng),
			Container: "scattered",
		}
	default:
		return nil
	}
}

// generateLootItems creates a random set of loot items based on level.
func generateLootItems(level int, rng *rand.Rand) []LootItem {
	itemCount := 1 + rng.Intn(3) // 1-3 items
	items := make([]LootItem, 0, itemCount)

	rarities := []string{"common", "uncommon"}
	if level >= 5 {
		rarities = append(rarities, "rare")
	}
	if level >= 8 {
		rarities = append(rarities, "very_rare")
	}

	for i := 0; i < itemCount; i++ {
		rarity := rarities[rng.Intn(len(rarities))]
		items = append(items, LootItem{
			Name:   lootItemName(rarity, rng),
			Count:  1,
			Rarity: rarity,
		})
	}

	return items
}

// lootItemName returns a localized name for a loot item.
func lootItemName(rarity string, rng *rand.Rand) LocalizedString {
	commonItems := []LocalizedString{
		{En: "Healing Potion", Ru: "Зелье лечения"},
		{En: "Torch Bundle", Ru: "Связка факелов"},
		{En: "Rope (50 ft)", Ru: "Верёвка (15 м)"},
	}
	uncommonItems := []LocalizedString{
		{En: "Potion of Greater Healing", Ru: "Зелье большого лечения"},
		{En: "Scroll of Protection", Ru: "Свиток защиты"},
		{En: "Bag of Holding", Ru: "Сумка хранения"},
	}
	rareItems := []LocalizedString{
		{En: "Flame Tongue Sword", Ru: "Меч огненного языка"},
		{En: "Cloak of Displacement", Ru: "Плащ смещения"},
		{En: "Ring of Protection", Ru: "Кольцо защиты"},
	}
	veryRareItems := []LocalizedString{
		{En: "Staff of Power", Ru: "Посох силы"},
		{En: "Amulet of Health", Ru: "Амулет здоровья"},
	}

	var pool []LocalizedString
	switch rarity {
	case "common":
		pool = commonItems
	case "uncommon":
		pool = uncommonItems
	case "rare":
		pool = rareItems
	case "very_rare":
		pool = veryRareItems
	default:
		pool = commonItems
	}

	return pool[rng.Intn(len(pool))]
}

// generateTraps creates traps for trap rooms.
func generateTraps(room DungeonRoom, theme *ThemeDefinition, tier int, rng *rand.Rand) []TrapSetup {
	if room.Type != RoomTrap {
		return nil
	}
	if theme == nil || len(theme.TrapTypes) == 0 {
		return nil
	}

	trapCount := 1 + rng.Intn(2) // 1-2 traps
	traps := make([]TrapSetup, 0, trapCount)

	dcMin := theme.TrapDCRange[0]
	dcMax := theme.TrapDCRange[1]
	if dcMax <= dcMin {
		dcMax = dcMin + 1
	}

	for i := 0; i < trapCount; i++ {
		trapType := theme.TrapTypes[rng.Intn(len(theme.TrapTypes))]
		detectionDC := dcMin + rng.Intn(dcMax-dcMin+1)
		disarmDC := detectionDC + rng.Intn(4) // 0-3 higher

		damage := trapDamage[trapType][tier]

		pos := randomInteriorCell(room.Bounds, rng)

		traps = append(traps, TrapSetup{
			TrapType:    trapType,
			DetectionDC: detectionDC,
			DisarmDC:    disarmDC,
			Damage:      damage,
			Position:    pos,
		})
	}

	return traps
}

// generateSecrets creates secrets for secret rooms.
func generateSecrets(room DungeonRoom, secretRoomReveal bool, rng *rand.Rand) []SecretSetup {
	if room.Type != RoomSecret {
		return nil
	}

	detectionDC := 13 + rng.Intn(5) // 13-17
	if secretRoomReveal {
		detectionDC -= 3
		if detectionDC < 10 {
			detectionDC = 10
		}
	}

	pos := randomWallCell(room.Bounds, rng)

	return []SecretSetup{{
		SecretType:  "hidden_passage",
		DetectionDC: detectionDC,
		Position:    pos,
	}}
}

// randomInteriorCell returns a cell away from walls.
func randomInteriorCell(bounds RoomBounds, rng *rand.Rand) [2]int {
	minR, maxR := 1, bounds.Rows-2
	if maxR < minR {
		maxR = minR
	}
	minC, maxC := 1, bounds.Cols-2
	if maxC < minC {
		maxC = minC
	}
	return [2]int{
		minR + rng.Intn(maxR-minR+1),
		minC + rng.Intn(maxC-minC+1),
	}
}

// randomWallCell returns a cell on the perimeter (not corner).
func randomWallCell(bounds RoomBounds, rng *rand.Rand) [2]int {
	side := rng.Intn(4)
	midR := 1
	if bounds.Rows > 3 {
		midR = 1 + rng.Intn(bounds.Rows-2)
	}
	midC := 1
	if bounds.Cols > 3 {
		midC = 1 + rng.Intn(bounds.Cols-2)
	}

	switch side {
	case 0: // top
		return [2]int{0, midC}
	case 1: // bottom
		return [2]int{bounds.Rows - 1, midC}
	case 2: // left
		return [2]int{midR, 0}
	default: // right
		return [2]int{midR, bounds.Cols - 1}
	}
}

// --- Predefined themes ---

// DefaultThemes contains the 8 dungeon themes.
var DefaultThemes = map[string]*ThemeDefinition{
	"catacombs": {
		Theme:                  "catacombs",
		PrimaryCreatureTypes:   []string{"undead"},
		SecondaryCreatureTypes: []string{"construct"},
		TrapTypes:              []TrapType{TrapDart, TrapGlyph},
		TrapDCRange:            [2]int{12, 18},
		Narratives: map[RoomType][]LocalizedString{
			RoomEntrance: {
				{En: "A cold draft seeps from the ancient stone entrance.", Ru: "Холодный сквозняк проникает из древнего каменного входа."},
			},
			RoomCombat: {
				{En: "Bones crunch underfoot. Something stirs in the shadows ahead.", Ru: "Кости хрустят под ногами. Что-то шевелится в тенях впереди."},
			},
			RoomTreasure: {
				{En: "A dusty sarcophagus holds offerings long forgotten.", Ru: "Пыльный саркофаг хранит давно забытые подношения."},
			},
			RoomTrap: {
				{En: "Ancient glyphs glow faintly on the floor tiles.", Ru: "Древние глифы слабо светятся на напольных плитах."},
			},
			RoomRest: {
				{En: "A quiet alcove, undisturbed for centuries.", Ru: "Тихая ниша, нетронутая веками."},
			},
			RoomBoss: {
				{En: "The crypt lord awaits on a throne of bones.", Ru: "Повелитель склепа ожидает на троне из костей."},
			},
			RoomSecret: {
				{En: "A hidden passage behind a crumbling wall.", Ru: "Потайной ход за рассыпающейся стеной."},
			},
		},
	},
	"cave": {
		Theme:                  "cave",
		PrimaryCreatureTypes:   []string{"beast"},
		SecondaryCreatureTypes: []string{"monstrosity"},
		TrapTypes:              []TrapType{TrapPit, TrapFallingRocks},
		TrapDCRange:            [2]int{10, 15},
		Narratives: map[RoomType][]LocalizedString{
			RoomEntrance: {
				{En: "The cave mouth yawns before you, swallowing the light.", Ru: "Пасть пещеры разверзается перед вами, поглощая свет."},
			},
			RoomCombat: {
				{En: "A growl echoes off the cavern walls. You are not alone.", Ru: "Рычание отражается от стен пещеры. Вы не одни."},
			},
			RoomBoss: {
				{En: "The great beast's lair. Bones of past victims litter the ground.", Ru: "Логово великого зверя. Кости прошлых жертв усеивают землю."},
			},
		},
	},
	"fortress": {
		Theme:                  "fortress",
		PrimaryCreatureTypes:   []string{"humanoid"},
		SecondaryCreatureTypes: []string{"construct"},
		TrapTypes:              []TrapType{TrapDart, TrapAlarm},
		TrapDCRange:            [2]int{12, 17},
		Narratives: map[RoomType][]LocalizedString{
			RoomEntrance: {
				{En: "The fortress gates stand ajar, inviting and threatening at once.", Ru: "Ворота крепости стоят приоткрытыми, одновременно приглашая и угрожая."},
			},
			RoomCombat: {
				{En: "Guards patrol this section. Weapons at the ready.", Ru: "Стражники патрулируют этот участок. Оружие наготове."},
			},
			RoomBoss: {
				{En: "The warlord's command room. Battle plans cover every surface.", Ru: "Командная комната полководца. Боевые планы покрывают каждую поверхность."},
			},
		},
	},
	"temple": {
		Theme:                  "temple",
		PrimaryCreatureTypes:   []string{"undead", "construct"},
		SecondaryCreatureTypes: []string{"fiend"},
		TrapTypes:              []TrapType{TrapGlyph, TrapPoisonGas},
		TrapDCRange:            [2]int{14, 20},
		Narratives: map[RoomType][]LocalizedString{
			RoomCombat: {
				{En: "Corrupted guardians still defend these sacred halls.", Ru: "Испорченные стражи всё ещё защищают эти священные залы."},
			},
			RoomBoss: {
				{En: "The inner sanctum. A corrupted high priest channels dark power.", Ru: "Внутреннее святилище. Испорченный первосвященник направляет тёмную силу."},
			},
		},
	},
	"sewer": {
		Theme:                  "sewer",
		PrimaryCreatureTypes:   []string{"ooze", "aberration"},
		SecondaryCreatureTypes: []string{"beast"},
		TrapTypes:              []TrapType{TrapPit, TrapPoisonGas},
		TrapDCRange:            [2]int{11, 16},
		Narratives: map[RoomType][]LocalizedString{
			RoomCombat: {
				{En: "Something moves beneath the filthy water. Multiple somethings.", Ru: "Что-то движется под грязной водой. Несколько чего-то."},
			},
		},
	},
	"mine": {
		Theme:                  "mine",
		PrimaryCreatureTypes:   []string{"construct", "elemental"},
		SecondaryCreatureTypes: []string{"beast"},
		TrapTypes:              []TrapType{TrapFallingRocks, TrapPit},
		TrapDCRange:            [2]int{10, 14},
		Narratives: map[RoomType][]LocalizedString{
			RoomCombat: {
				{En: "Animated mining constructs still follow their last orders — defend the vein.", Ru: "Оживлённые горные конструкты всё ещё следуют своим последним приказам — защищать жилу."},
			},
		},
	},
	"crypt": {
		Theme:                  "crypt",
		PrimaryCreatureTypes:   []string{"undead"},
		SecondaryCreatureTypes: []string{"fiend"},
		TrapTypes:              []TrapType{TrapGlyph, TrapDart},
		TrapDCRange:            [2]int{13, 19},
		Narratives: map[RoomType][]LocalizedString{
			RoomCombat: {
				{En: "The tombs open. The honored dead rise as twisted mockeries.", Ru: "Гробницы открываются. Почитаемые мертвецы восстают искажёнными пародиями."},
			},
			RoomBoss: {
				{En: "The crypt keeper emerges from the central tomb.", Ru: "Хранитель склепа выходит из центральной гробницы."},
			},
		},
	},
	"forest_ruin": {
		Theme:                  "forest_ruin",
		PrimaryCreatureTypes:   []string{"fey", "beast"},
		SecondaryCreatureTypes: []string{"plant"},
		TrapTypes:              []TrapType{TrapPit, TrapAlarm},
		TrapDCRange:            [2]int{10, 15},
		Narratives: map[RoomType][]LocalizedString{
			RoomCombat: {
				{En: "The forest spirits do not welcome intruders. Thorns and claws await.", Ru: "Лесные духи не приветствуют вторженцев. Шипы и когти ждут."},
			},
		},
	},
}
