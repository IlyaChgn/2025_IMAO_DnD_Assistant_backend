package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
)

var defaultBooks = []string{
	"DMG", "MM", "VGM", "XGE", "MTF", "AAtM", "TCE", "FTD", "MPMM", "MCV1", "tVD", "BGG", "BMT",
	"MPP", "RoT", "LMoP", "HotDQ", "PotA", "OotA", "COS", "SKT", "TOA", "KKW", "WDMM", "WDH",
	"HftT", "BGDIA", "AI", "GOS", "OoW", "IDRotF", "CM", "WBtW", "JttRC", "LoX", "MCV2DC",
	"DSotDQ", "DoSI", "CRCotN", "PaBTSO", "ToFW", "KftGV", "HAT", "DitLCoT", "VEoR", "QftIS",
	"DIP", "GGR", "ERLW", "MOT", "SCC", "VRGR", "BAM", "MCV4EC", "UA22GO", "UAMoS",
	"UA22WotM", "MHH", "ODL", "EGtW", "GHtPG", "TDCS", "VSoS", "TLtRW", "DoDk",
	"MCV3MC", "BotJR", "CoN", "LH", "PG", "CoA", "LHEX", "HB",
}

type bestiaryStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewBestiaryStorage(db *mongo.Database, metrics mymetrics.DBMetrics) bestiaryinterfaces.BestiaryRepository {
	return &bestiaryStorage{
		db:      db,
		metrics: metrics,
	}
}

func (s *bestiaryStorage) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams) ([]*models.BestiaryCreature, error) {
	l := logger.FromContext(ctx)
	filters := buildTypesFilters(filter)

	if search.Value != "" {
		field, isCorrect := detectLanguageField(search.Value)
		if !isCorrect {
			return nil, nil
		}

		filters = append(filters, bson.E{Key: field, Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	findOptions, err := buildFindOptions(start, size, order)
	if err != nil {
		l.RepoError(err, map[string]any{"start": start, "size": size})
		return nil, err
	}

	return s.getCreaturesList(ctx, filters, findOptions, false)
}

func (s *bestiaryStorage) GetUserCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams, userID int) ([]*models.BestiaryCreature, error) {
	l := logger.FromContext(ctx)
	filters := buildTypesFilters(filter)

	if search.Value != "" {
		field, isCorrect := detectLanguageField(search.Value)
		if !isCorrect {
			return nil, nil
		}

		filters = append(filters, bson.E{Key: field, Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	filters = append(filters, bson.E{Key: "userID", Value: strconv.Itoa(userID)})

	findOptions, err := buildFindOptions(start, size, order)
	if err != nil {
		l.RepoError(err, map[string]any{"start": start, "size": size})
		return nil, err
	}

	return s.getCreaturesList(ctx, filters, findOptions, true)
}

func (s *bestiaryStorage) GetCreatureByEngName(ctx context.Context, url string,
	isUserCollection bool) (*models.Creature, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var collection *mongo.Collection

	if isUserCollection {
		collection = s.db.Collection("generated_creatures")
	} else {
		collection = s.db.Collection("creatures")
	}

	filter := bson.M{"url": fmt.Sprintf("/bestiary/%s", url)}
	var creature models.Creature

	_, err := dbcall.DBCall[*models.Creature](fnName, s.metrics, func() (*models.Creature, error) {
		err := collection.FindOne(ctx, filter).Decode(&creature)
		return &creature, err
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, map[string]any{"url": url})
		return nil, nil
	} else if err != nil {
		l.RepoError(err, map[string]any{"url": url})
		return nil, apperrors.FindMongoDataErr
	}

	return &creature, nil
}

func (s *bestiaryStorage) getCreaturesList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions, isUserCollection bool) ([]*models.BestiaryCreature, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var collection *mongo.Collection

	if isUserCollection {
		collection = s.db.Collection("generated_creatures")
	} else {
		collection = s.db.Collection("creatures")
	}

	var allCreatures []*models.BestiaryCreature

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, filters, findOptions)
	})
	if err != nil && errors.Is(err, mongo.ErrNoDocuments) {
		l.RepoWarn(err, nil)
		return nil, nil
	} else if err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var creature models.BestiaryCreature

		if err := cursor.Decode(&creature); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}

		allCreatures = append(allCreatures, &creature)
	}

	return allCreatures, nil
}

func (s *bestiaryStorage) AddGeneratedCreature(ctx context.Context, creature models.Creature) error {
	creaturesCollection := s.db.Collection("generated_creatures")
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err := creaturesCollection.InsertOne(ctx, creature)
		return err
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": creature.ID})
		return apperrors.InsertMongoDataErr
	}

	return nil
}
