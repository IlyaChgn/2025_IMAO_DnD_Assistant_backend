package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type bestiaryUsecases struct {
	repo      bestiaryinterface.BestiaryRepository
	s3        bestiaryinterface.BestiaryS3Manager
	geminiAPI bestiaryinterface.GeminiAPI
}

func NewBestiaryUsecases(
	repo bestiaryinterface.BestiaryRepository,
	s3 bestiaryinterface.BestiaryS3Manager,
	geminiAPI bestiaryinterface.GeminiAPI,
) bestiaryinterface.BestiaryUsecases {
	return &bestiaryUsecases{
		repo:      repo,
		s3:        s3,
		geminiAPI: geminiAPI,
	}
}

func (uc *bestiaryUsecases) GetCreaturesList(ctx context.Context, size, start int, order []models.Order,
	filter models.FilterParams, search models.SearchParams) ([]*models.BestiaryCreature, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetCreaturesList(ctx, size, start, order, filter, search, true)
}

func (uc *bestiaryUsecases) GetCreatureByEngName(ctx context.Context, engName string) (*models.Creature, error) {

	creature, err := uc.repo.GetCreatureByEngName(ctx, engName, true)
	if err != nil {
		return nil, err
	}

	return creature, nil
}

func (uc *bestiaryUsecases) AddGeneratedCreature(ctx context.Context,
	creatureInput models.CreatureInput, userID int) error {
	var generatedCreature = creatureInput.Creature // скопировали всё, кроме ID

	if creatureInput.ID == "current" || creatureInput.ID == "" {
		generatedCreature.ID = primitive.NewObjectID()
		generatedCreature.UserID = strconv.Itoa(userID)
	} else {
		objectID, err := primitive.ObjectIDFromHex(creatureInput.ID)
		if err != nil {

			return apperrors.InvalidInputError
		}
		generatedCreature.ID = objectID
	}

	if creatureInput.Name.Eng == "" {
		return apperrors.InvalidInputError
	}

	var stringCreatureId = generatedCreature.ID.Hex()

	generatedCreature.URL = fmt.Sprintf("/bestiary/%s", stringCreatureId)

	var creatureImageRect = creatureInput.ImageBase64
	objectNameRect := "generated-creature-images/processed/" + stringCreatureId + ".webp"

	urlRect, err := uc.s3.UploadImage(creatureImageRect, objectNameRect)
	if err != nil {
		return err
	}

	var creatureImageToken = creatureInput.ImageBase64Circle
	objectNameToken := "generated-creature-images/tokens/" + stringCreatureId + ".webp"

	urlToken, err := uc.s3.UploadImage(creatureImageToken, objectNameToken)
	if err != nil {
		return err
	}

	generatedCreature.Images = append(generatedCreature.Images, urlToken, urlRect, urlRect)

	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// NEED TO WRITE SOME BETTER CHECKS LATER
	// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

	return uc.repo.AddGeneratedCreature(ctx, generatedCreature)
}

func (uc *bestiaryUsecases) ParseCreatureFromImage(ctx context.Context, image []byte) (*models.Creature, error) {
	parsedJSON, err := uc.geminiAPI.GenerateFromImage(image)
	if err != nil {
		return nil, err
	}

	parsedBytes, err := json.Marshal(parsedJSON)
	if err != nil {
		return nil, err
	}

	var creature models.Creature
	if err := json.Unmarshal(parsedBytes, &creature); err != nil {
		return nil, err
	}

	return &creature, nil
}

func (uc *bestiaryUsecases) GenerateCreatureFromDescription(ctx context.Context, description string) (*models.Creature, error) {
	parsedJSON, err := uc.geminiAPI.GenerateFromDescription(description)
	if err != nil {
		return nil, err
	}

	parsedBytes, err := json.Marshal(parsedJSON)
	if err != nil {
		return nil, err
	}

	var creature models.Creature
	err = json.Unmarshal(parsedBytes, &creature)
	if err != nil {
		return nil, err
	}

	return &creature, nil
}
