package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// ItemCategory represents the high-level classification of an item.
type ItemCategory string

const (
	ItemCategoryEquipment  ItemCategory = "equipment"
	ItemCategoryConsumable ItemCategory = "consumable"
	ItemCategoryAmmo       ItemCategory = "ammo"
	ItemCategoryUtility    ItemCategory = "utility"
	ItemCategoryQuest      ItemCategory = "quest"
	ItemCategoryReagent    ItemCategory = "reagent"
)

// ItemRarity represents the rarity tier of an item.
type ItemRarity string

const (
	ItemRarityCommon    ItemRarity = "common"
	ItemRarityUncommon  ItemRarity = "uncommon"
	ItemRarityRare      ItemRarity = "rare"
	ItemRarityVeryRare  ItemRarity = "very_rare"
	ItemRarityLegendary ItemRarity = "legendary"
	ItemRarityArtifact  ItemRarity = "artifact"
)

// ItemValue represents a monetary value with currency denomination.
type ItemValue struct {
	Amount   int    `json:"amount" bson:"amount"`
	Currency string `json:"currency" bson:"currency"`
}

// EquipmentData holds equipment-specific properties.
type EquipmentData struct {
	Slot       string   `json:"slot" bson:"slot"`
	Properties []string `json:"properties,omitempty" bson:"properties,omitempty"`
}

// WeaponData holds weapon-specific properties.
type WeaponData struct {
	AttackType string     `json:"attackType" bson:"attackType"`
	DamageDice string     `json:"damageDice" bson:"damageDice"`
	DamageType string     `json:"damageType" bson:"damageType"`
	Properties []string   `json:"properties,omitempty" bson:"properties,omitempty"`
	Range      *RangeData `json:"range,omitempty" bson:"range,omitempty"`
	Reach      int        `json:"reach,omitempty" bson:"reach,omitempty"`
}

// ArmorData holds armor-specific properties.
type ArmorData struct {
	ArmorType string `json:"armorType" bson:"armorType"`
	BaseAC    int    `json:"baseAC" bson:"baseAC"`
	DexCap    *int   `json:"dexCap,omitempty" bson:"dexCap,omitempty"`
	StrReq    int    `json:"strReq,omitempty" bson:"strReq,omitempty"`
	Stealth   string `json:"stealth,omitempty" bson:"stealth,omitempty"`
}

// ConsumableData holds consumable-specific properties.
type ConsumableData struct {
	MaxCharges int    `json:"maxCharges" bson:"maxCharges"`
	RechargeOn string `json:"rechargeOn,omitempty" bson:"rechargeOn,omitempty"`
}

// AmmoData holds ammunition-specific properties.
type AmmoData struct {
	AmmoType string `json:"ammoType" bson:"ammoType"`
}

// ItemDefinition represents a catalog item template (D&D item definition).
type ItemDefinition struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	EngName     string             `json:"engName" bson:"engName"`
	Name        Name               `json:"name" bson:"name"`
	Description Name               `json:"description" bson:"description"`

	Category    ItemCategory `json:"category" bson:"category"`
	Subcategory string       `json:"subcategory,omitempty" bson:"subcategory,omitempty"`
	Type        string       `json:"type,omitempty" bson:"type,omitempty"`
	Tags        []string     `json:"tags,omitempty" bson:"tags,omitempty"`

	Rarity ItemRarity `json:"rarity" bson:"rarity"`
	Tier   int        `json:"tier,omitempty" bson:"tier,omitempty"`
	Value  *ItemValue `json:"value,omitempty" bson:"value,omitempty"`
	Weight float64    `json:"weight,omitempty" bson:"weight,omitempty"`

	Icon string `json:"icon,omitempty" bson:"icon,omitempty"`

	Equipment  *EquipmentData  `json:"equipment,omitempty" bson:"equipment,omitempty"`
	Weapon     *WeaponData     `json:"weapon,omitempty" bson:"weapon,omitempty"`
	Armor      *ArmorData      `json:"armor,omitempty" bson:"armor,omitempty"`
	Consumable *ConsumableData `json:"consumable,omitempty" bson:"consumable,omitempty"`
	Ammo       *AmmoData       `json:"ammo,omitempty" bson:"ammo,omitempty"`

	Modifiers []ItemModifierDef `json:"modifiers,omitempty" bson:"modifiers,omitempty"`
	Triggers  []TriggerEffect   `json:"triggers,omitempty" bson:"triggers,omitempty"`

	RequiresAttunement bool     `json:"requiresAttunement" bson:"requiresAttunement"`
	AttunementBy       []string `json:"attunementBy,omitempty" bson:"attunementBy,omitempty"`
	Prerequisites      []string `json:"prerequisites,omitempty" bson:"prerequisites,omitempty"`

	Source        string `json:"source" bson:"source"`
	IsCustom      bool   `json:"isCustom" bson:"isCustom"`
	CreatedBy     *int   `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
	SchemaVersion int    `json:"schemaVersion" bson:"schemaVersion"`
}

// ItemFilterParams holds query parameters for filtering item definitions.
type ItemFilterParams struct {
	Category ItemCategory `json:"category,omitempty"`
	Rarity   ItemRarity   `json:"rarity,omitempty"`
	Search   string       `json:"search,omitempty"`
	Tags     []string     `json:"tags,omitempty"`
	Page     int          `json:"page"`
	Size     int          `json:"size"`
}

// ItemListResponse wraps a paginated list of item definitions.
type ItemListResponse struct {
	Items []*ItemDefinition `json:"items"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}
