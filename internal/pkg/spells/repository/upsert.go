package repository

import (
	"context"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *spellsStorage) UpsertSpell(ctx context.Context, spell *models.SpellDefinition) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(spellDefinitionsCollection)

	filter := bson.D{{Key: "engName", Value: spell.EngName}}
	update := bson.D{{Key: "$set", Value: spell}}
	opts := options.Update().SetUpsert(true)

	_, err := dbcall.DBCall[*mongo.UpdateResult](fnName, s.metrics, func() (*mongo.UpdateResult, error) {
		return collection.UpdateOne(ctx, filter, update, opts)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"engName": spell.EngName})
		return err
	}

	return nil
}
