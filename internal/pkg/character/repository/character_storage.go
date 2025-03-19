package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
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

func buildMongoFilter(filter models.CharacterFilterParams) bson.D {

	_ = filter

	mongoFilter := bson.D{}

	return mongoFilter
}

// Функция для удаления всех вхождений символа `\` из строки
func removeBackslashes(input string) string {
	return strings.ReplaceAll(input, "\\", "")
}

func (s *characterStorage) GetCharactersList(ctx context.Context, size, start int, order []models.Order, filter models.CharacterFilterParams,
	search models.SearchParams) ([]*models.CharacterShort, error) {
	creaturesCollection := s.db.Collection("characters")

	mongoFilter := buildMongoFilter(filter)

	// Добавляем поиск по имени (если указан)
	if search.Value != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "data.name", Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(start))

	// Добавляем сортировку (если указана)
	if len(order) > 0 {
		sort := bson.D{}
		for _, o := range order {
			sort = append(sort, bson.E{Key: o.Field, Value: 1}) // 1 для asc, -1 для desc
		}
		findOptions.SetSort(sort)
	}

	cursor, err := creaturesCollection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		log.Println(err)
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

func (s *characterStorage) GetCharacterByMongoId(ctx context.Context, id string) (*models.Character, error) {
	collection := s.db.Collection("characters")

	primitiveId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Fatal(err)
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

	// Очищаем поле data от обратных слэшей (если нужно)
	cleanedData := removeBackslashes(rawChar.Data)

	// Анмаршалим очищенную строку data в структуру CharacterData
	var characterData models.CharacterData
	err := json.Unmarshal([]byte(cleanedData), &characterData)
	if err != nil {
		fmt.Printf("Ошибка при анмаршалинге поля data: %v\n", err) // NEED TO MOOVE TO APPERRORS

		return err
	}

	character := models.Character{
		Tags:           rawChar.Tags,
		DisabledBlocks: rawChar.DisabledBlocks,
		Spells:         rawChar.Spells,
		Data:           characterData,
		JsonType:       rawChar.JsonType,
		Version:        rawChar.Version,
	}

	// Записываем структуру в коллекцию
	insertResult, err := creaturesCollection.InsertOne(context.TODO(), character)
	if err != nil {
		fmt.Printf("ошибка при записи в базу: %v\n", err) // NEED TO MOOVE TO APPERRORS

		return err
	}

	fmt.Println("Inserted document with ID:", insertResult.InsertedID)

	return nil
}
