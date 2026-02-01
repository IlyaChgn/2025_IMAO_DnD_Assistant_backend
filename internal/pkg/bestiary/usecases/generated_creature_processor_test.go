package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/stretchr/testify/assert"
)

// --- fake action processor ---

type fakeActionProcessor struct {
	result []models.AttackLLM
	err    error
}

func (f *fakeActionProcessor) ProcessActions(_ context.Context, _ []models.Action) ([]models.AttackLLM, error) {
	return f.result, f.err
}

// --- tests ---

func TestValidateAndProcessGeneratedCreature(t *testing.T) {
	t.Parallel()

	attacks := []models.AttackLLM{{Name: "Bite"}}
	processorErr := errors.New("grpc error")

	tests := []struct {
		name      string
		creature  *models.Creature
		processor *fakeActionProcessor
		wantErr   error
		wantNil   bool
	}{
		{
			name:      "nil creature returns NilCreatureErr",
			creature:  nil,
			processor: &fakeActionProcessor{},
			wantErr:   apperrors.NilCreatureErr,
			wantNil:   true,
		},
		{
			name: "action processor error does not fail overall",
			creature: &models.Creature{
				Actions: []models.Action{{Name: "Bite", Value: "attack"}},
			},
			processor: &fakeActionProcessor{err: processorErr},
		},
		{
			name: "happy path sets LLMParsedAttack",
			creature: &models.Creature{
				Actions: []models.Action{{Name: "Bite", Value: "attack"}},
			},
			processor: &fakeActionProcessor{result: attacks},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			proc := NewGeneratedCreatureProcessor(tt.processor)
			result, err := proc.ValidateAndProcessGeneratedCreature(context.Background(), tt.creature)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if tt.wantErr == nil {
				assert.NotNil(t, result)
				if tt.processor.err == nil {
					assert.Equal(t, attacks, result.LLMParsedAttack)
				}
			}
		})
	}
}

func TestProcessActionValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, output string)
	}{
		{
			name:  "empty string returns empty",
			input: "",
			check: func(t *testing.T, output string) {
				assert.Equal(t, "", output)
			},
		},
		{
			name:  "whitespace only returns as-is",
			input: "   ",
			check: func(t *testing.T, output string) {
				assert.Equal(t, "   ", output)
			},
		},
		{
			name:  "text with colon wraps prefix in em",
			input: "Bite: melee attack",
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "<p><em>Bite:</em>")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := processActionValue(tt.input)
			tt.check(t, result)
		})
	}
}
