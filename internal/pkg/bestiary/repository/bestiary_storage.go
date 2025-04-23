package repository

import (
	"context"
	"errors"
	"fmt"

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
	"MCV3MC", "BotJR", "CoN", "LH", "PG", "CoA", "LHEX", "HB",
}

type bestiaryStorage struct {
	db *mongo.Database
}

func NewBestiaryStorage(db *mongo.Database) bestiaryinterfaces.BestiaryRepository {
	return &bestiaryStorage{
		db: db,
	}
}

func (s *bestiaryStorage) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams, searchInSecondCollection bool) ([]*models.BestiaryCreature, error) {

	filters := buildTypesFilters(filter)

	if search.Value != "" {
		field, err := detectLanguageField(search.Value)
		if err != nil {
			return nil, err
		}

		filters = append(filters, bson.E{Key: field, Value: bson.M{"$regex": search.Value, "$options": "i"}})
	}

	findOptions := options.Find()
	findOptions.SetLimit(int64(size))
	findOptions.SetSkip(int64(start))

	if len(order) <= 0 {
		return s.getCreaturesList(ctx, filters, findOptions, searchInSecondCollection)
	}

	sort := bson.D{}
	for _, o := range order {
		var direction int

		if o.Direction == "asc" {
			direction = 1
		} else if o.Direction == "desc" {
			direction = -1
		} else {
			return nil, apperrors.UnknownDirectionError
		}

		sort = append(sort, bson.E{Key: o.Field, Value: direction}) // 1 для asc, -1 для desc
	}

	findOptions.SetSort(sort)

	return s.getCreaturesList(ctx, filters, findOptions, searchInSecondCollection)
}

func (s *bestiaryStorage) GetCreatureByEngName(ctx context.Context, url string,
	searchInSecondCollection bool) (*models.Creature, error) {
	primaryCollection := s.db.Collection("creatures")
	secondaryCollection := s.db.Collection("generated_creatures")

	filter := bson.M{"url": fmt.Sprintf("/bestiary/%s", url)}

	var creature models.Creature

	err := primaryCollection.FindOne(ctx, filter).Decode(&creature)
	if err == nil {
		return &creature, nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) && searchInSecondCollection {
		err = secondaryCollection.FindOne(ctx, filter).Decode(&creature)
		if err == nil {
			return &creature, nil
		}
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, apperrors.NoDocsErr
	}
	return nil, apperrors.FindMongoDataErr
}

func (s *bestiaryStorage) getCreaturesList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions, includeSecondCollection bool) ([]*models.BestiaryCreature, error) {

	creaturesCollection := s.db.Collection("creatures")
	additionalCollection := s.db.Collection("generated_creatures")

	var allCreatures []*models.BestiaryCreature

	findAndAppend := func(collection *mongo.Collection) error {
		cursor, err := collection.Find(ctx, filters, findOptions)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil
			}
			return apperrors.FindMongoDataErr
		}
		defer cursor.Close(ctx)

		for cursor.Next(ctx) {
			var creature models.BestiaryCreature
			if err := cursor.Decode(&creature); err != nil {
				return apperrors.DecodeMongoDataErr
			}
			allCreatures = append(allCreatures, &creature)
		}
		return nil
	}

	if err := findAndAppend(creaturesCollection); err != nil {
		return nil, err
	}

	if includeSecondCollection {
		if err := findAndAppend(additionalCollection); err != nil {
			return nil, err
		}
	}

	return allCreatures, nil
}

func (s *bestiaryStorage) AddGeneratedCreature(ctx context.Context, creature models.Creature) error {
	creaturesCollection := s.db.Collection("generated_creatures")

	_, err := creaturesCollection.InsertOne(ctx, creature)
	if err != nil {
		return apperrors.InsertMongoDataErr
	}

	return nil
}
