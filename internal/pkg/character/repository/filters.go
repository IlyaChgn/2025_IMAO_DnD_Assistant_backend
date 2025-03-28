package repository

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func buildFilters(filter models.CharacterFilterParams) bson.D {
	return bson.D{}
}
