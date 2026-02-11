package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertLSS_RealSample(t *testing.T) {
	// Use a real LSS sample if available
	sampleDir := filepath.Join("..", "..", "..", "..", "long_story_short_jsons")
	pattern := filepath.Join(sampleDir, "*Long Story Short*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		// Also try the Windows-style path
		pattern = filepath.Join(sampleDir, "*Long_Story_Short*.json")
		matches, err = filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			t.Skip("No LSS sample files found, skipping integration test")
		}
	}

	for _, path := range matches {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			rawJSON, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}

			char, report, err := ConvertLSS(rawJSON, "test-user-123")
			if err != nil {
				t.Fatalf("ConvertLSS failed: %v", err)
			}

			if char == nil {
				t.Fatal("char is nil")
			}
			if report == nil {
				t.Fatal("report is nil")
			}

			// Basic validation
			if char.Name == "" {
				t.Error("character name is empty")
			}
			if char.UserID != "test-user-123" {
				t.Errorf("userID = %q, want 'test-user-123'", char.UserID)
			}
			if char.Version != 1 {
				t.Errorf("version = %d, want 1", char.Version)
			}
			if char.CreatedAt == "" {
				t.Error("createdAt is empty")
			}
			if char.ImportSource == nil {
				t.Error("importSource is nil")
			}

			// Report should have positive fields
			totalFields := report.FieldsCopied + report.FieldsParsed
			if totalFields == 0 {
				t.Error("no fields were copied or parsed")
			}

			t.Logf("Converted: %s (copied: %d, parsed: %d, skipped: %d, warnings: %d)",
				char.Name, report.FieldsCopied, report.FieldsParsed,
				report.FieldsSkipped, len(report.Warnings))
			for _, w := range report.Warnings {
				t.Logf("  [%s] %s: %s", w.Level, w.Field, w.Message)
			}
		})
	}
}

func TestConvertLSS_MinimalValid(t *testing.T) {
	// Minimal valid LSS JSON
	rawJSON := []byte(`{
		"data": "{\"name\":{\"value\":\"Test Hero\"},\"info\":{\"charClass\":{\"value\":\"Воин\"},\"level\":{\"value\":5},\"race\":{\"value\":\"Эльф\"},\"background\":{\"value\":\"Солдат\"},\"alignment\":{\"value\":\"Хаотично-нейтральный\"},\"experience\":{\"value\":\"6500\"}},\"stats\":{\"str\":{\"score\":16,\"modifier\":3},\"dex\":{\"score\":14,\"modifier\":2},\"con\":{\"score\":12,\"modifier\":1},\"int\":{\"score\":10,\"modifier\":0},\"wis\":{\"score\":8,\"modifier\":-1},\"cha\":{\"score\":13,\"modifier\":1}},\"saves\":{\"str\":{\"isProf\":true},\"con\":{\"isProf\":true}},\"skills\":{\"athletics\":{\"name\":\"athletics\",\"isProf\":1},\"perception\":{\"name\":\"perception\",\"isProf\":0}},\"vitality\":{\"hp-max\":{\"value\":44},\"ac\":{\"value\":\"18\"},\"speed\":{\"value\":30},\"hit-die\":{\"value\":\"1к10\"}},\"coins\":{\"gp\":{\"value\":50},\"sp\":{\"value\":0},\"cp\":{\"value\":0},\"pp\":{\"value\":0},\"ep\":{\"value\":0}},\"text\":{\"personality\":{\"value\":{\"data\":\"Brave and bold.\"}},\"ideals\":{\"value\":{\"data\":\"Honor above all.\"}},\"bonds\":{\"value\":{\"data\":\"My regiment.\"}},\"flaws\":{\"value\":{\"data\":\"Too stubborn.\"}}},\"weaponsList\":[{\"name\":{\"value\":\"Длинный меч\"},\"mod\":{\"value\":\"5\"},\"dmg\":{\"value\":\"1к8 рубящий\"}}],\"subInfo\":{\"age\":{\"value\":\"25\"},\"height\":{\"value\":\"6.0\"},\"weight\":{\"value\":\"180\"},\"eyes\":{\"value\":\"Green\"},\"skin\":{\"value\":\"Fair\"},\"hair\":{\"value\":\"Black\"}}}",
		"edition": "2014",
		"jsonType": "character",
		"version": "2"
	}`)

	char, report, err := ConvertLSS(rawJSON, "user-456")
	if err != nil {
		t.Fatalf("ConvertLSS failed: %v", err)
	}

	// Verify identity
	if char.Name != "Test Hero" {
		t.Errorf("name = %q, want 'Test Hero'", char.Name)
	}
	if char.Race != "Эльф" {
		t.Errorf("race = %q, want 'Эльф'", char.Race)
	}
	if len(char.Classes) != 1 || char.Classes[0].ClassName != "fighter" {
		t.Errorf("classes = %+v, want [{className: fighter, level: 5}]", char.Classes)
	}
	if char.Classes[0].Level != 5 {
		t.Errorf("level = %d, want 5", char.Classes[0].Level)
	}
	if char.Edition != "2014" {
		t.Errorf("edition = %q, want '2014'", char.Edition)
	}

	// Verify ability scores
	if char.AbilityScores.Str != 16 {
		t.Errorf("STR = %d, want 16", char.AbilityScores.Str)
	}
	if char.AbilityScores.Dex != 14 {
		t.Errorf("DEX = %d, want 14", char.AbilityScores.Dex)
	}

	// Verify saves
	if len(char.Proficiencies.SavingThrows) != 2 {
		t.Errorf("saving throws count = %d, want 2", len(char.Proficiencies.SavingThrows))
	}

	// Verify vitality
	if char.HitPoints.MaxOverride == nil || *char.HitPoints.MaxOverride != 44 {
		t.Errorf("maxHP = %v, want 44", char.HitPoints.MaxOverride)
	}
	if char.ArmorClassOverride == nil || *char.ArmorClassOverride != 18 {
		t.Errorf("AC = %v, want 18", char.ArmorClassOverride)
	}
	if char.BaseSpeed != 30 {
		t.Errorf("speed = %d, want 30", char.BaseSpeed)
	}
	if char.HitPoints.HitDie != "d10" {
		t.Errorf("hitDie = %q, want 'd10'", char.HitPoints.HitDie)
	}

	// Verify weapons
	if len(char.Weapons) != 1 {
		t.Errorf("weapons count = %d, want 1", len(char.Weapons))
	} else {
		w := char.Weapons[0]
		if w.Name != "Длинный меч" {
			t.Errorf("weapon name = %q, want 'Длинный меч'", w.Name)
		}
		if w.DamageDice != "1d8" {
			t.Errorf("weapon dice = %q, want '1d8'", w.DamageDice)
		}
		if w.DamageType != "slashing" {
			t.Errorf("weapon type = %q, want 'slashing'", w.DamageType)
		}
	}

	// Verify coins
	if char.Coins.Gp != 50 {
		t.Errorf("gold = %d, want 50", char.Coins.Gp)
	}

	// Verify text
	if char.PersonalityTraits != "Brave and bold." {
		t.Errorf("personality = %q, want 'Brave and bold.'", char.PersonalityTraits)
	}
	if char.Ideals != "Honor above all." {
		t.Errorf("ideals = %q, want 'Honor above all.'", char.Ideals)
	}

	// Verify appearance
	if char.Appearance.Age != "25" {
		t.Errorf("age = %q, want '25'", char.Appearance.Age)
	}

	// Verify report
	if !report.Success {
		t.Error("report.Success is false")
	}

	t.Logf("Report: copied=%d, parsed=%d, skipped=%d, warnings=%d",
		report.FieldsCopied, report.FieldsParsed, report.FieldsSkipped, len(report.Warnings))
}

func TestConvertLSS_IncorrectModifier(t *testing.T) {
	// Test that incorrect LSS modifiers generate a warning
	rawJSON := []byte(`{
		"data": "{\"name\":{\"value\":\"Bad Mod Character\"},\"info\":{\"charClass\":{\"value\":\"Воин\"},\"level\":{\"value\":1},\"race\":{\"value\":\"Human\"}},\"stats\":{\"int\":{\"score\":10,\"modifier\":-5}}}",
		"jsonType": "character",
		"version": "2"
	}`)

	_, report, err := ConvertLSS(rawJSON, "user")
	if err != nil {
		t.Fatalf("ConvertLSS failed: %v", err)
	}

	// Should have a warning about INT modifier mismatch (LSS has -5, should be 0)
	found := false
	for _, w := range report.Warnings {
		if w.Field == "stats.int.modifier" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about incorrect INT modifier, got none")
	}
}

func TestConvertLSS_InvalidJSON(t *testing.T) {
	_, _, err := ConvertLSS([]byte("not json"), "user")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestConvertLSS_MissingDataField(t *testing.T) {
	_, _, err := ConvertLSS([]byte(`{"version": "2"}`), "user")
	if err == nil {
		t.Error("expected error for missing data field")
	}
}
