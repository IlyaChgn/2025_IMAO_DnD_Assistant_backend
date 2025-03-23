package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
)

type creatureStorage struct {
	db *mongo.Database
}

func NewCreatureStorage(db *mongo.Database) creatureinterfaces.CreatureRepository {
	return &creatureStorage{
		db: db,
	}
}

func (s *creatureStorage) GetCreatureByEngName(ctx context.Context, url string) (*models.Creature, error) {
	collection := s.db.Collection("creatures")

	filter := bson.M{"url": fmt.Sprintf("/bestiary/%s", url)}

	var creature models.Creature

	err := collection.FindOne(ctx, filter).Decode(&creature)
	if err != nil {
		log.Println(err)

		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}

	return &creature, nil
}
