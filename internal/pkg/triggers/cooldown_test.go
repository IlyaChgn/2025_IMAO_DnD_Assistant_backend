package triggers

import (
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestParseCooldown(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *ParsedCooldown
		wantErr bool
	}{
		{
			name:  "1/turn",
			input: "1/turn",
			want:  &ParsedCooldown{MaxUses: 1, Period: PeriodTurn},
		},
		{
			name:  "3/short_rest",
			input: "3/short_rest",
			want:  &ParsedCooldown{MaxUses: 3, Period: PeriodShortRest},
		},
		{
			name:  "1/long_rest",
			input: "1/long_rest",
			want:  &ParsedCooldown{MaxUses: 1, Period: PeriodLongRest},
		},
		{
			name:  "2/dawn",
			input: "2/dawn",
			want:  &ParsedCooldown{MaxUses: 2, Period: PeriodDawn},
		},
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:    "malformed input",
			input:   "bad",
			wantErr: true,
		},
		{
			name:    "zero uses invalid",
			input:   "0/turn",
			wantErr: true,
		},
		{
			name:    "negative uses invalid",
			input:   "-1/turn",
			wantErr: true,
		},
		{
			name:    "unknown period",
			input:   "1/combat",
			wantErr: true,
		},
		{
			name:    "non-numeric uses",
			input:   "abc/turn",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCooldown(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result, got nil")
			}
			if got.MaxUses != tt.want.MaxUses || got.Period != tt.want.Period {
				t.Errorf("got {%d, %q}, want {%d, %q}",
					got.MaxUses, got.Period, tt.want.MaxUses, tt.want.Period)
			}
		})
	}
}

func TestBuildCooldownState_FreshCharges(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},
	}
	state := BuildCooldownState(defs, "wpn-1", nil)
	if state != nil {
		t.Errorf("expected nil state for nil charges, got %v", state)
	}

	state = BuildCooldownState(defs, "wpn-1", map[string]int{})
	if state != nil {
		t.Errorf("expected nil state for empty charges, got %v", state)
	}
}

func TestBuildCooldownState_ExhaustedKey(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},
	}
	charges := map[string]int{"wpn-1:0": 1}
	state := BuildCooldownState(defs, "wpn-1", charges)

	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if !state["wpn-1:0"] {
		t.Error("expected key wpn-1:0 to be on cooldown")
	}
}

func TestBuildCooldownState_PartialCharges(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "3/turn"},
	}
	charges := map[string]int{"wpn-1:0": 1}
	state := BuildCooldownState(defs, "wpn-1", charges)

	// 1 < 3 → not on cooldown
	if state != nil && state["wpn-1:0"] {
		t.Error("expected key wpn-1:0 NOT to be on cooldown (1/3 used)")
	}
}

func TestBuildCooldownState_MultipleTriggers(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},       // idx 0
		{Trigger: models.ItemTriggerOnHit, Chance: 1.0},              // idx 1, no cooldown
		{Trigger: models.ItemTriggerOnCritical, Cooldown: "1/turn"},  // idx 2
	}
	charges := map[string]int{
		"wpn-1:0": 1, // exhausted
		"wpn-1:2": 0, // not exhausted
	}
	state := BuildCooldownState(defs, "wpn-1", charges)

	if !state["wpn-1:0"] {
		t.Error("expected wpn-1:0 on cooldown")
	}
	if state["wpn-1:2"] {
		t.Error("expected wpn-1:2 NOT on cooldown")
	}
}

func TestConsumeCooldown_NilMap(t *testing.T) {
	result := ConsumeCooldown(nil, "wpn-1:0", "1/turn")
	if result == nil {
		t.Fatal("expected non-nil map after consume")
	}
	if result["wpn-1:0"] != 1 {
		t.Errorf("expected charges[wpn-1:0]=1, got %d", result["wpn-1:0"])
	}
}

func TestConsumeCooldown_ExistingMap(t *testing.T) {
	charges := map[string]int{"wpn-1:0": 1}
	result := ConsumeCooldown(charges, "wpn-1:0", "3/turn")
	if result["wpn-1:0"] != 2 {
		t.Errorf("expected charges[wpn-1:0]=2, got %d", result["wpn-1:0"])
	}
}

func TestConsumeCooldown_EmptyCooldownString(t *testing.T) {
	charges := map[string]int{"wpn-1:0": 1}
	result := ConsumeCooldown(charges, "wpn-1:0", "")
	if result["wpn-1:0"] != 1 {
		t.Errorf("expected charges unchanged, got %d", result["wpn-1:0"])
	}
}

func TestConsumeCooldown_NilMapEmptyCooldown(t *testing.T) {
	result := ConsumeCooldown(nil, "wpn-1:0", "")
	if result != nil {
		t.Errorf("expected nil map for empty cooldown, got %v", result)
	}
}

func TestResetCooldowns_ByPeriod(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},       // idx 0
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/short_rest"}, // idx 1
	}
	charges := map[string]int{
		"wpn-1:0": 1, // turn cooldown
		"wpn-1:1": 1, // short_rest cooldown
	}

	result := ResetCooldowns(charges, defs, "wpn-1", PeriodTurn)

	// Turn cooldown should be reset (deleted)
	if _, ok := result["wpn-1:0"]; ok {
		t.Error("expected wpn-1:0 to be reset (deleted)")
	}
	// Short rest cooldown should remain
	if result["wpn-1:1"] != 1 {
		t.Errorf("expected wpn-1:1 unchanged at 1, got %d", result["wpn-1:1"])
	}
}

func TestResetCooldowns_NilMap(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},
	}
	result := ResetCooldowns(nil, defs, "wpn-1", PeriodTurn)
	if result != nil {
		t.Errorf("expected nil for nil input, got %v", result)
	}
}

func TestResetCooldowns_NoMatchingPeriod(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/short_rest"},
	}
	charges := map[string]int{"wpn-1:0": 1}

	result := ResetCooldowns(charges, defs, "wpn-1", PeriodTurn)
	if result["wpn-1:0"] != 1 {
		t.Errorf("expected charges unchanged, got %d", result["wpn-1:0"])
	}
}

func TestResetCooldowns_ShortRest(t *testing.T) {
	defs := []models.TriggerEffect{
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/turn"},       // idx 0
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/short_rest"}, // idx 1
		{Trigger: models.ItemTriggerOnHit, Cooldown: "1/long_rest"},  // idx 2
	}
	charges := map[string]int{
		"wpn-1:0": 1,
		"wpn-1:1": 1,
		"wpn-1:2": 1,
	}

	result := ResetCooldowns(charges, defs, "wpn-1", PeriodShortRest)

	// Only short_rest should be reset
	if result["wpn-1:0"] != 1 {
		t.Errorf("turn cooldown should remain, got %d", result["wpn-1:0"])
	}
	if _, ok := result["wpn-1:1"]; ok {
		t.Error("short_rest cooldown should be reset")
	}
	if result["wpn-1:2"] != 1 {
		t.Errorf("long_rest cooldown should remain, got %d", result["wpn-1:2"])
	}
}
