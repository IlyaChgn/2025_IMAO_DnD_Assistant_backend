package usecases

import (
	"context"
	"sort"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

const (
	baseAC              = 10
	placeholderCapacity = 150.0 // STR 10 × 15
	encumberedThreshold = 5.0   // STR × 5
	heavyThreshold      = 10.0  // STR × 10

	modTargetAC         = "ac"
	modTargetACBase     = "ac_base"
	modTargetResistance = "resistance"
	modTargetImmunity   = "immunity"
	modTargetVulnerable = "vulnerability"

	opSet      = "set"
	opAdd      = "add"
	opMultiply = "multiply"
)

// ComputeCharacterStats calculates derived stats from a character container's items.
func (uc *inventoryUsecases) ComputeCharacterStats(ctx context.Context, container *models.InventoryContainer) *models.ComputedCharacterStats {
	l := logger.FromContext(ctx)

	equipped := collectEquippedItems(container)
	equippedDefs := uc.resolveDefinitions(ctx, equipped, l)
	modifiers := collectActiveModifiers(equipped, equippedDefs)

	// Resolve definitions for ALL items (weight includes unequipped)
	allDefs := uc.resolveDefinitions(ctx, container.Items, l)

	ac, breakdown := calculateAC(modifiers)
	totalWeight := calculateTotalWeight(container.Items, allDefs)

	return &models.ComputedCharacterStats{
		AC:                ac,
		ACBreakdown:       breakdown,
		TotalWeight:       totalWeight,
		CarryingCapacity:  placeholderCapacity,
		Encumbered:        totalWeight > placeholderCapacity*encumberedThreshold/15.0,
		HeavilyEncumbered: totalWeight > placeholderCapacity*heavyThreshold/15.0,
		Resistances:       collectDamageTypes(modifiers, modTargetResistance),
		Immunities:        collectDamageTypes(modifiers, modTargetImmunity),
		Vulnerabilities:   collectDamageTypes(modifiers, modTargetVulnerable),
	}
}

func collectEquippedItems(container *models.InventoryContainer) []models.ItemInstance {
	var items []models.ItemInstance
	for _, item := range container.Items {
		if item.IsEquipped {
			items = append(items, item)
		}
	}
	return items
}

func (uc *inventoryUsecases) resolveDefinitions(ctx context.Context, items []models.ItemInstance, l logger.Logger) map[string]*models.ItemDefinition {
	defs := make(map[string]*models.ItemDefinition, len(items))
	for _, item := range items {
		if _, ok := defs[item.DefinitionID]; ok {
			continue
		}
		def, err := uc.itemRepo.GetItemByID(ctx, item.DefinitionID)
		if err != nil {
			l.UsecasesError(err, 0, map[string]any{"definitionId": item.DefinitionID})
			continue
		}
		defs[item.DefinitionID] = def
	}
	return defs
}

func collectActiveModifiers(items []models.ItemInstance, defs map[string]*models.ItemDefinition) []models.ItemModifierDef {
	var mods []models.ItemModifierDef
	for _, item := range items {
		def, ok := defs[item.DefinitionID]
		if !ok {
			continue
		}
		for _, mod := range def.Modifiers {
			if mod.Condition == models.ModifierConditionWhileAttuned && !item.IsAttuned {
				continue
			}
			mods = append(mods, mod)
		}
	}
	return mods
}

func calculateAC(modifiers []models.ItemModifierDef) (int, []models.ACBreakdownEntry) {
	// Sort by priority ascending
	sort.Slice(modifiers, func(i, j int) bool {
		return modifiers[i].Priority < modifiers[j].Priority
	})

	ac := float64(baseAC)
	var breakdown []models.ACBreakdownEntry

	// Phase 1: SET operations (ac_base)
	for _, mod := range modifiers {
		if mod.Target.Type == modTargetACBase && mod.Operation == opSet {
			ac = mod.Value
			breakdown = append(breakdown, models.ACBreakdownEntry{
				Source:    mod.Source,
				Operation: opSet,
				Value:     mod.Value,
			})
		}
	}

	// Phase 2: ADD operations (ac)
	for _, mod := range modifiers {
		if mod.Target.Type == modTargetAC && mod.Operation == opAdd {
			ac += mod.Value
			breakdown = append(breakdown, models.ACBreakdownEntry{
				Source:    mod.Source,
				Operation: opAdd,
				Value:     mod.Value,
			})
		}
	}

	// Phase 3: MULTIPLY operations (ac)
	for _, mod := range modifiers {
		if mod.Target.Type == modTargetAC && mod.Operation == opMultiply {
			ac *= mod.Value
			breakdown = append(breakdown, models.ACBreakdownEntry{
				Source:    mod.Source,
				Operation: opMultiply,
				Value:     mod.Value,
			})
		}
	}

	return int(ac), breakdown
}

func calculateTotalWeight(items []models.ItemInstance, defs map[string]*models.ItemDefinition) float64 {
	var total float64
	for _, item := range items {
		def, ok := defs[item.DefinitionID]
		if !ok {
			continue
		}
		total += def.Weight * float64(item.Quantity)
	}
	return total
}

func collectDamageTypes(modifiers []models.ItemModifierDef, targetType string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, mod := range modifiers {
		if mod.Target.Type == targetType && mod.Target.DamageType != "" {
			if !seen[mod.Target.DamageType] {
				seen[mod.Target.DamageType] = true
				result = append(result, mod.Target.DamageType)
			}
		}
	}
	if result == nil {
		result = []string{}
	}
	return result
}
