package delivery

import (
	generatedcreatureinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/statblockgenerator"
)

type GeneratedCreatureHandler struct {
	usecases generatedcreatureinterfaces.GeneratedCreatureUsecases
}

func NewCreatureHandler(usecases generatedcreatureinterfaces.GeneratedCreatureUsecases) *GeneratedCreatureHandler {
	return &GeneratedCreatureHandler{
		usecases: usecases,
	}
}
