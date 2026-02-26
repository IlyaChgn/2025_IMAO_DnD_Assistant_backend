package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// TileRole describes the functional role of a tile in a dungeon layout.
type TileRole string

const (
	TileRoleRoom      TileRole = "room"
	TileRoleCorridorH TileRole = "corridor_h"
	TileRoleCorridorV TileRole = "corridor_v"
	TileRoleCorner    TileRole = "corner"
	TileRoleJunctionT TileRole = "junction_t"
	TileRoleJunctionX TileRole = "junction_x"
	TileRoleDeadEnd   TileRole = "dead_end"
	TileRoleOpen      TileRole = "open"
	TileRoleWall      TileRole = "wall"
)

// OpeningSummary indicates which sides of a tile have walkable openings.
type OpeningSummary struct {
	Top    bool `json:"top"    bson:"top"`
	Right  bool `json:"right"  bson:"right"`
	Bottom bool `json:"bottom" bson:"bottom"`
	Left   bool `json:"left"   bson:"left"`
}

// EdgeSignatures stores 6-bit binary strings for each side at each rotation.
// 4 sides × 4 rotations = 16 strings total.
type EdgeSignatures struct {
	// Rotation 0 (original)
	Top    string `json:"top"    bson:"top"`
	Right  string `json:"right"  bson:"right"`
	Bottom string `json:"bottom" bson:"bottom"`
	Left   string `json:"left"   bson:"left"`
	// Rotation 1 (90° CCW)
	R1Top    string `json:"r1Top"    bson:"r1Top"`
	R1Right  string `json:"r1Right"  bson:"r1Right"`
	R1Bottom string `json:"r1Bottom" bson:"r1Bottom"`
	R1Left   string `json:"r1Left"   bson:"r1Left"`
	// Rotation 2 (180°)
	R2Top    string `json:"r2Top"    bson:"r2Top"`
	R2Right  string `json:"r2Right"  bson:"r2Right"`
	R2Bottom string `json:"r2Bottom" bson:"r2Bottom"`
	R2Left   string `json:"r2Left"   bson:"r2Left"`
	// Rotation 3 (270° CCW)
	R3Top    string `json:"r3Top"    bson:"r3Top"`
	R3Right  string `json:"r3Right"  bson:"r3Right"`
	R3Bottom string `json:"r3Bottom" bson:"r3Bottom"`
	R3Left   string `json:"r3Left"   bson:"r3Left"`
}

// TileMetadata stores pre-computed classification data for a single map tile.
type TileMetadata struct {
	MongoID        primitive.ObjectID `json:"-"              bson:"_id,omitempty"`
	TileID         string             `json:"tileId"         bson:"tileId"`
	Role           TileRole           `json:"role"           bson:"role"`
	ThemeTags      []string           `json:"themeTags"      bson:"themeTags"`
	WalkableRatio  float64            `json:"walkableRatio"  bson:"walkableRatio"`
	Openings       OpeningSummary     `json:"openings"       bson:"openings"`
	EdgeSignatures EdgeSignatures     `json:"edgeSignatures" bson:"edgeSignatures"`
	AutoClassified bool               `json:"autoClassified" bson:"autoClassified"`
}
