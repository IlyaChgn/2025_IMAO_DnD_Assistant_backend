package converter

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestMapClassName(t *testing.T) {
	tests := []struct {
		input     string
		expected  string
		found     bool
	}{
		{"Следопыт", "ranger", true},
		{"Волшебник", "wizard", true},
		{"Волшебница", "wizard", true},
		{"Волшебница ШВ", "wizard", true}, // first word match
		{"Плут", "rogue", true},
		{"Паладин", "paladin", true},
		{"Жрец", "cleric", true},
		{"Литера", "Литера", false},        // unknown class
		{"  Бард  ", "bard", true},         // with whitespace
		{"", "", false},                     // empty
		{"Особый", "Особый", false},         // homebrew
	}

	for _, tt := range tests {
		result, found := MapClassName(tt.input)
		if result != tt.expected || found != tt.found {
			t.Errorf("MapClassName(%q) = (%q, %v), want (%q, %v)",
				tt.input, result, found, tt.expected, tt.found)
		}
	}
}

func TestMapAbilityCode(t *testing.T) {
	tests := []struct {
		input    string
		expected models.AbilityType
		found    bool
	}{
		{"str", models.AbilitySTR, true},
		{"dex", models.AbilityDEX, true},
		{"con", models.AbilityCON, true},
		{"int", models.AbilityINT, true},
		{"wis", models.AbilityWIS, true},
		{"cha", models.AbilityCHA, true},
		{"STR", models.AbilitySTR, true},  // case insensitive
		{"  int  ", models.AbilityINT, true}, // with whitespace
		{"xyz", "", false},
	}

	for _, tt := range tests {
		result, found := MapAbilityCode(tt.input)
		if result != tt.expected || found != tt.found {
			t.Errorf("MapAbilityCode(%q) = (%q, %v), want (%q, %v)",
				tt.input, result, found, tt.expected, tt.found)
		}
	}
}

func TestMapAbilityNameRu(t *testing.T) {
	tests := []struct {
		input    string
		expected models.AbilityType
		found    bool
	}{
		{"Мудрость", models.AbilityWIS, true},
		{"Интеллект", models.AbilityINT, true},
		{"Харизма", models.AbilityCHA, true},
		{"Сила", models.AbilitySTR, true},
		{"мудрость", models.AbilityWIS, true}, // lowercase
		{"Unknown", "", false},
	}

	for _, tt := range tests {
		result, found := MapAbilityNameRu(tt.input)
		if result != tt.expected || found != tt.found {
			t.Errorf("MapAbilityNameRu(%q) = (%q, %v), want (%q, %v)",
				tt.input, result, found, tt.expected, tt.found)
		}
	}
}

func TestParseDamageString(t *testing.T) {
	tests := []struct {
		input      string
		wantDice   string
		wantType   string
	}{
		{"1к10 рубящий", "1d10", "slashing"},
		{"1к6 колющий", "1d6", "piercing"},
		{"2к6 дробящий", "2d6", "bludgeoning"},
		{"3к10", "3d10", ""},
		{"4к4+10", "4d4", ""},
		{"", "", ""},
		{"1к8 огнём", "1d8", "fire"},
	}

	for _, tt := range tests {
		dice, dmgType := ParseDamageString(tt.input)
		if dice != tt.wantDice || dmgType != tt.wantType {
			t.Errorf("ParseDamageString(%q) = (%q, %q), want (%q, %q)",
				tt.input, dice, dmgType, tt.wantDice, tt.wantType)
		}
	}
}

func TestParseHitDie(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1к10", "d10"},
		{"1к8", "d8"},
		{"1к12", "d12"},
		{"1к6", "d6"},
		{"к10", "d10"},
	}

	for _, tt := range tests {
		result := ParseHitDie(tt.input)
		if result != tt.expected {
			t.Errorf("ParseHitDie(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMapSkillName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		found    bool
	}{
		{"Акробатика", "acrobatics", true},
		{"Внимательность", "perception", true},
		{"Unknown", "", false},
	}

	for _, tt := range tests {
		result, found := MapSkillName(tt.input)
		if result != tt.expected || found != tt.found {
			t.Errorf("MapSkillName(%q) = (%q, %v), want (%q, %v)",
				tt.input, result, found, tt.expected, tt.found)
		}
	}
}
