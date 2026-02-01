package usecases

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

type LLMUsecase struct {
	storage                    bestiaryinterface.LLMJobRepository
	geminiAPI                  bestiaryinterface.GeminiAPI
	generatedCreatureProcessor bestiaryinterface.GeneratedCreatureProcessorUsecases
	runner                     bestiaryinterface.AsyncRunner
	idGen                      bestiaryinterface.IDGenerator
}

func NewLLMUsecase(storage bestiaryinterface.LLMJobRepository,
	geminiAPI bestiaryinterface.GeminiAPI,
	generatedCreatureProcessor bestiaryinterface.GeneratedCreatureProcessorUsecases,
	runner bestiaryinterface.AsyncRunner,
	idGen bestiaryinterface.IDGenerator) *LLMUsecase {
	return &LLMUsecase{
		storage:                    storage,
		geminiAPI:                  geminiAPI,
		generatedCreatureProcessor: generatedCreatureProcessor,
		runner:                     runner,
		idGen:                      idGen,
	}
}

func (uc *LLMUsecase) SubmitText(ctx context.Context, desc string) (string, error) {
	l := logger.FromContext(ctx)
	id := uc.idGen.NewID()
	job := &models.LLMJob{
		ID:          id,
		Description: &desc,
		Status:      "pending",
	}

	if err := uc.storage.Create(ctx, job); err != nil {
		l.UsecasesError(err, 0, nil)
		return "", err
	}

	uc.runner.Go(func() { uc.process(ctx, id) })

	return id, nil
}

func (uc *LLMUsecase) SubmitImage(ctx context.Context, img []byte) (string, error) {
	l := logger.FromContext(ctx)
	id := uc.idGen.NewID()
	job := &models.LLMJob{
		ID:     id,
		Image:  img,
		Status: "pending",
	}

	if err := uc.storage.Create(ctx, job); err != nil {
		l.UsecasesError(err, 0, nil)
		return "", err
	}

	uc.runner.Go(func() { uc.process(ctx, id) })

	return id, nil
}

func (uc *LLMUsecase) GetJob(ctx context.Context, id string) (*models.LLMJob, error) {
	return uc.storage.Get(ctx, id)
}

func (uc *LLMUsecase) process(ctx context.Context, id string) {
	l := logger.FromContext(ctx)

	job, err := uc.storage.Get(ctx, id)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return
	}

	job.Status = "processing_step_1"
	if err := uc.storage.Update(ctx, job); err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return
	}

	var raw map[string]interface{}

	// Генерация сущности из описания или изображения
	if job.Description != nil {
		raw, err = uc.geminiAPI.GenerateFromDescription(ctx, *job.Description)
	} else {
		raw, err = uc.geminiAPI.GenerateFromImage(ctx, job.Image)
	}

	if err != nil {
		job.Status = "error"
		l.UsecasesError(err, 0, map[string]any{"id": id})
		_ = uc.storage.Update(ctx, job)
		return
	}

	var cr models.Creature

	// Преобразуем map → JSON → Creature
	b, err := json.Marshal(raw)
	if err != nil {
		job.Status = "error"
		l.UsecasesError(err, 0, map[string]any{"id": id})
		_ = uc.storage.Update(ctx, job)
		return
	}

	if err := json.Unmarshal(b, &cr); err != nil {
		job.Status = "error"
		l.UsecasesError(err, 0, map[string]any{"id": id})
		_ = uc.storage.Update(ctx, job)
		return
	}

	job.Status = "processing_step_2"
	if err := uc.storage.Update(ctx, job); err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return
	}

	// Обработка/валидация через Processor
	processed, err := uc.generatedCreatureProcessor.ValidateAndProcessGeneratedCreature(ctx, &cr)
	if err != nil {
		job.Status = "error"
		_ = uc.storage.Update(ctx, job)
		return
	}

	job.Status = "done"
	job.Result = processed
	_ = uc.storage.Update(ctx, job)
}
