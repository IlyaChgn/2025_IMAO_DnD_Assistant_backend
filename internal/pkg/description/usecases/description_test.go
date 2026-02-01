package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	descriptionproto "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/delivery/protobuf"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// --- fake gRPC client ---

type fakeDescriptionClient struct {
	result *descriptionproto.DescriptionResponse
	err    error
}

func (f *fakeDescriptionClient) GenerateDescription(_ context.Context,
	_ *descriptionproto.DescriptionRequest, _ ...grpc.CallOption) (*descriptionproto.DescriptionResponse, error) {
	return f.result, f.err
}

// --- tests ---

func TestGenerateDescription(t *testing.T) {
	t.Parallel()

	grpcErr := errors.New("grpc unavailable")

	tests := []struct {
		name      string
		client    *fakeDescriptionClient
		wantErr   error
		wantEmpty bool
	}{
		{
			name:      "gRPC error returns ReceivedDescriptionError",
			client:    &fakeDescriptionClient{err: grpcErr},
			wantErr:   apperrors.ReceivedDescriptionError,
			wantEmpty: true,
		},
		{
			name: "happy path returns description",
			client: &fakeDescriptionClient{
				result: &descriptionproto.DescriptionResponse{
					BattleDescription: "The goblin attacks!",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uc := NewDescriptionUsecase(tt.client)
			result, err := uc.GenerateDescription(context.Background(),
				models.DescriptionGenerationRequest{
					FirstCharID:  "char-1",
					SecondCharID: "char-2",
				})

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "The goblin attacks!", result.BattleDescription)
			}

			if tt.wantEmpty {
				assert.Empty(t, result.BattleDescription)
			}
		})
	}
}
