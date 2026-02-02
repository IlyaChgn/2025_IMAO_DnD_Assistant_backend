package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/description/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGenerateDescription(t *testing.T) {
	t.Parallel()

	gatewayErr := errors.New("grpc unavailable")

	tests := []struct {
		name      string
		setup     func(gw *mocks.MockDescriptionGateway)
		wantErr   error
		wantEmpty bool
	}{
		{
			name: "gateway error returns ReceivedDescriptionError",
			setup: func(gw *mocks.MockDescriptionGateway) {
				gw.EXPECT().Describe(gomock.Any(), "char-1", "char-2").Return("", gatewayErr)
			},
			wantErr:   apperrors.ReceivedDescriptionError,
			wantEmpty: true,
		},
		{
			name: "happy path returns description",
			setup: func(gw *mocks.MockDescriptionGateway) {
				gw.EXPECT().Describe(gomock.Any(), "char-1", "char-2").Return("The goblin attacks!", nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mocks.NewMockDescriptionGateway(ctrl)
			tt.setup(gw)

			uc := NewDescriptionUsecase(gw)
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
