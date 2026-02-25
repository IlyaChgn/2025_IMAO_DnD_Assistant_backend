package seed

import (
	"encoding/json"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/stretchr/testify/assert"
)

var validCategories = map[models.ItemCategory]bool{
	models.ItemCategoryEquipment:  true,
	models.ItemCategoryConsumable: true,
	models.ItemCategoryAmmo:       true,
	models.ItemCategoryUtility:    true,
	models.ItemCategoryQuest:      true,
	models.ItemCategoryReagent:    true,
}

var validRarities = map[models.ItemRarity]bool{
	models.ItemRarityCommon:    true,
	models.ItemRarityUncommon:  true,
	models.ItemRarityRare:      true,
	models.ItemRarityVeryRare:  true,
	models.ItemRarityLegendary: true,
	models.ItemRarityArtifact:  true,
}

func loadSeedItems(t *testing.T) []models.ItemDefinition {
	t.Helper()
	var items []models.ItemDefinition
	err := json.Unmarshal(combinedItemsJSON, &items)
	if !assert.NoError(t, err, "combined item JSONs must be valid") {
		t.FailNow()
	}
	if !assert.NotEmpty(t, items, "combined item JSONs must contain at least one item") {
		t.FailNow()
	}
	return items
}

func TestSeedItems_NoDuplicateEngNames(t *testing.T) {
	items := loadSeedItems(t)

	seen := make(map[string]int)
	for i, item := range items {
		if prev, exists := seen[item.EngName]; exists {
			t.Errorf("duplicate engName %q: index %d and %d", item.EngName, prev, i)
		}
		seen[item.EngName] = i
	}
}

func TestSeedItems_RequiredFields(t *testing.T) {
	items := loadSeedItems(t)

	for i, item := range items {
		t.Run(item.EngName, func(t *testing.T) {
			assert.NotEmpty(t, item.EngName, "item[%d]: engName is required", i)
			assert.NotEmpty(t, item.Name.Eng, "item[%d] %s: name.eng is required", i, item.EngName)
			assert.NotEmpty(t, item.Name.Rus, "item[%d] %s: name.rus is required", i, item.EngName)
			assert.NotEmpty(t, item.Description.Eng, "item[%d] %s: description.eng is required", i, item.EngName)
			assert.NotEmpty(t, item.Description.Rus, "item[%d] %s: description.rus is required", i, item.EngName)
			assert.NotEmpty(t, item.Source, "item[%d] %s: source is required", i, item.EngName)
		})
	}
}

func TestSeedItems_ValidEnums(t *testing.T) {
	items := loadSeedItems(t)

	for i, item := range items {
		t.Run(item.EngName, func(t *testing.T) {
			assert.True(t, validCategories[item.Category],
				"item[%d] %s: invalid category %q", i, item.EngName, item.Category)
			assert.True(t, validRarities[item.Rarity],
				"item[%d] %s: invalid rarity %q", i, item.EngName, item.Rarity)
		})
	}
}

func TestSeedItems_WeaponData(t *testing.T) {
	items := loadSeedItems(t)

	for _, item := range items {
		if item.Weapon == nil {
			continue
		}
		t.Run(item.EngName, func(t *testing.T) {
			assert.NotEmpty(t, item.Weapon.AttackType, "%s: weapon.attackType is required", item.EngName)
			assert.NotEmpty(t, item.Weapon.DamageDice, "%s: weapon.damageDice is required", item.EngName)
			assert.NotEmpty(t, item.Weapon.DamageType, "%s: weapon.damageType is required", item.EngName)

			assert.Contains(t, []string{"melee", "ranged"}, item.Weapon.AttackType,
				"%s: weapon.attackType must be melee or ranged", item.EngName)
		})
	}
}

func TestSeedItems_ArmorData(t *testing.T) {
	items := loadSeedItems(t)

	validArmorTypes := map[string]bool{"light": true, "medium": true, "heavy": true, "shield": true}

	for _, item := range items {
		if item.Armor == nil {
			continue
		}
		t.Run(item.EngName, func(t *testing.T) {
			assert.True(t, validArmorTypes[item.Armor.ArmorType],
				"%s: invalid armor.armorType %q", item.EngName, item.Armor.ArmorType)
			assert.Greater(t, item.Armor.BaseAC, 0,
				"%s: armor.baseAC must be positive", item.EngName)
		})
	}
}

func TestSeedItems_CategoryConsistency(t *testing.T) {
	items := loadSeedItems(t)

	for _, item := range items {
		t.Run(item.EngName, func(t *testing.T) {
			switch item.Category {
			case models.ItemCategoryEquipment:
				hasData := item.Weapon != nil || item.Armor != nil || item.Equipment != nil
				assert.True(t, hasData, "%s: equipment items should have weapon, armor, or equipment data", item.EngName)
			case models.ItemCategoryAmmo:
				assert.NotNil(t, item.Ammo, "%s: ammo items should have ammo data", item.EngName)
			case models.ItemCategoryConsumable:
				assert.NotNil(t, item.Consumable, "%s: consumable items should have consumable data", item.EngName)
			}
		})
	}
}

func TestSeedItems_Count(t *testing.T) {
	items := loadSeedItems(t)
	assert.GreaterOrEqual(t, len(items), 200, "should have at least 200 SRD items")
}

func TestSeedItems_TierField(t *testing.T) {
	items := loadSeedItems(t)

	for _, item := range items {
		switch {
		case item.Category == models.ItemCategoryReagent:
			t.Run("reagent/"+item.EngName, func(t *testing.T) {
				assert.GreaterOrEqual(t, item.Tier, 1, "%s: reagent tier must be >= 1", item.EngName)
				assert.LessOrEqual(t, item.Tier, 5, "%s: reagent tier must be <= 5", item.EngName)
			})
		case item.Tier > 0:
			t.Run("tiered/"+item.EngName, func(t *testing.T) {
				assert.GreaterOrEqual(t, item.Tier, 1, "%s: tier must be >= 1", item.EngName)
				assert.LessOrEqual(t, item.Tier, 5, "%s: tier must be <= 5", item.EngName)
			})
		}
	}
}
