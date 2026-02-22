package combatai

import (
	"math"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestDistanceFt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    *models.CellsCoordinates
		b    *models.CellsCoordinates
		want int
	}{
		{
			name: "diagonal 3,4 → Chebyshev max(3,4)*5 = 20",
			a:    &models.CellsCoordinates{CellsX: 0, CellsY: 0},
			b:    &models.CellsCoordinates{CellsX: 3, CellsY: 4},
			want: 20,
		},
		{
			name: "same position",
			a:    &models.CellsCoordinates{CellsX: 5, CellsY: 5},
			b:    &models.CellsCoordinates{CellsX: 5, CellsY: 5},
			want: 0,
		},
		{
			name: "horizontal only",
			a:    &models.CellsCoordinates{CellsX: 0, CellsY: 0},
			b:    &models.CellsCoordinates{CellsX: 6, CellsY: 0},
			want: 30,
		},
		{
			name: "vertical only",
			a:    &models.CellsCoordinates{CellsX: 0, CellsY: 0},
			b:    &models.CellsCoordinates{CellsX: 0, CellsY: 3},
			want: 15,
		},
		{
			name: "negative coordinates",
			a:    &models.CellsCoordinates{CellsX: -2, CellsY: -3},
			b:    &models.CellsCoordinates{CellsX: 2, CellsY: 1},
			want: 20, // max(4, 4) * 5
		},
		{
			name: "nil first",
			a:    nil,
			b:    &models.CellsCoordinates{CellsX: 3, CellsY: 4},
			want: math.MaxInt32,
		},
		{
			name: "nil second",
			a:    &models.CellsCoordinates{CellsX: 3, CellsY: 4},
			b:    nil,
			want: math.MaxInt32,
		},
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: math.MaxInt32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DistanceFt(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("DistanceFt(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
