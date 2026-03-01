package combatai

import (
	"math"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestComputeExpectedDamage_AttackRoll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action models.StructuredAction
		target CombatantStats
		want   float64
		tol    float64
	}{
		{
			name: "melee +5 vs AC 15, 2d6+3",
			action: models.StructuredAction{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 2, DiceType: "d6", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
			target: CombatantStats{AC: 15},
			// hit = (21-(15-5))/20 = 11/20 = 0.55
			// avg = 2*(6+1)/2 + 3 = 7 + 3 = 10
			// expected = 0.55*10 + 0.05*10 = 5.5 + 0.5 = 6.0
			want: 6.0,
			tol:  0.01,
		},
		{
			name: "high AC clamps hit to 0.05",
			action: models.StructuredAction{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "slashing"},
					},
				},
			},
			target: CombatantStats{AC: 30},
			// hit = (21-(30-5))/20 = -4/20 → clamped to 0.05
			// avg = 1*(8+1)/2 + 3 = 4.5 + 3 = 7.5
			// expected = 0.05*7.5 + 0.05*7.5 = 0.375 + 0.375 = 0.75
			want: 0.75,
			tol:  0.01,
		},
		{
			name: "low AC clamps hit to 0.95",
			action: models.StructuredAction{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 10,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d6", Bonus: 5, DamageType: "slashing"},
					},
				},
			},
			target: CombatantStats{AC: 5},
			// hit = (21-(5-10))/20 = 26/20 → clamped to 0.95
			// avg = 1*(6+1)/2 + 5 = 3.5 + 5 = 8.5
			// expected = 0.95*8.5 + 0.05*8.5 = 8.075 + 0.425 = 8.5
			want: 8.5,
			tol:  0.01,
		},
		{
			name: "multiple damage rolls, 1d8+3 piercing + 2d6 fire",
			action: models.StructuredAction{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 7,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "piercing"},
						{DiceCount: 2, DiceType: "d6", Bonus: 0, DamageType: "fire"},
					},
				},
			},
			target: CombatantStats{AC: 15},
			// hit = (21-(15-7))/20 = 13/20 = 0.65
			// avg = (1*(8+1)/2 + 3) + (2*(6+1)/2 + 0) = 7.5 + 7.0 = 14.5
			// expected = 0.65*14.5 + 0.05*14.5 = 9.425 + 0.725 = 10.15
			want: 10.15,
			tol:  0.01,
		},
		{
			name: "conditional damage skipped",
			action: models.StructuredAction{
				Attack: &models.AttackRollData{
					Type:  models.AttackRollMeleeWeapon,
					Bonus: 5,
					Damage: []models.DamageRoll{
						{DiceCount: 1, DiceType: "d8", Bonus: 3, DamageType: "piercing"},
						{DiceCount: 2, DiceType: "d6", Bonus: 0, DamageType: "fire", Condition: "undead target"},
					},
				},
			},
			target: CombatantStats{AC: 15},
			// hit = 0.55, avg = 4.5+3 = 7.5 (conditional 2d6 skipped)
			// expected = 0.55*7.5 + 0.05*7.5 = 4.125 + 0.375 = 4.5
			want: 4.5,
			tol:  0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ComputeExpectedDamage(tt.action, tt.target)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("ComputeExpectedDamage() = %.4f, want %.4f (±%.2f)", got, tt.want, tt.tol)
			}
		})
	}
}

func TestComputeExpectedDamage_SavingThrow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action models.StructuredAction
		target CombatantStats
		want   float64
		tol    float64
	}{
		{
			name: "DC 15 vs +2 save, half on success",
			action: models.StructuredAction{
				SavingThrow: &models.SavingThrowData{
					Ability:   models.AbilityDEX,
					DC:        15,
					OnFail:    "full damage",
					OnSuccess: "half damage",
					Damage: []models.DamageRoll{
						{DiceCount: 8, DiceType: "d6", Bonus: 0, DamageType: "fire"},
					},
				},
			},
			target: CombatantStats{SaveBonuses: map[string]int{"DEX": 2}},
			// fail = (15-2-1)/20 = 12/20 = 0.60
			// avg = 8*(6+1)/2 = 28
			// expected = 0.60*28 + 0.40*14 = 16.8 + 5.6 = 22.4
			want: 22.4,
			tol:  0.01,
		},
		{
			name: "DC 15 vs +2 save, no effect on success",
			action: models.StructuredAction{
				SavingThrow: &models.SavingThrowData{
					Ability:   models.AbilityCON,
					DC:        15,
					OnFail:    "full damage",
					OnSuccess: "no effect",
					Damage: []models.DamageRoll{
						{DiceCount: 4, DiceType: "d8", Bonus: 0, DamageType: "necrotic"},
					},
				},
			},
			target: CombatantStats{SaveBonuses: map[string]int{"CON": 2}},
			// fail = (15-2-1)/20 = 0.60
			// avg = 4*(8+1)/2 = 18
			// expected = 0.60*18 = 10.8
			want: 10.8,
			tol:  0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ComputeExpectedDamage(tt.action, tt.target)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("ComputeExpectedDamage() = %.4f, want %.4f (±%.2f)", got, tt.want, tt.tol)
			}
		})
	}
}

func TestComputeExpectedDamage_NoAttackOrSave(t *testing.T) {
	t.Parallel()

	got := ComputeExpectedDamage(models.StructuredAction{}, CombatantStats{AC: 15})
	if got != 0 {
		t.Errorf("ComputeExpectedDamage() = %f, want 0", got)
	}
}

func TestEstimateSpellDamage(t *testing.T) {
	t.Parallel()

	qr := &models.SpellQuickRef{Range: "60 feet"}
	selfQR := &models.SpellQuickRef{Range: "Self"}

	tests := []struct {
		name         string
		spell        models.SpellKnown
		spellcasting models.Spellcasting
		target       CombatantStats
		want         float64
		tol          float64
	}{
		{
			name:         "QuickRef nil → 0",
			spell:        models.SpellKnown{Name: "Unknown", Level: 3},
			spellcasting: models.Spellcasting{SpellAttackBonus: 7},
			target:       CombatantStats{AC: 15},
			want:         0,
			tol:          0.001,
		},
		{
			name:         "Self range → 0",
			spell:        models.SpellKnown{Name: "Shield", Level: 1, QuickRef: selfQR},
			spellcasting: models.Spellcasting{SpellAttackBonus: 7},
			target:       CombatantStats{AC: 15},
			want:         0,
			tol:          0.001,
		},
		{
			name:         "Cantrip CasterLevel 1",
			spell:        models.SpellKnown{Name: "Fire Bolt", Level: 0, QuickRef: qr},
			spellcasting: models.Spellcasting{CasterLevel: 1},
			target:       CombatantStats{AC: 15},
			want:         5.5,
			tol:          0.001,
		},
		{
			name:         "Cantrip CasterLevel 5",
			spell:        models.SpellKnown{Name: "Fire Bolt", Level: 0, QuickRef: qr},
			spellcasting: models.Spellcasting{CasterLevel: 5},
			target:       CombatantStats{AC: 15},
			want:         11.0,
			tol:          0.001,
		},
		{
			name:         "Cantrip CasterLevel 11",
			spell:        models.SpellKnown{Name: "Fire Bolt", Level: 0, QuickRef: qr},
			spellcasting: models.Spellcasting{CasterLevel: 11},
			target:       CombatantStats{AC: 15},
			want:         16.5,
			tol:          0.001,
		},
		{
			name:         "Cantrip CasterLevel 17",
			spell:        models.SpellKnown{Name: "Fire Bolt", Level: 0, QuickRef: qr},
			spellcasting: models.Spellcasting{CasterLevel: 17},
			target:       CombatantStats{AC: 15},
			want:         22.0,
			tol:          0.001,
		},
		{
			name:         "Spell attack level 3, bonus +7 vs AC 15",
			spell:        models.SpellKnown{Name: "Scorching Ray", Level: 3, QuickRef: qr},
			spellcasting: models.Spellcasting{SpellAttackBonus: 7},
			target:       CombatantStats{AC: 15},
			// hit = (21-(15-7))/20 = 13/20 = 0.65
			// dmg = 3*3.5+3 = 13.5
			// expected = 0.65*13.5 = 8.775
			want: 8.775,
			tol:  0.01,
		},
		{
			name:         "Save spell level 3, DC 15 vs DEX +2",
			spell:        models.SpellKnown{Name: "Fireball", Level: 3, QuickRef: qr},
			spellcasting: models.Spellcasting{SpellSaveDC: 15},
			target:       CombatantStats{AC: 15, SaveBonuses: map[string]int{"DEX": 2}},
			// fail = (15-2-1)/20 = 0.60
			// dmg = 3*4.5 = 13.5
			// expected = 0.60*13.5 + 0.40*6.75 = 8.1 + 2.7 = 10.8
			want: 10.8,
			tol:  0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := EstimateSpellDamage(tt.spell, tt.spellcasting, tt.target)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("EstimateSpellDamage() = %.4f, want %.4f (±%.2f)", got, tt.want, tt.tol)
			}
		})
	}
}

func TestParseDiceMax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{"d6", 6},
		{"d8", 8},
		{"d10", 10},
		{"d12", 12},
		{"d20", 20},
		{"d4", 4},
		{"D6", 6},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := parseDiceMax(tt.input)
			if got != tt.want {
				t.Errorf("parseDiceMax(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
