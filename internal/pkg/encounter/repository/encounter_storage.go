package repository

import (
	"context"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	encounterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/encounter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type encounterStorage struct {
	db *mongo.Database
}

func NewEncounterStorage(db *mongo.Database) encounterinterfaces.EncounterRepository {
	return &encounterStorage{
		db: db,
	}
}

func (s *encounterStorage) GetEncountersList(ctx context.Context, size, start int, order []models.Order,
	filter models.EncounterFilterParams, search models.SearchParams) ([]*models.EncounterShort, error) {

	filters := buildFilters(filter)

	if search.Value != "" {
		filters = append(filters, bson.E{Key: "name", Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(start))

	if len(order) > 0 {
		sort := bson.D{}

		for _, o := range order {
			sort = append(sort, bson.E{Key: o.Field, Value: 1}) // 1 для asc, -1 для desc
		}

		findOptions.SetSort(sort)
	}

	return s.getEncountersList(ctx, filters, findOptions)
}

func (s *encounterStorage) GetEncounterByMongoId(ctx context.Context, id string) (*models.Encounter, error) {
	collection := s.db.Collection("encounters")

	primitiveId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.InvalidIDErr
	}

	filter := bson.M{"_id": primitiveId}

	var encounter models.Encounter

	err = collection.FindOne(ctx, filter).Decode(&encounter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}

	return &encounter, nil
}

func (s *encounterStorage) AddEncounter(ctx context.Context, encounter models.EncounterRaw) error {
	encountersCollection := s.db.Collection("encounters")

	_, err := encountersCollection.InsertOne(ctx, encounter)
	if err != nil {
		return apperrors.InsertMongoDataErr
	}

	return nil
}

func (s *encounterStorage) getEncountersList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions) ([]*models.EncounterShort, error) {
	encountersCollection := s.db.Collection("encounters")

	cursor, err := encountersCollection.Find(ctx, filters, findOptions)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var encountersShort []*models.EncounterShort

	for cursor.Next(ctx) {
		var encounter models.Encounter

		if err := cursor.Decode(&encounter); err != nil {
			return nil, apperrors.DecodeMongoDataErr
		}

		encounterShort := models.EncounterShort{
			ID:            encounter.ID,
			EncounterName: encounter.EncounterName,
		}

		encountersShort = append(encountersShort, &encounterShort)
	}

	return encountersShort, nil
}
