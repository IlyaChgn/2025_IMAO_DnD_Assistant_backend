package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const inventoryContainersCollection = "inventory_containers"

type inventoryStorage struct {
	db      *mongo.Database
	metrics mymetrics.DBMetrics
}

func NewInventoryStorage(db *mongo.Database, metrics mymetrics.DBMetrics) itemsinterfaces.InventoryRepository {
	return &inventoryStorage{db: db, metrics: metrics}
}

func (s *inventoryStorage) EnsureInventoryContainerIndexes(ctx context.Context) error {
	l := logger.FromContext(ctx)
	collection := s.db.Collection(inventoryContainersCollection)

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "encounterId", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "ownerId", Value: 1},
				{Key: "kind", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "items.id", Value: 1}},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		l.RepoError(err, map[string]any{"action": "EnsureInventoryContainerIndexes"})
		return err
	}

	return nil
}

func (s *inventoryStorage) GetContainer(ctx context.Context, id string) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	result, err := dbcall.DBCall[*mongo.SingleResult](fnName, s.metrics, func() (*mongo.SingleResult, error) {
		return collection.FindOne(ctx, bson.D{{Key: "_id", Value: objID}}), nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.FindMongoDataErr
	}

	var container models.InventoryContainer
	if err := result.Decode(&container); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			l.RepoWarn(err, map[string]any{"id": id})
			return nil, apperrors.ContainerNotFoundErr
		}
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.DecodeMongoDataErr
	}

	return &container, nil
}

func (s *inventoryStorage) GetContainers(ctx context.Context, filter models.ContainerFilterParams) ([]*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	mongoFilter := bson.D{}
	if filter.EncounterID != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "encounterId", Value: filter.EncounterID})
	}
	if filter.OwnerID != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "ownerId", Value: filter.OwnerID})
	}
	if filter.Kind != "" {
		mongoFilter = append(mongoFilter, bson.E{Key: "kind", Value: filter.Kind})
	}

	cursor, err := dbcall.DBCall[*mongo.Cursor](fnName, s.metrics, func() (*mongo.Cursor, error) {
		return collection.Find(ctx, mongoFilter)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"filter": filter})
		return nil, apperrors.FindMongoDataErr
	}
	defer cursor.Close(ctx)

	var containers []*models.InventoryContainer
	for cursor.Next(ctx) {
		var c models.InventoryContainer
		if err := cursor.Decode(&c); err != nil {
			l.RepoError(err, nil)
			return nil, apperrors.DecodeMongoDataErr
		}
		containers = append(containers, &c)
	}
	if err := cursor.Err(); err != nil {
		l.RepoError(err, nil)
		return nil, apperrors.FindMongoDataErr
	}

	if containers == nil {
		containers = []*models.InventoryContainer{}
	}

	return containers, nil
}

func (s *inventoryStorage) CreateContainer(ctx context.Context, container *models.InventoryContainer) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	container.Version = 1
	container.UpdatedAt = time.Now()
	if container.Items == nil {
		container.Items = []models.ItemInstance{}
	}

	res, err := dbcall.DBCall[*mongo.InsertOneResult](fnName, s.metrics, func() (*mongo.InsertOneResult, error) {
		return collection.InsertOne(ctx, container)
	})
	if err != nil {
		l.RepoError(err, map[string]any{"name": container.Name})
		return nil, apperrors.UpdateMongoDataErr
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		container.ID = oid
	}

	return container, nil
}

func (s *inventoryStorage) UpdateContainer(ctx context.Context, container *models.InventoryContainer) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	container.UpdatedAt = time.Now()
	container.Version++

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		res, err := collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: container.ID}}, container)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return apperrors.ContainerNotFoundErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.ContainerNotFoundErr) {
			return nil, err
		}
		l.RepoError(err, map[string]any{"id": container.ID.Hex()})
		return nil, apperrors.UpdateMongoDataErr
	}

	return container, nil
}

func (s *inventoryStorage) UpdateContainerWithVersion(ctx context.Context, container *models.InventoryContainer, expectedVersion int) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	container.UpdatedAt = time.Now()
	container.Version = expectedVersion + 1

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		res, err := collection.ReplaceOne(ctx, bson.D{
			{Key: "_id", Value: container.ID},
			{Key: "version", Value: expectedVersion},
		}, container)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return apperrors.VersionConflictErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.VersionConflictErr) {
			return nil, err
		}
		l.RepoError(err, map[string]any{"id": container.ID.Hex(), "expectedVersion": expectedVersion})
		return nil, apperrors.UpdateMongoDataErr
	}

	return container, nil
}

func (s *inventoryStorage) DeleteContainer(ctx context.Context, id string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	collection := s.db.Collection(inventoryContainersCollection)

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		l.RepoWarn(err, map[string]any{"id": id})
		return apperrors.InvalidIDErr
	}

	err = dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		res, err := collection.DeleteOne(ctx, bson.D{{Key: "_id", Value: objID}})
		if err != nil {
			return err
		}
		if res.DeletedCount == 0 {
			return apperrors.ContainerNotFoundErr
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, apperrors.ContainerNotFoundErr) {
			return err
		}
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.DeleteMongoDataErr
	}

	return nil
}

func (s *inventoryStorage) MoveItemAcrossContainers(
	ctx context.Context,
	sourceID string,
	sourceVersion int,
	targetID string,
	itemID string,
	toPlacement models.ItemPlacement,
) (*models.InventoryContainer, *models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	client := s.db.Client()
	collection := s.db.Collection(inventoryContainersCollection)

	sourceObjID, err := primitive.ObjectIDFromHex(sourceID)
	if err != nil {
		return nil, nil, apperrors.InvalidIDErr
	}
	targetObjID, err := primitive.ObjectIDFromHex(targetID)
	if err != nil {
		return nil, nil, apperrors.InvalidIDErr
	}

	var source, target models.InventoryContainer

	session, err := client.StartSession()
	if err != nil {
		log.Printf("Warning: transactions not available, falling back to sequential updates: %v", err)
		return s.moveItemSequential(ctx, collection, sourceObjID, sourceVersion, targetObjID, itemID, toPlacement)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		// Load source with version check
		err := collection.FindOne(sc, bson.D{
			{Key: "_id", Value: sourceObjID},
			{Key: "version", Value: sourceVersion},
		}).Decode(&source)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, apperrors.VersionConflictErr
			}
			return nil, err
		}

		// Find and remove item from source
		var movedItem *models.ItemInstance
		newItems := make([]models.ItemInstance, 0, len(source.Items))
		for i := range source.Items {
			if source.Items[i].ID == itemID {
				item := source.Items[i]
				movedItem = &item
			} else {
				newItems = append(newItems, source.Items[i])
			}
		}
		if movedItem == nil {
			return nil, apperrors.ItemNotInContainerErr
		}
		source.Items = newItems
		source.Version++
		source.UpdatedAt = time.Now()

		res, err := collection.ReplaceOne(sc, bson.D{
			{Key: "_id", Value: sourceObjID},
			{Key: "version", Value: sourceVersion},
		}, &source)
		if err != nil {
			return nil, err
		}
		if res.MatchedCount == 0 {
			return nil, apperrors.VersionConflictErr
		}

		// Load target
		err = collection.FindOne(sc, bson.D{{Key: "_id", Value: targetObjID}}).Decode(&target)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, apperrors.ContainerNotFoundErr
			}
			return nil, err
		}

		// Add item to target
		movedItem.Placement = toPlacement
		movedItem.IsEquipped = false
		movedItem.EquippedSlot = ""
		targetVersion := target.Version
		target.Items = append(target.Items, *movedItem)
		target.Version++
		target.UpdatedAt = time.Now()

		res, err = collection.ReplaceOne(sc, bson.D{
			{Key: "_id", Value: targetObjID},
			{Key: "version", Value: targetVersion},
		}, &target)
		if err != nil {
			return nil, err
		}
		if res.MatchedCount == 0 {
			return nil, apperrors.ContainerNotFoundErr
		}

		return nil, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{
			"sourceID": sourceID,
			"targetID": targetID,
			"itemID":   itemID,
		})
		return nil, nil, err
	}

	return &source, &target, nil
}

// moveItemSequential is a fallback when transactions are not available (no replica set).
func (s *inventoryStorage) moveItemSequential(
	ctx context.Context,
	collection *mongo.Collection,
	sourceObjID primitive.ObjectID,
	sourceVersion int,
	targetObjID primitive.ObjectID,
	itemID string,
	toPlacement models.ItemPlacement,
) (*models.InventoryContainer, *models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	var source models.InventoryContainer
	err := collection.FindOne(ctx, bson.D{
		{Key: "_id", Value: sourceObjID},
		{Key: "version", Value: sourceVersion},
	}).Decode(&source)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, apperrors.VersionConflictErr
		}
		l.RepoError(err, nil)
		return nil, nil, apperrors.FindMongoDataErr
	}

	var movedItem *models.ItemInstance
	newItems := make([]models.ItemInstance, 0, len(source.Items))
	for i := range source.Items {
		if source.Items[i].ID == itemID {
			item := source.Items[i]
			movedItem = &item
		} else {
			newItems = append(newItems, source.Items[i])
		}
	}
	if movedItem == nil {
		return nil, nil, apperrors.ItemNotInContainerErr
	}
	source.Items = newItems
	source.Version++
	source.UpdatedAt = time.Now()

	res, err := collection.ReplaceOne(ctx, bson.D{
		{Key: "_id", Value: sourceObjID},
		{Key: "version", Value: sourceVersion},
	}, &source)
	if err != nil {
		l.RepoError(err, nil)
		return nil, nil, apperrors.UpdateMongoDataErr
	}
	if res.MatchedCount == 0 {
		return nil, nil, apperrors.VersionConflictErr
	}

	var target models.InventoryContainer
	err = collection.FindOne(ctx, bson.D{{Key: "_id", Value: targetObjID}}).Decode(&target)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, apperrors.ContainerNotFoundErr
		}
		l.RepoError(err, nil)
		return nil, nil, apperrors.FindMongoDataErr
	}

	movedItem.Placement = toPlacement
	movedItem.IsEquipped = false
	movedItem.EquippedSlot = ""
	targetVersion := target.Version
	target.Items = append(target.Items, *movedItem)
	target.Version++
	target.UpdatedAt = time.Now()

	res, err = collection.ReplaceOne(ctx, bson.D{
		{Key: "_id", Value: targetObjID},
		{Key: "version", Value: targetVersion},
	}, &target)
	if err != nil {
		l.RepoError(err, nil)
		return nil, nil, apperrors.UpdateMongoDataErr
	}
	if res.MatchedCount == 0 {
		return nil, nil, apperrors.VersionConflictErr
	}

	return &source, &target, nil
}
