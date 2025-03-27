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
	"log"
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

func (s *bestiaryStorage) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams) ([]*models.BestiaryCreature, error) {

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

	if len(order) > 0 {
		sort := bson.D{}
		for _, o := range order {
			sort = append(sort, bson.E{Key: o.Field, Value: 1}) // 1 для asc, -1 для desc
		}

		findOptions.SetSort(sort)
	}

	return s.getCreaturesList(ctx, filters, findOptions)
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

func (s *bestiaryStorage) getCreaturesList(ctx context.Context, filters bson.D,
	findOptions *options.FindOptions) ([]*models.BestiaryCreature, error) {
	creaturesCollection := s.db.Collection("creatures")

	cursor, err := creaturesCollection.Find(ctx, filters, findOptions)
	if err != nil {
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
			return nil, apperrors.DecodeMongoDataErr
		}

		creatures = append(creatures, &creature)
	}

	return creatures, nil
}
