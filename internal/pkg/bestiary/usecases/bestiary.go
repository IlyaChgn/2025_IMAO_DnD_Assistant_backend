package usecases

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

type bestiaryUsecases struct {
	repo bestiaryinterface.BestiaryRepository
}

func NewBestiaryUsecases(repo bestiaryinterface.BestiaryRepository) bestiaryinterface.BestiaryUsecases {
	return &bestiaryUsecases{
		repo: repo,
	}
}

func (uc *bestiaryUsecases) GetCreaturesList(ctx context.Context, size, start int) ([]*models.BestiaryCreature, error) {
	if start < 0 || size <= 0 {
		return nil, apperrors.StartPosSizeError
	}

	return uc.repo.GetCreaturesList(ctx, size, start)
}
