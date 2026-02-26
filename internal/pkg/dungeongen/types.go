package dungeongen

// DungeonSize controls the overall dungeon length.
type DungeonSize string

const (
	SizeShort  DungeonSize = "short"
	SizeMedium DungeonSize = "medium"
	SizeLong   DungeonSize = "long"
)

// SizeRange defines the min/max total room count for a dungeon size.
type SizeRange struct{ Min, Max int }

// SizeRanges maps dungeon sizes to their room count ranges.
var SizeRanges = map[DungeonSize]SizeRange{
	SizeShort:  {8, 10},
	SizeMedium: {11, 13},
	SizeLong:   {14, 16},
}

// RoomType describes the gameplay purpose of a room.
type RoomType string

const (
	RoomEntrance       RoomType = "entrance"
	RoomCombat         RoomType = "combat"
	RoomCombatOptional RoomType = "combat_optional"
	RoomTreasure       RoomType = "treasure"
	RoomTrap           RoomType = "trap"
	RoomRest           RoomType = "rest"
	RoomBoss           RoomType = "boss"
	RoomExtraction     RoomType = "extraction"
	RoomSecret         RoomType = "secret"
)

// GraphPosition is the room's position on the abstract graph.
// X = index along main path, Y = 0 for main path, +1/-1 for branches.
type GraphPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// RoomBounds defines the physical placement of a room/corridor on the map.
type RoomBounds struct {
	OriginRow int `json:"originRow"`
	OriginCol int `json:"originCol"`
	Rows      int `json:"rows"`
	Cols      int `json:"cols"`
}

// DungeonRoom represents a single room in the dungeon graph.
type DungeonRoom struct {
	ID            string        `json:"id"`
	Type          RoomType      `json:"type"`
	GraphPosition GraphPosition `json:"graphPosition"`
	Bounds        RoomBounds    `json:"bounds"`
	Discovered    bool          `json:"discovered"`
}

// RoomConnection links two rooms, optionally through a corridor.
type RoomConnection struct {
	ID             string     `json:"id"`
	FromRoomID     string     `json:"fromRoomId"`
	ToRoomID       string     `json:"toRoomId"`
	CorridorBounds RoomBounds `json:"corridorBounds"`
}

// ExtractionPoint is a location where the party can exit the dungeon.
type ExtractionPoint struct {
	RoomID             string `json:"roomId"`
	Type               string `json:"type"` // "entrance" or "boss_exit"
	InitiallyAvailable bool   `json:"initiallyAvailable"`
}

// DungeonGraph is the complete abstract dungeon structure.
type DungeonGraph struct {
	Rooms            []DungeonRoom    `json:"rooms"`
	Connections      []RoomConnection `json:"connections"`
	MainPathLength   int              `json:"mainPathLength"`
	ExtractionPoints []ExtractionPoint `json:"extractionPoints"`
}

// DungeonConfig holds the input parameters for dungeon generation.
type DungeonConfig struct {
	Seed       int64       `json:"seed"`
	Size       DungeonSize `json:"size"`
	PartyLevel int         `json:"partyLevel"`
	PartySize  int         `json:"partySize"`
}

// TileAssignment maps a graph node (room or connection) to a physical tile.
type TileAssignment struct {
	NodeID   string `json:"nodeId"`   // room ID or connection ID
	TileID   string `json:"tileId"`   // reference to MapTile.ID
	Rotation int    `json:"rotation"` // 0, 1, 2, 3
}

// TileSize is the side length of every macrotile in cells.
const TileSize = 6

// MacroTilePlacement describes one tile placed on the final map grid.
type MacroTilePlacement struct {
	TileID    string `json:"tileId"`
	NodeID    string `json:"nodeId"`    // room ID or connection ID
	OriginRow int    `json:"originRow"`
	OriginCol int    `json:"originCol"`
	Rotation  int    `json:"rotation"`
}

// MapComposition is the physical map: total dimensions + all tile placements.
type MapComposition struct {
	Rows       int                  `json:"rows"`
	Cols       int                  `json:"cols"`
	Placements []MacroTilePlacement `json:"placements"`
}

