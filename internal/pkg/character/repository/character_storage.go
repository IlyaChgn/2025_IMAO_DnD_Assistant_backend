package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type characterStorage struct {
	db *mongo.Database
}

func NewCharacterStorage(db *mongo.Database) characterinterfaces.CharacterRepository {
	return &characterStorage{
		db: db,
	}
}

func (s *characterStorage) GetCharactersList(ctx context.Context, size, start int, order []models.Order,
	filter models.CharacterFilterParams, search models.SearchParams) ([]*models.CharacterShort, error) {

	filters := buildFilters(filter)

	if search.Value != "" {
		filters = append(filters,
			bson.E{Key: "data.name", Value: bson.M{"$regex": search.Value, "$options": "i"}})
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

	return s.getCharactersList(ctx, filters, findOptions)
}

func (s *characterStorage) GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error) {
	collection := s.db.Collection("characters")

	primitiveId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.InvalidIDErr
	}

	filter := bson.M{"_id": primitiveId}

	var character models.Character

	err = collection.FindOne(ctx, filter).Decode(&character)
	if err != nil {
		log.Println(err)

		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}

	return &character, nil
}

func (s *characterStorage) AddCharacter(ctx context.Context, rawChar models.CharacterRaw) error {
	creaturesCollection := s.db.Collection("characters")

	cleanedData := utils.RemoveBackslashes(rawChar.Data)

	var characterData models.CharacterData

	err := json.Unmarshal([]byte(cleanedData), &characterData)
	if err != nil {
		return apperrors.UnmarashallingJSONError
	}

	character := models.Character{
		ID:             primitive.NewObjectID(),
		Tags:           rawChar.Tags,
		DisabledBlocks: rawChar.DisabledBlocks,
		Spells:         rawChar.Spells,
		Data:           characterData,
		JsonType:       rawChar.JsonType,
		Version:        rawChar.Version,
	}

	_, err = creaturesCollection.InsertOne(ctx, character)
	if err != nil {
		return apperrors.InsertMongoDataErr
	}

	return nil
}

func (s *characterStorage) getCharactersList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions) ([]*models.CharacterShort, error) {
	creaturesCollection := s.db.Collection("characters")

	cursor, err := creaturesCollection.Find(ctx, filters, findOptions)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.NoDocsErr
		}

		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var charactersShort []*models.CharacterShort

	for cursor.Next(ctx) {
		var character models.Character
		if err := cursor.Decode(&character); err != nil {
			log.Println(err)
			return nil, apperrors.DecodeMongoDataErr
		}
		characterShort := models.CharacterShort{
			ID:             character.ID,
			CharClass:      character.Data.Info.CharClass,
			CharacterLevel: character.Data.Info.Level,
			CharacterName:  character.Data.Name,
			CharacterRace:  character.Data.Info.Race,
			Avatar:         character.Data.Avatar,
		}

		charactersShort = append(charactersShort, &characterShort)
	}

	return charactersShort, nil
}
