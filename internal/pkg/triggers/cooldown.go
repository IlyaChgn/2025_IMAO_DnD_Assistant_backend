package triggers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// CooldownPeriod enumerates supported reset periods.
type CooldownPeriod string

const (
	PeriodTurn      CooldownPeriod = "turn"
	PeriodShortRest CooldownPeriod = "short_rest"
	PeriodLongRest  CooldownPeriod = "long_rest"
	PeriodDawn      CooldownPeriod = "dawn"
)

// ParsedCooldown represents a parsed "N/period" string.
type ParsedCooldown struct {
	MaxUses int
	Period  CooldownPeriod
}

var validPeriods = map[CooldownPeriod]bool{
	PeriodTurn:      true,
	PeriodShortRest: true,
	PeriodLongRest:  true,
	PeriodDawn:      true,
}

// ParseCooldown parses strings like "1/turn", "3/short_rest".
// Returns nil, nil for empty string (no cooldown). Error for malformed input.
func ParseCooldown(s string) (*ParsedCooldown, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cooldown format %q: expected N/period", s)
	}

	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid cooldown uses %q: %w", parts[0], err)
	}
	if n <= 0 {
		return nil, fmt.Errorf("invalid cooldown uses %d: must be > 0", n)
	}

	period := CooldownPeriod(parts[1])
	if !validPeriods[period] {
		return nil, fmt.Errorf("unknown cooldown period %q", parts[1])
	}

	return &ParsedCooldown{MaxUses: n, Period: period}, nil
}

// BuildCooldownState converts integer charge counts into the engine's
// boolean CooldownState. A key is "on cooldown" if charges[key] >= maxUses.
func BuildCooldownState(
	triggerDefs []models.TriggerEffect,
	sourceID string,
	charges map[string]int,
) models.CooldownState {
	if len(charges) == 0 {
		return nil
	}

	var state models.CooldownState
	for i, td := range triggerDefs {
		if td.Cooldown == "" {
			continue
		}
		parsed, err := ParseCooldown(td.Cooldown)
		if err != nil {
			continue
		}
		key := sourceID + ":" + strconv.Itoa(i)
		if charges[key] >= parsed.MaxUses {
			if state == nil {
				state = make(models.CooldownState)
			}
			state[key] = true
		}
	}
	return state
}

// ConsumeCooldown increments the charge count for a trigger key.
// Lazy-inits the map if nil. Returns the (possibly new) map.
// No-op if cooldownStr is empty (trigger has no cooldown).
func ConsumeCooldown(
	charges map[string]int,
	key string,
	cooldownStr string,
) map[string]int {
	if cooldownStr == "" {
		return charges
	}
	if charges == nil {
		charges = make(map[string]int)
	}
	charges[key]++
	return charges
}

// ResetCooldowns zeroes out charges for triggers matching the given period.
// Used on turn advance, short rest, long rest, dawn.
// Returns the (possibly nil) map.
func ResetCooldowns(
	charges map[string]int,
	triggerDefs []models.TriggerEffect,
	sourceID string,
	period CooldownPeriod,
) map[string]int {
	if charges == nil {
		return nil
	}

	for i, td := range triggerDefs {
		if td.Cooldown == "" {
			continue
		}
		parsed, err := ParseCooldown(td.Cooldown)
		if err != nil {
			continue
		}
		if parsed.Period == period {
			key := sourceID + ":" + strconv.Itoa(i)
			delete(charges, key)
		}
	}
	return charges
}
