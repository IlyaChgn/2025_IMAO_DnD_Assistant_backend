package combatai

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestGetCurrentHP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    *models.ParticipantFull
		want int
	}{
		{
			name: "NPC alive",
			p: &models.ParticipantFull{
				RuntimeState: models.CreatureRuntimeState{
					CurrentHP: 45,
					MaxHP:     60,
				},
			},
			want: 45,
		},
		{
			name: "NPC dead",
			p: &models.ParticipantFull{
				RuntimeState: models.CreatureRuntimeState{
					CurrentHP: 0,
					MaxHP:     60,
				},
			},
			want: 0,
		},
		{
			name: "PC alive",
			p: &models.ParticipantFull{
				CharacterRuntime: &models.CharacterRuntime{
					CurrentHP: 32,
				},
			},
			want: 32,
		},
		{
			name: "PC dead",
			p: &models.ParticipantFull{
				CharacterRuntime: &models.CharacterRuntime{
					CurrentHP: 0,
				},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GetCurrentHP(tt.p)
			if got != tt.want {
				t.Errorf("GetCurrentHP() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIsAlive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    *models.ParticipantFull
		want bool
	}{
		{
			name: "NPC alive",
			p: &models.ParticipantFull{
				RuntimeState: models.CreatureRuntimeState{CurrentHP: 10},
			},
			want: true,
		},
		{
			name: "NPC dead",
			p: &models.ParticipantFull{
				RuntimeState: models.CreatureRuntimeState{CurrentHP: 0},
			},
			want: false,
		},
		{
			name: "PC alive",
			p: &models.ParticipantFull{
				CharacterRuntime: &models.CharacterRuntime{CurrentHP: 5},
			},
			want: true,
		},
		{
			name: "PC dead",
			p: &models.ParticipantFull{
				CharacterRuntime: &models.CharacterRuntime{CurrentHP: 0},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsAlive(tt.p)
			if got != tt.want {
				t.Errorf("IsAlive() = %v, want %v", got, tt.want)
			}
		})
	}
}
