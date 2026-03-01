package dice

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		count    int
		sides    int
		modifier int
		wantErr  bool
	}{
		{"simple", "2d6", 2, 6, 0, false},
		{"with positive modifier", "1d20+5", 1, 20, 5, false},
		{"with negative modifier", "3d8-2", 3, 8, -2, false},
		{"single die", "1d4", 1, 4, 0, false},
		{"large dice", "10d10+10", 10, 10, 10, false},
		{"invalid format", "abc", 0, 0, 0, true},
		{"missing count", "d6", 0, 0, 0, true},
		{"zero count", "0d6", 0, 0, 0, true},
		{"zero sides", "2d0", 0, 0, 0, true},
		{"empty string", "", 0, 0, 0, true},
		{"bounds exceeded count", "101d6", 0, 0, 0, true},
		{"bounds exceeded sides", "1d1001", 0, 0, 0, true},
		{"max bounds", "100d1000", 100, 1000, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, sides, modifier, err := Parse(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr = %v", tt.expr, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if count != tt.count || sides != tt.sides || modifier != tt.modifier {
				t.Errorf("Parse(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.expr, count, sides, modifier, tt.count, tt.sides, tt.modifier)
			}
		})
	}
}

func TestRollDice(t *testing.T) {
	rolls, total := RollDice(3, 6)
	if len(rolls) != 3 {
		t.Fatalf("RollDice(3, 6) returned %d rolls, want 3", len(rolls))
	}

	sum := 0
	for _, r := range rolls {
		if r < 1 || r > 6 {
			t.Errorf("RollDice(3, 6) roll %d out of range [1, 6]", r)
		}
		sum += r
	}

	if total != sum {
		t.Errorf("RollDice(3, 6) total = %d, want sum of rolls = %d", total, sum)
	}
}

func TestRoll(t *testing.T) {
	result, err := Roll("2d6+3")
	if err != nil {
		t.Fatalf("Roll(\"2d6+3\") error = %v", err)
	}

	if len(result.Rolls) != 2 {
		t.Errorf("Roll(\"2d6+3\") returned %d rolls, want 2", len(result.Rolls))
	}

	if result.Modifier != 3 {
		t.Errorf("Roll(\"2d6+3\") modifier = %d, want 3", result.Modifier)
	}

	diceSum := 0
	for _, r := range result.Rolls {
		diceSum += r
	}

	if result.Total != diceSum+3 {
		t.Errorf("Roll(\"2d6+3\") total = %d, want %d", result.Total, diceSum+3)
	}
}

func TestRollInvalid(t *testing.T) {
	_, err := Roll("invalid")
	if err == nil {
		t.Error("Roll(\"invalid\") expected error, got nil")
	}
}

func TestRollD20Normal(t *testing.T) {
	for range 100 {
		natural, total, rolls := RollD20(5, false, false)
		if len(rolls) != 1 {
			t.Fatalf("RollD20 normal: got %d rolls, want 1", len(rolls))
		}
		if natural < 1 || natural > 20 {
			t.Errorf("RollD20 normal: natural %d out of range", natural)
		}
		if total != natural+5 {
			t.Errorf("RollD20 normal: total = %d, want %d", total, natural+5)
		}
	}
}

func TestRollD20Advantage(t *testing.T) {
	for range 100 {
		natural, total, rolls := RollD20(3, true, false)
		if len(rolls) != 2 {
			t.Fatalf("RollD20 advantage: got %d rolls, want 2", len(rolls))
		}
		expected := max(rolls[0], rolls[1])
		if natural != expected {
			t.Errorf("RollD20 advantage: natural = %d, want max(%d, %d) = %d",
				natural, rolls[0], rolls[1], expected)
		}
		if total != natural+3 {
			t.Errorf("RollD20 advantage: total = %d, want %d", total, natural+3)
		}
	}
}

func TestRollD20Disadvantage(t *testing.T) {
	for range 100 {
		natural, total, rolls := RollD20(2, false, true)
		if len(rolls) != 2 {
			t.Fatalf("RollD20 disadvantage: got %d rolls, want 2", len(rolls))
		}
		expected := min(rolls[0], rolls[1])
		if natural != expected {
			t.Errorf("RollD20 disadvantage: natural = %d, want min(%d, %d) = %d",
				natural, rolls[0], rolls[1], expected)
		}
		if total != natural+2 {
			t.Errorf("RollD20 disadvantage: total = %d, want %d", total, natural+2)
		}
	}
}

func TestRollD20BothCancelOut(t *testing.T) {
	for range 100 {
		_, _, rolls := RollD20(0, true, true)
		if len(rolls) != 1 {
			t.Fatalf("RollD20 both: got %d rolls, want 1 (should cancel out)", len(rolls))
		}
	}
}
