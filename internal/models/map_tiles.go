package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Один тайл
type MapTile struct {
	ID       string `json:"id"       bson:"id"`
	Name     string `json:"name"     bson:"name"`
	ImageURL string `json:"imageUrl" bson:"imageUrl"`
}

// Категория тайлов
type MapTileCategory struct {
	// Mongo ID, наружу не отдаем (может пригодиться внутри)
	MongoID primitive.ObjectID `json:"-"       bson:"_id,omitempty"`
	// Внешний бизнес-ID категории, который ожидает фронт (например "village", "dungeon")
	ID    string    `json:"id"    bson:"id"`
	Name  string    `json:"name"  bson:"name"`
	Tiles []MapTile `json:"tiles" bson:"tiles"`
	// Для фильтрации видимости; во внешнем JSON не нужен
	UserID string `json:"-" bson:"userID"`
}

// SerializedEdge represents a single edge with blocking properties
type SerializedEdge struct {
	Key       string `json:"key"       bson:"key"`
	MoveBlock bool   `json:"moveBlock" bson:"moveBlock"`
	LosBlock  bool   `json:"losBlock"  bson:"losBlock"`
}

// TileWalkability содержит данные проходимости и окклюзии для одного тайла
type TileWalkability struct {
	MongoID     primitive.ObjectID `json:"-"           bson:"_id,omitempty"`
	TileID      string             `json:"tileId"      bson:"tileId"`
	SetID       string             `json:"setId"       bson:"setId"`
	Rows        int                `json:"rows"        bson:"rows"`
	Cols        int                `json:"cols"        bson:"cols"`
	Walkability [][]int            `json:"walkability" bson:"walkability"`
	Occlusion   [][]int            `json:"occlusion"   bson:"occlusion"`
	Edges       []SerializedEdge   `json:"edges"       bson:"edges"`
}

// CreateTileRequest — body for POST /api/map-tiles
type CreateTileRequest struct {
	CategoryID string  `json:"categoryId"`
	Tile       MapTile `json:"tile"`
}

// UpdateTileRequest — body for PUT /api/map-tiles/{tileId}
type UpdateTileRequest struct {
	CategoryID string  `json:"categoryId"`
	Tile       MapTile `json:"tile"`
}
