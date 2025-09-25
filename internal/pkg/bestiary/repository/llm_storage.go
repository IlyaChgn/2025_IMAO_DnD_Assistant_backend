package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"sync"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

// InMemoryLLMRepo — потокобезопасное хранилище в памяти.
type inMemoryLLMStorage struct {
	mu   sync.RWMutex
	jobs map[string]*models.LLMJob
}

// NewInMemoryLLMRepo конструктор.
func NewInMemoryLLMRepo() bestiaryinterfaces.LLMJobRepository {
	return &inMemoryLLMStorage{
		jobs: make(map[string]*models.LLMJob),
	}
}

func (r *inMemoryLLMStorage) Create(ctx context.Context, job *models.LLMJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	r.jobs[job.ID] = job

	return nil
}

func (r *inMemoryLLMStorage) Get(ctx context.Context, id string) (*models.LLMJob, error) {
	l := logger.FromContext(ctx)

	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.jobs[id]
	if !ok {
		l.RepoWarn(apperrors.NotFoundError, map[string]any{"id": id})
		return nil, apperrors.NotFoundError
	}
	return job, nil
}

func (r *inMemoryLLMStorage) Update(ctx context.Context, job *models.LLMJob) error {
	l := logger.FromContext(ctx)

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.jobs[job.ID]
	if !ok {
		l.RepoWarn(apperrors.NotFoundError, map[string]any{"id": job.ID})
		return apperrors.NotFoundError
	}

	job.UpdatedAt = time.Now()
	stored.Status = job.Status
	stored.Result = job.Result
	stored.UpdatedAt = job.UpdatedAt

	return nil
}
