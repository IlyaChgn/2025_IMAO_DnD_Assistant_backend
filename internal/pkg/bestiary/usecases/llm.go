package usecases

import (
	"context"
	"encoding/json"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
	"github.com/google/uuid"
)

type LLMUsecase struct {
	storage   bestiaryinterface.LLMJobRepository
	geminiAPI bestiaryinterface.GeminiAPI
}

func NewLLMUsecase(store bestiaryinterface.LLMJobRepository, cli bestiaryinterface.GeminiAPI) *LLMUsecase {
	return &LLMUsecase{storage: store, geminiAPI: cli}
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
	uc.storage.Update(ctx, job)

	var (
		raw map[string]interface{}
		cr  models.Creature
	)

	if job.Description != nil {
		raw, err = uc.geminiAPI.GenerateFromDescription(*job.Description)
	} else {
		raw, err = uc.geminiAPI.GenerateFromImage(job.Image)
	}

	if err != nil {
		job.Status = "error"
	} else {
		// маршалим map → JSON → твой Creature
		b, _ := json.Marshal(raw)
		if uerr := json.Unmarshal(b, &cr); uerr != nil {
			job.Status = "error"
		} else {
			job.Status = "done"
			job.Result = &cr
		}
	}

	uc.storage.Update(ctx, job)
}
