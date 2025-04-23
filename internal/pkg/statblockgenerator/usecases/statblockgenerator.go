package usecases

import (
	generatedcreatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/statblockgenerator"
)

type generatedCreatureUsecases struct {
	repo generatedcreatureinterfaces.GeneratedCreatureRepository
}

func NewCreatureUsecases(repo generatedcreatureinterfaces.GeneratedCreatureRepository) generatedcreatureinterfaces.GeneratedCreatureUsecases {
	return &generatedCreatureUsecases{
		repo: repo,
	}
}
