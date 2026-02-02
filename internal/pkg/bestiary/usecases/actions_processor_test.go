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

func TestProcessActions(t *testing.T) {
	t.Parallel()

	gatewayErr := errors.New("grpc unavailable")

	tests := []struct {
		name    string
		actions []models.Action
		setup   func(gw *mocks.MockActionProcessorGateway)
		wantErr error
		wantLen int
	}{
		{
			name:    "gateway error returns ReceivedActionProcessingError",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).Return(nil, gatewayErr)
			},
			wantErr: apperrors.ReceivedActionProcessingError,
		},
		{
			name:    "missing parsed_actions key returns ParsedActionsErr",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{"other_key": "value"}, nil)
			},
			wantErr: apperrors.ParsedActionsErr,
		},
		{
			name:    "empty map returns ParsedActionsErr",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{}, nil)
			},
			wantErr: apperrors.ParsedActionsErr,
		},
		{
			name:    "parsed_actions is not a list returns unmarshal error",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{"parsed_actions": "not-a-list"}, nil)
			},
		},
		{
			name:    "parsed_actions with wrong field types returns unmarshal error",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{
						"parsed_actions": []interface{}{
							map[string]interface{}{
								"name":         123,
								"attack_bonus": true,
							},
						},
					}, nil)
			},
		},
		{
			name:    "happy path: single attack parsed",
			actions: []models.Action{{Name: "Bite", Value: "melee attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{
						"parsed_actions": []interface{}{
							map[string]interface{}{
								"name":        "Bite",
								"type":        "melee",
								"attackBonus": "+5",
								"reach":       "5 ft.",
								"target":      "one target",
							},
						},
					}, nil)
			},
			wantLen: 1,
		},
		{
			name:    "happy path: multiple attacks parsed",
			actions: []models.Action{{Name: "Bite", Value: "a"}, {Name: "Claw", Value: "b"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{
						"parsed_actions": []interface{}{
							map[string]interface{}{"name": "Bite", "type": "melee"},
							map[string]interface{}{"name": "Claw", "type": "melee"},
						},
					}, nil)
			},
			wantLen: 2,
		},
		{
			name:    "happy path: empty parsed_actions list returns empty slice",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{"parsed_actions": []interface{}{}}, nil)
			},
			wantLen: 0,
		},
		{
			name:    "nil actions still calls gateway",
			actions: nil,
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Nil()).
					Return(map[string]interface{}{"parsed_actions": []interface{}{}}, nil)
			},
			wantLen: 0,
		},
		{
			name:    "parsed_actions with nested damage object",
			actions: []models.Action{{Name: "Bite", Value: "attack"}},
			setup: func(gw *mocks.MockActionProcessorGateway) {
				gw.EXPECT().ProcessActions(gomock.Any(), gomock.Any()).
					Return(map[string]interface{}{
						"parsed_actions": []interface{}{
							map[string]interface{}{
								"name": "Bite",
								"type": "melee",
								"damage": map[string]interface{}{
									"dice":  "d8",
									"count": float64(2),
									"type":  "piercing",
									"bonus": float64(4),
								},
							},
						},
					}, nil)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			gw := mocks.NewMockActionProcessorGateway(ctrl)
			tt.setup(gw)

			uc := NewActionProcessorUsecase(gw)
			result, err := uc.ProcessActions(context.Background(), tt.actions)

			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
				assert.Nil(t, result)
				return
			}

			if tt.name == "parsed_actions is not a list returns unmarshal error" ||
				tt.name == "parsed_actions with wrong field types returns unmarshal error" {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, result, tt.wantLen)

			if tt.wantLen > 0 && tt.name == "happy path: single attack parsed" {
				assert.Equal(t, "Bite", result[0].Name)
				assert.Equal(t, "melee", result[0].Type)
				assert.Equal(t, "+5", result[0].AttackBonus)
			}

			if tt.wantLen > 0 && tt.name == "parsed_actions with nested damage object" {
				assert.NotNil(t, result[0].Damage)
				assert.Equal(t, "d8", result[0].Damage.Dice)
				assert.Equal(t, 2, result[0].Damage.Count)
				assert.Equal(t, "piercing", result[0].Damage.Type)
				assert.Equal(t, 4, result[0].Damage.Bonus)
			}
		})
	}
}
