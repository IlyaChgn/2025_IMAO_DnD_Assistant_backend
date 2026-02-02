package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// --- fakes ---

type fakeLLMStorage struct {
	jobs      map[string]*models.LLMJob
	createErr error
	getErr    error
	updateErr error
}

func newFakeLLMStorage() *fakeLLMStorage {
	return &fakeLLMStorage{jobs: make(map[string]*models.LLMJob)}
}

func (f *fakeLLMStorage) Create(_ context.Context, job *models.LLMJob) error {
	if f.createErr != nil {
		return f.createErr
	}
	cp := *job
	f.jobs[job.ID] = &cp
	return nil
}

func (f *fakeLLMStorage) Get(_ context.Context, id string) (*models.LLMJob, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	j, ok := f.jobs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *j
	return &cp, nil
}

func (f *fakeLLMStorage) Update(_ context.Context, job *models.LLMJob) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	cp := *job
	f.jobs[job.ID] = &cp
	return nil
}

type fakeGeminiAPI struct {
	descResult  map[string]interface{}
	descErr     error
	imageResult map[string]interface{}
	imageErr    error
}

func (f *fakeGeminiAPI) GenerateFromDescription(_ context.Context, _ string) (map[string]interface{}, error) {
	return f.descResult, f.descErr
}

func (f *fakeGeminiAPI) GenerateFromImage(_ context.Context, _ []byte) (map[string]interface{}, error) {
	return f.imageResult, f.imageErr
}

type fakeCreatureProcessor struct {
	result *models.Creature
	err    error
}

func (f *fakeCreatureProcessor) ValidateAndProcessGeneratedCreature(_ context.Context,
	_ *models.Creature) (*models.Creature, error) {
	return f.result, f.err
}

type syncRunner struct{}

func (s *syncRunner) Go(fn func()) { fn() }

type fixedIDGen struct {
	id string
}

func (f *fixedIDGen) NewID() string { return f.id }

// --- helpers ---

func validCreatureMap() map[string]interface{} {
	return map[string]interface{}{
		"name": map[string]interface{}{
			"rus": "Гоблин",
			"eng": "Goblin",
		},
	}
}

// --- tests ---

func TestSubmitText(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("storage failure")

	tests := []struct {
		name      string
		desc      string
		storage   *fakeLLMStorage
		gemini    *fakeGeminiAPI
		processor *fakeCreatureProcessor
		fixedID   string
		wantID    string
		wantErr   error
		wantState string // final job status after sync process
	}{
		{
			name:    "storage create error is propagated",
			desc:    "a goblin",
			storage: &fakeLLMStorage{jobs: make(map[string]*models.LLMJob), createErr: storageErr},
			gemini:  &fakeGeminiAPI{},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID: "test-id-1",
			wantErr: storageErr,
		},
		{
			name:    "happy path: text submission processes to done",
			desc:    "a goblin",
			storage: newFakeLLMStorage(),
			gemini: &fakeGeminiAPI{
				descResult: validCreatureMap(),
			},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID:   "test-id-2",
			wantID:    "test-id-2",
			wantState: "done",
		},
		{
			name:    "gemini error sets status to error",
			desc:    "a goblin",
			storage: newFakeLLMStorage(),
			gemini: &fakeGeminiAPI{
				descErr: errors.New("gemini down"),
			},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID:   "test-id-3",
			wantID:    "test-id-3",
			wantState: "error",
		},
		{
			name:    "processor error sets status to error",
			desc:    "a goblin",
			storage: newFakeLLMStorage(),
			gemini: &fakeGeminiAPI{
				descResult: validCreatureMap(),
			},
			processor: &fakeCreatureProcessor{
				err: errors.New("processor failure"),
			},
			fixedID:   "test-id-4",
			wantID:    "test-id-4",
			wantState: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewLLMUsecase(tt.storage, tt.gemini, tt.processor,
				&syncRunner{}, &fixedIDGen{id: tt.fixedID})

			id, err := uc.SubmitText(context.Background(), tt.desc)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Empty(t, id)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, id)

			// With sync runner, process() has already completed
			job, getErr := tt.storage.Get(context.Background(), id)
			assert.NoError(t, getErr)
			assert.Equal(t, tt.wantState, job.Status)
		})
	}
}

func TestSubmitImage(t *testing.T) {
	t.Parallel()

	storageErr := errors.New("storage failure")

	tests := []struct {
		name      string
		image     []byte
		storage   *fakeLLMStorage
		gemini    *fakeGeminiAPI
		processor *fakeCreatureProcessor
		fixedID   string
		wantID    string
		wantErr   error
		wantState string
	}{
		{
			name:    "storage create error is propagated",
			image:   []byte("img-data"),
			storage: &fakeLLMStorage{jobs: make(map[string]*models.LLMJob), createErr: storageErr},
			gemini:  &fakeGeminiAPI{},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID: "img-id-1",
			wantErr: storageErr,
		},
		{
			name:    "happy path: image submission processes to done",
			image:   []byte("img-data"),
			storage: newFakeLLMStorage(),
			gemini: &fakeGeminiAPI{
				imageResult: validCreatureMap(),
			},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID:   "img-id-2",
			wantID:    "img-id-2",
			wantState: "done",
		},
		{
			name:    "gemini image error sets status to error",
			image:   []byte("img-data"),
			storage: newFakeLLMStorage(),
			gemini: &fakeGeminiAPI{
				imageErr: errors.New("gemini image error"),
			},
			processor: &fakeCreatureProcessor{
				result: &models.Creature{},
			},
			fixedID:   "img-id-3",
			wantID:    "img-id-3",
			wantState: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewLLMUsecase(tt.storage, tt.gemini, tt.processor,
				&syncRunner{}, &fixedIDGen{id: tt.fixedID})

			id, err := uc.SubmitImage(context.Background(), tt.image)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Empty(t, id)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantID, id)

			job, getErr := tt.storage.Get(context.Background(), id)
			assert.NoError(t, getErr)
			assert.Equal(t, tt.wantState, job.Status)
		})
	}
}

func TestGetJob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		storage *fakeLLMStorage
		jobID   string
		wantErr bool
	}{
		{
			name: "existing job is returned",
			storage: func() *fakeLLMStorage {
				s := newFakeLLMStorage()
				s.jobs["existing-id"] = &models.LLMJob{ID: "existing-id", Status: "done"}
				return s
			}(),
			jobID: "existing-id",
		},
		{
			name:    "missing job returns error",
			storage: newFakeLLMStorage(),
			jobID:   "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewLLMUsecase(tt.storage, &fakeGeminiAPI{}, &fakeCreatureProcessor{},
				&syncRunner{}, &fixedIDGen{id: "unused"})

			job, err := uc.GetJob(context.Background(), tt.jobID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, job)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.jobID, job.ID)
			}
		})
	}
}

func TestProcess_DoneJobHasResult(t *testing.T) {
	t.Parallel()

	expected := &models.Creature{}
	storage := newFakeLLMStorage()
	gemini := &fakeGeminiAPI{descResult: validCreatureMap()}
	processor := &fakeCreatureProcessor{result: expected}

	uc := NewLLMUsecase(storage, gemini, processor, &syncRunner{}, &fixedIDGen{id: "result-id"})

	id, err := uc.SubmitText(context.Background(), "a dragon")
	assert.NoError(t, err)

	job, err := uc.GetJob(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "done", job.Status)
	assert.NotNil(t, job.Result)
}

func TestProcess_UpdateErrorDuringStep1_StopsProcessing(t *testing.T) {
	t.Parallel()

	storage := newFakeLLMStorage()
	gemini := &fakeGeminiAPI{descResult: validCreatureMap()}
	processor := &fakeCreatureProcessor{result: &models.Creature{}}

	uc := NewLLMUsecase(storage, gemini, processor, &syncRunner{}, &fixedIDGen{id: "upd-err-id"})

	// First create the job normally (Create succeeds)
	id, err := uc.SubmitText(context.Background(), "a goblin")
	assert.NoError(t, err)

	// Now set update error and re-run process to test step_1 update failure
	storage.updateErr = errors.New("update failure")
	storage2 := newFakeLLMStorage()
	storage2.updateErr = errors.New("update failure")

	desc := "a goblin"
	storage2.jobs["step1-fail"] = &models.LLMJob{
		ID:          "step1-fail",
		Description: &desc,
		Status:      "pending",
	}

	uc2 := NewLLMUsecase(storage2, gemini, processor, &syncRunner{}, &fixedIDGen{id: "step1-fail"})
	uc2.process(context.Background(), "step1-fail")

	job, _ := storage2.Get(context.Background(), "step1-fail")
	// Job status stays "pending" because step_1 update failed and process returned
	assert.Equal(t, "pending", job.Status)

	// Original job (no update error) should be "done"
	_ = id
	origJob, _ := storage.Get(context.Background(), "upd-err-id")
	assert.Equal(t, "done", origJob.Status)
}
