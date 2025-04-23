package repository

import (
	generatedcreatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/statblockgenerator"
	"go.mongodb.org/mongo-driver/mongo"
)

type generatedCreatureStorage struct {
	db *mongo.Database
}

func NewCreatureStorage(db *mongo.Database) generatedcreatureinterfaces.GeneratedCreatureRepository {
	return &generatedCreatureStorage{
		db: db,
	}
}
