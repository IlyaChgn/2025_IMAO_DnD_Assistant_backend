package usecases

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/google/uuid"
)

type LLMUsecase struct {
	storage                    bestiaryinterface.LLMJobRepository
	geminiAPI                  bestiaryinterface.GeminiAPI
	generatedCreatureProcessor bestiaryinterface.GeneratedCreatureProcessorUsecases
}

func NewLLMUsecase(storage bestiaryinterface.LLMJobRepository,
	geminiAPI bestiaryinterface.GeminiAPI,
	generatedCreatureProcessor bestiaryinterface.GeneratedCreatureProcessorUsecases) *LLMUsecase {
	return &LLMUsecase{storage: storage, geminiAPI: geminiAPI, generatedCreatureProcessor: generatedCreatureProcessor}
}

func (uc *LLMUsecase) SubmitText(ctx context.Context, desc string) (string, error) {
	id := uuid.New().String()
	job := &models.LLMJob{
		ID:          id,
		Description: &desc,
		Status:      "pending",
	}
	if err := uc.storage.Create(ctx, job); err != nil {
		return "", err
	}
	go uc.process(ctx, id)
	return id, nil
}

// Image
func (uc *LLMUsecase) SubmitImage(ctx context.Context, img []byte) (string, error) {
	id := uuid.New().String()
	job := &models.LLMJob{
		ID:     id,
		Image:  img,
		Status: "pending",
	}
	if err := uc.storage.Create(ctx, job); err != nil {
		return "", err
	}
	go uc.process(ctx, id)
	return id, nil
}

func (uc *LLMUsecase) GetJob(ctx context.Context, id string) (*models.LLMJob, error) {
	return uc.storage.Get(ctx, id)
}

func (uc *LLMUsecase) process(ctx context.Context, id string) {
	job, err := uc.storage.Get(ctx, id)
	if err != nil {
		return
	}

	job.Status = "processing"
	if err := uc.storage.Update(ctx, job); err != nil {
		return
	}

	var raw map[string]interface{}

	// Генерация сущности из описания или изображения
	if job.Description != nil {
		raw, err = uc.geminiAPI.GenerateFromDescription(*job.Description)
	} else {
		raw, err = uc.geminiAPI.GenerateFromImage(job.Image)
	}

	if err != nil {
		job.Status = "error"
		_ = uc.storage.Update(ctx, job)
		return
	}

	// Преобразуем map → JSON → Creature
	var cr models.Creature
	b, err := json.Marshal(raw)
	if err != nil {
		job.Status = "error"
		_ = uc.storage.Update(ctx, job)
		return
	}

	if err := json.Unmarshal(b, &cr); err != nil {
		job.Status = "error"
		_ = uc.storage.Update(ctx, job)
		return
	}

	// Обработка/валидация через Processor
	processed, err := uc.generatedCreatureProcessor.ValidateAndProcessGeneratedCreature(&cr)
	if err != nil {
		job.Status = "error"
		_ = uc.storage.Update(ctx, job)
		return
	}

	job.Status = "done"
	job.Result = processed
	_ = uc.storage.Update(ctx, job)
}
