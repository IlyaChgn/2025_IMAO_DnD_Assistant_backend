package repository

import (
	"context"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type bestiaryStorage struct {
	db *mongo.Database
}

func NewBestiaryStorage(db *mongo.Database) bestiaryinterfaces.BestiaryRepository {
	return &bestiaryStorage{
		db: db,
	}
}

func (s *bestiaryStorage) GetCreaturesList(ctx context.Context, size, start int) ([]*models.BestiaryCreature, error) {
	creaturesCollection := s.db.Collection("creatures")

	filter := bson.D{}

	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(start))

	cursor, err := creaturesCollection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Println(err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var creatures []*models.BestiaryCreature

	for cursor.Next(ctx) {
		var creature models.BestiaryCreature

		if err := cursor.Decode(&creature); err != nil {
			log.Println(err)
			return nil, apperrors.DecodeMongoDataErr
		}

		creatures = append(creatures, &creature)
	}

	return creatures, nil
}
