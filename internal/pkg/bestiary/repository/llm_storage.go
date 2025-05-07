package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	bestiaryinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary"
)

var ErrNotFound = errors.New("job not found")

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
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return job, nil
}

func (r *inMemoryLLMStorage) Update(ctx context.Context, job *models.LLMJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, ok := r.jobs[job.ID]
	if !ok {
		return ErrNotFound
	}
	job.UpdatedAt = time.Now()
	stored.Status = job.Status
	stored.Result = job.Result
	stored.UpdatedAt = job.UpdatedAt
	return nil
}
