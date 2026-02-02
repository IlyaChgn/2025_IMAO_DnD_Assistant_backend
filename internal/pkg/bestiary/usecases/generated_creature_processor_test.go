package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/bestiary/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestValidateAndProcessGeneratedCreature(t *testing.T) {
	t.Parallel()

	attacks := []models.AttackLLM{{Name: "Bite"}}
	processorErr := errors.New("grpc error")

	tests := []struct {
		name     string
		creature *models.Creature
		setup    func(proc *mocks.MockActionProcessorUsecases)
		wantErr  error
		wantNil  bool
	}{
		{
			name:     "nil creature returns NilCreatureErr",
			creature: nil,
			setup:    func(_ *mocks.MockActionProcessorUsecases) {},
			wantErr:  apperrors.NilCreatureErr,
			wantNil:  true,
		},
		{
			name: "action processor error does not fail overall",
			creature: &models.Creature{
				Actions: []models.Action{{Name: "Bite", Value: "attack"}},
			},
			setup: func(proc *mocks.MockActionProcessorUsecases) {
				proc.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).Return(nil, processorErr)
			},
		},
		{
			name: "happy path sets LLMParsedAttack",
			creature: &models.Creature{
				Actions: []models.Action{{Name: "Bite", Value: "attack"}},
			},
			setup: func(proc *mocks.MockActionProcessorUsecases) {
				proc.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).Return(attacks, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			proc := mocks.NewMockActionProcessorUsecases(ctrl)
			tt.setup(proc)

			processor := NewGeneratedCreatureProcessor(proc)
			result, err := processor.ValidateAndProcessGeneratedCreature(context.Background(), tt.creature)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.wantNil {
				assert.Nil(t, result)
			} else if tt.wantErr == nil {
				assert.NotNil(t, result)
				if processorErr == nil || tt.name == "happy path sets LLMParsedAttack" {
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
