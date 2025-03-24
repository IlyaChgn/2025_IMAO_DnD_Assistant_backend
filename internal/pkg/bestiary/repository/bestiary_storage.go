package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var defaultBooks = []string{
	"DMG", "MM", "VGM", "XGE", "MTF", "AAtM", "TCE", "FTD", "MPMM", "MCV1", "tVD", "BGG", "BMT",
	"MPP", "RoT", "LMoP", "HotDQ", "PotA", "OotA", "COS", "SKT", "TOA", "KKW", "WDMM", "WDH",
	"HftT", "BGDIA", "AI", "GOS", "OoW", "IDRotF", "CM", "WBtW", "JttRC", "LoX", "MCV2DC",
	"DSotDQ", "DoSI", "CRCotN", "PaBTSO", "ToFW", "KftGV", "HAT", "DitLCoT", "VEoR", "QftIS",
	"DIP", "GGR", "ERLW", "MOT", "SCC", "VRGR", "BAM", "MCV4EC", "UA22GO", "UAMoS",
	"UA22WotM", "MHH", "ODL", "EGtW", "GHtPG", "TDCS", "VSoS", "TLtRW", "DoDk",
	"MCV3MC", "BotJR", "CoN", "LH", "PG", "CoA", "LHEX",
}

type bestiaryStorage struct {
	db *mongo.Database
}

func NewBestiaryStorage(db *mongo.Database) bestiaryinterfaces.BestiaryRepository {
	return &bestiaryStorage{
		db: db,
	}
}

func excludeBooks(defaultBooks, requestedBooks []string) []string {
	if len(requestedBooks) == 0 {
		return defaultBooks
	}

	requestedMap := make(map[string]bool)
	for _, book := range requestedBooks {
		requestedMap[book] = true
	}

	var result []string
	for _, book := range defaultBooks {
		if !requestedMap[book] {
			result = append(result, book)
		}
	}

	return result
}

func createMovingFilter(moving []string) bson.E {
	// Маппинг для соответствия значений
	mapping := map[string]string{
		"летает":  "летая",
		"парит":   "парит", // Теперь "парит" остается как есть
		"лазает":  "лазая",
		"плавает": "плавая",
		"копает":  "копая",
	}

	// Преобразуем значения из moving в соответствующие значения из mapping
	var mappedValues []string
	for _, move := range moving {
		if mappedValue, ok := mapping[move]; ok {
			mappedValues = append(mappedValues, mappedValue)
		}
	}

	// Создаем фильтр для MongoDB
	if len(mappedValues) > 0 {
		return bson.E{
			Key: "speed",
			Value: bson.M{
				"$elemMatch": bson.M{
					"$or": []bson.M{
						{"name": bson.M{"$in": mappedValues}},       // Фильтр по полю name
						{"additional": bson.M{"$in": mappedValues}}, // Фильтр по полю additional
					},
				},
			},
		}
	}

	// Если mappedValues пуст, возвращаем пустой фильтр
	return bson.E{}
}

func buildMongoFilter(filter models.FilterParams) bson.D {
	mongoFilter := bson.D{}

	// Обрабатываем поле "book"
	if len(filter.Book) > 0 {
		booksToInclude := excludeBooks(defaultBooks, filter.Book)
		mongoFilter = append(mongoFilter, bson.E{Key: "source.shortName", Value: bson.M{"$nin": booksToInclude}})
	} else {
		mongoFilter = append(mongoFilter, bson.E{Key: "source.shortName", Value: bson.M{"$in": defaultBooks}})
	}

	// Фильтр по NPC (если есть поле, связанное с NPC)
	if len(filter.Npc) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "npc", Value: bson.M{"$in": filter.Npc}})
	}

	// Фильтр по типу (type.name)
	if len(filter.Type) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "type.name", Value: bson.M{"$in": filter.Type}})
	}

	// Фильтр по рейтингу сложности (challengeRating)
	if len(filter.ChallengeRating) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "challengeRating", Value: bson.M{"$in": filter.ChallengeRating}})
	}

	// Фильтр по размеру (size.eng)
	if len(filter.Size) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "size.rus", Value: bson.M{"$in": filter.Size}})
	}

	// Фильтр по тегам (если есть поле, связанное с тегами)
	if len(filter.Tag) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "tags", Value: bson.M{"$in": filter.Tag}})
	}

	// Фильтр по движению (если есть поле, связанное с движением)
	if len(filter.Moving) > 0 {
		movingFilter := createMovingFilter(filter.Moving)
		if movingFilter.Key != "" { // Проверяем, что фильтр не пустой
			mongoFilter = append(mongoFilter, movingFilter)
		}
	}

	// Фильтр по чувствам (senses.senses.name)
	if len(filter.Senses) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "senses.senses.name", Value: bson.M{"$in": filter.Senses}})
	}

	// Фильтр по уязвимостям (если есть поле, связанное с уязвимостями)
	if len(filter.VulnerabilityDamage) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "damageVulnerabilities", Value: bson.M{"$in": filter.VulnerabilityDamage}})
	}

	// Фильтр по сопротивлениям (если есть поле, связанное с сопротивлениями)
	if len(filter.ResistanceDamage) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "damageResistances", Value: bson.M{"$in": filter.ResistanceDamage}})
	}

	// Фильтр по иммунитетам к урону (если есть поле, связанное с иммунитетами)
	if len(filter.ImmunityDamage) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "damageImmunities", Value: bson.M{"$in": filter.ImmunityDamage}})
	}

	// Фильтр по иммунитетам к состояниям (если есть поле, связанное с иммунитетами)
	if len(filter.ImmunityCondition) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "conditionImmunities", Value: bson.M{"$in": filter.ImmunityCondition}})
	}

	// Фильтр по особенностям (feats.name)
	if len(filter.Features) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "feats.name", Value: bson.M{"$in": filter.Features}})
	}

	// Фильтр по окружению (environment)
	if len(filter.Environment) > 0 {
		mongoFilter = append(mongoFilter, bson.E{Key: "environment", Value: bson.M{"$in": filter.Environment}})
	}

	return mongoFilter
}

func (s *bestiaryStorage) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams) ([]*models.BestiaryCreature, error) {
	creaturesCollection := s.db.Collection("creatures")

	// Формируем фильтр для MongoDB
	mongoFilter := buildMongoFilter(filter)

	// Добавляем поиск по имени (если указан)
	if search.Value != "" {
		hasRussian := false
		hasEnglish := false

		for _, r := range search.Value {
			if (r >= 'а' && r <= 'я') || (r >= 'А' && r <= 'Я') || r == 'ё' || r == 'Ё' {
				hasRussian = true
			} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				hasEnglish = true
			}

			if hasRussian && hasEnglish {
				return nil, apperrors.FindMongoDataErr
			}
		}

		field := "name.rus"
		if hasEnglish {
			field = "name.eng"
		}

		mongoFilter = append(mongoFilter, bson.E{Key: field, Value: bson.M{"$regex": search.Value, "$options": "i"}})
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

func (s *bestiaryStorage) GetCreatureByEngName(ctx context.Context, url string) (*models.Creature, error) {
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
