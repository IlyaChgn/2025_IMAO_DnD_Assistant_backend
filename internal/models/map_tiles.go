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
