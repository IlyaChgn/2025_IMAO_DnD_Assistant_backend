package repository

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	creatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/creature"
)

// Структура репозитория для работы с существами в MongoDB
type mongoCreatureRepository struct {
	db *mongo.Database
}

// NewMongoDBCreatureRepository создает новый репозиторий для работы с существами
func NewMongoDBCreatureRepository(db *mongo.Database) creatureinterfaces.CreatureRepository {
	return &mongoCreatureRepository{db: db}
}

// GetCreatureByEngName возвращает существо по полю url
func (r *mongoCreatureRepository) GetCreatureByEngName(ctx context.Context, url string) (*models.Creature, error) {
	// Выбираем коллекцию "creatures"
	collection := r.db.Collection("creatures")

	// Создаем фильтр для поиска по полю "url"
	filter := bson.M{"url": "/bestiary/" + url}

	// Выполняем запрос
	var creature models.Creature
	err := collection.FindOne(ctx, filter).Decode(&creature)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Если существо не найдено, возвращаем nil и ошибку
			return nil, fmt.Errorf("creature with url '/bestiary/%s' not found", url)
		}
		// В случае других ошибок возвращаем их
		return nil, fmt.Errorf("mongo db error: %v", err)
	}

	return &creature, nil
}
