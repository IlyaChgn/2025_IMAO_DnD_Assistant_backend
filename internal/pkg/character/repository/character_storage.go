package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"strconv"

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
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewCharacterStorage(db *mongo.Database, metrics mymetrics.DBMetrics) characterinterfaces.CharacterRepository {
	return &characterStorage{
		db:      db,
		metrics: metrics,
	}
}

func (s *characterStorage) GetCharactersList(ctx context.Context, size, start, userID int,
	search models.SearchParams) ([]*models.CharacterShort, error) {

	filters := bson.D{}

	if search.Value != "" {
		filters = append(filters,
			bson.E{Key: "data.name.value", Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	possibleIds := []string{"*", strconv.Itoa(userID)}
	filters = append(filters, bson.E{Key: "userID", Value: bson.M{"$in": possibleIds}})

	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(start))

	return s.getCharactersList(ctx, filters, findOptions)
}

func (s *characterStorage) GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection("characters")

	primitiveId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	filter := bson.M{"_id": primitiveId}

	var character models.Character

	_, err = dbcall.DBCall[*models.Character](fnName, s.metrics, func() (*models.Character, error) {
		err = collection.FindOne(ctx, filter).Decode(&character)
		return &character, err
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, map[string]any{"id": id})
		return nil, nil
	} else if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.FindMongoDataErr
	}

	return &character, nil
}

func (s *characterStorage) AddCharacter(ctx context.Context, rawChar models.CharacterRaw, userID int) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	creaturesCollection := s.db.Collection("characters")

	cleanedData := utils.RemoveBackslashes(rawChar.Data)

	var characterData models.CharacterData

	err := json.Unmarshal([]byte(cleanedData), &characterData)
	if err != nil {
		l.RepoError(err, nil)
		return apperrors.UnmarashallingJSONError
	}

	character := models.Character{
		ID:             primitive.NewObjectID(),
		UserID:         strconv.Itoa(userID),
		Tags:           rawChar.Tags,
		DisabledBlocks: rawChar.DisabledBlocks,
		Spells:         rawChar.Spells,
		Data:           characterData,
		JsonType:       rawChar.JsonType,
		Version:        rawChar.Version,
	}

	err = dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err = creaturesCollection.InsertOne(ctx, character)
		return err
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": character.ID})
		return apperrors.InsertMongoDataErr
	}

	return nil
}

func (s *characterStorage) getCharactersList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions) ([]*models.CharacterShort, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	creaturesCollection := s.db.Collection("characters")

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return creaturesCollection.Find(ctx, filters, findOptions)
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, nil)
		return nil, apperrors.NoDocsErr
	} else if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var charactersShort []*models.CharacterShort

	for cursor.Next(ctx) {
		var character models.Character
		if err := cursor.Decode(&character); err != nil {
			l.RepoError(err, map[string]any{"id": character.ID})
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
