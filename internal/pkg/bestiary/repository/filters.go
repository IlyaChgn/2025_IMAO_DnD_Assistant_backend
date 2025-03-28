package repository

import (
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

func createMovingFilter(moving []string) bson.E {
	mapping := map[string]string{
		"летает":  "летая",
		"парит":   "парит",
		"лазает":  "лазая",
		"плавает": "плавая",
		"копает":  "копая",
	}

	var mappedValues []string
	for _, move := range moving {
		if mappedValue, ok := mapping[move]; ok {
			mappedValues = append(mappedValues, mappedValue)
		}
	}

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

	return bson.E{}
}

func buildTypesFilters(filter models.FilterParams) bson.D {
	mongoFilter := bson.D{}

	// Обрабатываем поле "book"
	mongoFilter = append(mongoFilter, filterBook(filter.Book)...)

	// Фильтр по NPC (если есть поле, связанное с NPC)
	mongoFilter = append(mongoFilter, filterIn("npc", filter.Npc)...)

	// Фильтр по типу (type.name)
	mongoFilter = append(mongoFilter, filterIn("type.name", filter.Type)...)

	// Фильтр по рейтингу сложности (challengeRating)
	mongoFilter = append(mongoFilter, filterIn("challengeRating", filter.ChallengeRating)...)

	// Фильтр по размеру (size.eng)
	mongoFilter = append(mongoFilter, filterIn("size.rus", filter.Size)...)

	// Фильтр по тегам (если есть поле, связанное с тегами)
	mongoFilter = append(mongoFilter, filterIn("tags", filter.Tag)...)

	// Фильтр по движению (если есть поле, связанное с движением)
	mongoFilter = append(mongoFilter, filterMoving(filter.Moving)...)

	// Фильтр по чувствам (senses.senses.name)
	mongoFilter = append(mongoFilter, filterIn("senses.senses.name", filter.Senses)...)

	// Фильтр по уязвимостям (если есть поле, связанное с уязвимостями)
	mongoFilter = append(mongoFilter, filterIn("damageVulnerabilities", filter.VulnerabilityDamage)...)

	// Фильтр по сопротивлениям (если есть поле, связанное с сопротивлениями)
	mongoFilter = append(mongoFilter, filterIn("damageResistances", filter.ResistanceDamage)...)

	// Фильтр по иммунитетам к урону (если есть поле, связанное с иммунитетами)
	mongoFilter = append(mongoFilter, filterIn("damageImmunities", filter.ImmunityDamage)...)

	// Фильтр по иммунитетам к состояниям (если есть поле, связанное с иммунитетами)
	mongoFilter = append(mongoFilter, filterIn("conditionImmunities", filter.ImmunityCondition)...)

	// Фильтр по особенностям (feats.name)
	mongoFilter = append(mongoFilter, filterIn("feats.name", filter.Features)...)

	// Фильтр по окружению (environment)
	mongoFilter = append(mongoFilter, filterIn("environment", filter.Environment)...)

	return mongoFilter
}

// Обрабатываем поле "book"
func filterBook(book []string) bson.D {
	if len(book) > 0 {
		booksToInclude := excludeBooks(defaultBooks, book)
		return bson.D{{Key: "source.shortName", Value: bson.M{"$nin": booksToInclude}}}
	}
	return bson.D{{Key: "source.shortName", Value: bson.M{"$in": defaultBooks}}}
}

// Фильтр по значению "$in", если поле не пустое
func filterIn(field string, values []string) bson.D {
	if len(values) > 0 {
		return bson.D{{Key: field, Value: bson.M{"$in": values}}}
	}
	return nil
}

// Фильтр по движению (если есть поле, связанное с движением)
func filterMoving(moving []string) bson.D {
	if len(moving) > 0 {
		movingFilter := createMovingFilter(moving)
		if movingFilter.Key != "" { // Проверяем, что фильтр не пустой
			return bson.D{movingFilter}
		}
	}
	return nil
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
