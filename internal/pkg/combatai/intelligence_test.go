package combatai

import (
	"math"
	"testing"
)

func TestComputeIntelligence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		intScore      int
		difficultyMod float64
		want          float64
		tolerance     float64
	}{
		{
			name:      "zombie INT 3",
			intScore:  3,
			want:      (3.0 - 1.0) / 19.0, // ≈ 0.105
			tolerance: 0.01,
		},
		{
			name:      "lich INT 20",
			intScore:  20,
			want:      1.0,
			tolerance: 0.001,
		},
		{
			name:          "goblin INT 10 with negative mod",
			intScore:      10,
			difficultyMod: -0.3,
			want:          (10.0-1.0)/19.0 - 0.3, // ≈ 0.174
			tolerance:     0.01,
		},
		{
			name:     "minimum INT 1 clamps to 0.05",
			intScore: 1,
			want:     0.05,
		},
		{
			name:          "INT 1 with large negative mod still clamps to 0.05",
			intScore:      1,
			difficultyMod: -0.5,
			want:          0.05,
		},
		{
			name:          "INT 20 with positive mod clamps to 1.0",
			intScore:      20,
			difficultyMod: 0.5,
			want:          1.0,
		},
		{
			name:     "ogre INT 5",
			intScore: 5,
			want:     (5.0 - 1.0) / 19.0, // ≈ 0.211
		},
		{
			name:     "dragon INT 16",
			intScore: 16,
			want:     (16.0 - 1.0) / 19.0, // ≈ 0.789
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ComputeIntelligence(tt.intScore, tt.difficultyMod)

			tol := tt.tolerance
			if tol == 0 {
				tol = 0.001
			}

			if math.Abs(got-tt.want) > tol {
				t.Errorf("ComputeIntelligence(%d, %.2f) = %.4f, want %.4f (±%.3f)",
					tt.intScore, tt.difficultyMod, got, tt.want, tol)
			}

			// Always within bounds.
			if got < 0.05 || got > 1.0 {
				t.Errorf("ComputeIntelligence(%d, %.2f) = %.4f, out of [0.05, 1.0]",
					tt.intScore, tt.difficultyMod, got)
			}
		})
	}
}
