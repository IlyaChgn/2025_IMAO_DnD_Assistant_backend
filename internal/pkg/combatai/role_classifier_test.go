package combatai

import (
	"math"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestClassifyRole(t *testing.T) {
	t.Parallel()

	meleeAction := models.StructuredAction{
		ID:       "slam",
		Name:     "Slam",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollMeleeWeapon,
			Bonus: 6,
			Reach: 5,
			Damage: []models.DamageRoll{
				{DiceCount: 2, DiceType: "d8", Bonus: 4, DamageType: "bludgeoning"},
			},
		},
	}

	rangedAction := models.StructuredAction{
		ID:       "shortbow",
		Name:     "Shortbow",
		Category: models.ActionCategoryAction,
		Attack: &models.AttackRollData{
			Type:  models.AttackRollRangedWeapon,
			Bonus: 4,
			Range: &models.RangeData{Normal: 80, Long: 320},
			Damage: []models.DamageRoll{
				{DiceCount: 1, DiceType: "d6", Bonus: 2, DamageType: "piercing"},
			},
		},
	}

	saveAction := models.StructuredAction{
		ID:       "breath",
		Name:     "Fire Breath",
		Category: models.ActionCategoryAction,
		SavingThrow: &models.SavingThrowData{
			Ability:   models.AbilityDEX,
			DC:        15,
			OnFail:    "full damage",
			OnSuccess: "half damage",
			Damage: []models.DamageRoll{
				{DiceCount: 8, DiceType: "d6", DamageType: "fire"},
			},
		},
	}

	conditionAction := models.StructuredAction{
		ID:       "gaze",
		Name:     "Petrifying Gaze",
		Category: models.ActionCategoryAction,
		Effects: []models.ActionEffect{
			{Condition: &models.ConditionEffect{Condition: models.ConditionPetrified, Duration: "1 minute"}},
		},
	}

	tests := []struct {
		name     string
		creature models.Creature
		want     CreatureRole
	}{
		{
			name: "Lich → Caster",
			creature: models.Creature{
				Ability:    models.Ability{Str: 11, Dex: 16, Int: 20},
				ArmorClass: 17,
				Hits:       models.Hits{Average: 135},
				Spellcasting: &models.Spellcasting{
					SpellSaveDC:      20,
					SpellAttackBonus: 12,
					SpellsByLevel: map[int][]models.SpellKnown{
						1: {{Name: "Magic Missile", Level: 1}},
						5: {{Name: "Cone of Cold", Level: 5}},
					},
				},
				StructuredActions: []models.StructuredAction{meleeAction},
				ChallengeRating:   "21",
			},
			want: RoleCaster,
		},
		{
			name: "Skeleton Archer → Ranged",
			creature: models.Creature{
				Ability:           models.Ability{Str: 10, Dex: 14},
				ArmorClass:        13,
				Hits:              models.Hits{Average: 13},
				StructuredActions: []models.StructuredAction{rangedAction},
				ChallengeRating:   "1/4",
			},
			want: RoleRanged,
		},
		{
			name: "Ogre → Brute",
			creature: models.Creature{
				Ability:           models.Ability{Str: 19, Dex: 8, Int: 5},
				ArmorClass:        11,
				Hits:              models.Hits{Average: 59},
				StructuredActions: []models.StructuredAction{meleeAction},
				ChallengeRating:   "2",
				Movement:          models.CreatureMovement{Walk: 40},
			},
			want: RoleBrute,
		},
		{
			name: "Monk-like → Skirmisher",
			creature: models.Creature{
				Ability:           models.Ability{Str: 12, Dex: 18, Int: 10},
				ArmorClass:        15,
				Hits:              models.Hits{Average: 40},
				StructuredActions: []models.StructuredAction{meleeAction},
				ChallengeRating:   "3",
				Movement:          models.CreatureMovement{Walk: 40},
			},
			want: RoleSkirmisher,
		},
		{
			name: "Shield Guardian → Tank (high AC)",
			creature: models.Creature{
				Ability:           models.Ability{Str: 15, Dex: 8, Int: 7},
				ArmorClass:        18,
				Hits:              models.Hits{Average: 142},
				StructuredActions: []models.StructuredAction{meleeAction},
				ChallengeRating:   "7",
				Movement:          models.CreatureMovement{Walk: 30},
			},
			want: RoleTank,
		},
		{
			name: "Beholder-like → Controller",
			creature: models.Creature{
				Ability:           models.Ability{Str: 10, Dex: 14, Int: 17},
				ArmorClass:        17,
				Hits:              models.Hits{Average: 180},
				StructuredActions: []models.StructuredAction{saveAction, conditionAction},
				ChallengeRating:   "13",
				Movement:          models.CreatureMovement{Walk: 0},
			},
			want: RoleController,
		},
		{
			name: "Fallback melee → Brute",
			creature: models.Creature{
				Ability:           models.Ability{Str: 14, Dex: 10, Int: 8},
				ArmorClass:        12,
				Hits:              models.Hits{Average: 15},
				StructuredActions: []models.StructuredAction{meleeAction},
				ChallengeRating:   "1",
				Movement:          models.CreatureMovement{Walk: 30},
			},
			want: RoleBrute,
		},
		{
			name: "Fallback ranged → Ranged",
			creature: models.Creature{
				Ability:           models.Ability{Str: 10, Dex: 12, Int: 8},
				ArmorClass:        12,
				Hits:              models.Hits{Average: 11},
				StructuredActions: []models.StructuredAction{rangedAction},
				ChallengeRating:   "1/4",
				Movement:          models.CreatureMovement{Walk: 30},
			},
			want: RoleRanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyRole(tt.creature)
			if got != tt.want {
				t.Errorf("ClassifyRole() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseCR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  float64
	}{
		{"1/4", 0.25},
		{"1/2", 0.5},
		{"1/8", 0.125},
		{"0", 0.0},
		{"3", 3.0},
		{"21", 21.0},
		{"", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := parseCR(tt.input)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("parseCR(%q) = %f, want %f", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseProfBonus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{"+2", 2},
		{"+6", 6},
		{"", 0},
		{"3", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := parseProfBonus(tt.input)
			if got != tt.want {
				t.Errorf("parseProfBonus(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
