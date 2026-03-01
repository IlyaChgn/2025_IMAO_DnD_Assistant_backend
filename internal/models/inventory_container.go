package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ContainerKind represents the type of inventory container.
type ContainerKind string

const (
	ContainerKindCharacter ContainerKind = "character"
	ContainerKindChest     ContainerKind = "chest"
	ContainerKindLoot      ContainerKind = "loot"
	ContainerKindStash     ContainerKind = "stash"
)

// LayoutType represents the visual layout of a container.
type LayoutType string

const (
	LayoutTypeList      LayoutType = "list"
	LayoutTypeEquipment LayoutType = "equipment"
	LayoutTypeGrid      LayoutType = "grid"
)

// EquipmentSlot represents a canonical equipment location.
type EquipmentSlot string

const (
	SlotHead      EquipmentSlot = "head"
	SlotNeck      EquipmentSlot = "neck"
	SlotShoulders EquipmentSlot = "shoulders"
	SlotChest     EquipmentSlot = "chest"
	SlotHands     EquipmentSlot = "hands"
	SlotWaist     EquipmentSlot = "waist"
	SlotLegs      EquipmentSlot = "legs"
	SlotFeet      EquipmentSlot = "feet"
	SlotRing1     EquipmentSlot = "ring1"
	SlotRing2     EquipmentSlot = "ring2"
	SlotMainHand  EquipmentSlot = "mainHand"
	SlotOffHand   EquipmentSlot = "offHand"
)

// ContainerCapacity defines the limits of a container.
type ContainerCapacity struct {
	MaxItems  int     `json:"maxItems,omitempty" bson:"maxItems,omitempty"`
	MaxWeight float64 `json:"maxWeight,omitempty" bson:"maxWeight,omitempty"`
}

// ContainerPermissions defines access control for a container.
type ContainerPermissions struct {
	OwnerOnly bool `json:"ownerOnly,omitempty" bson:"ownerOnly,omitempty"`
	DmOnly    bool `json:"dmOnly,omitempty" bson:"dmOnly,omitempty"`
}

// ItemPlacement represents the position of an item within a container.
type ItemPlacement struct {
	Index int `json:"index" bson:"index"`
}

// ItemInstance represents a concrete item within a container.
type ItemInstance struct {
	ID           string `json:"id" bson:"id"`
	DefinitionID string `json:"definitionId" bson:"definitionId"`

	Quantity int  `json:"quantity" bson:"quantity"`
	Charges  *int `json:"charges,omitempty" bson:"charges,omitempty"`

	Placement ItemPlacement `json:"placement" bson:"placement"`

	CustomName       string                 `json:"customName,omitempty" bson:"customName,omitempty"`
	CustomProperties map[string]interface{} `json:"customProperties,omitempty" bson:"customProperties,omitempty"`

	IsEquipped   bool          `json:"isEquipped" bson:"isEquipped"`
	EquippedSlot EquipmentSlot `json:"equippedSlot,omitempty" bson:"equippedSlot,omitempty"`
	IsAttuned    bool          `json:"isAttuned" bson:"isAttuned"`
	IsIdentified bool          `json:"isIdentified" bson:"isIdentified"`

	AcquiredAt   time.Time `json:"acquiredAt" bson:"acquiredAt"`
	AcquiredFrom string    `json:"acquiredFrom,omitempty" bson:"acquiredFrom,omitempty"`
}

// InventoryContainer represents a container holding items (embedded).
type InventoryContainer struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"_id"`
	EncounterID string               `json:"encounterId,omitempty" bson:"encounterId,omitempty"`
	OwnerID     string               `json:"ownerId,omitempty" bson:"ownerId,omitempty"`
	Kind        ContainerKind        `json:"kind" bson:"kind"`
	Name        string               `json:"name" bson:"name"`
	Layout      LayoutType           `json:"layout" bson:"layout"`
	Capacity    ContainerCapacity    `json:"capacity" bson:"capacity"`
	Permissions ContainerPermissions `json:"permissions" bson:"permissions"`

	Items     []ItemInstance `json:"items" bson:"items"`
	Equipment *EquippedSlots `json:"equipment,omitempty" bson:"equipment,omitempty"`
	Coins     Coins          `json:"coins" bson:"coins"`

	Version   int       `json:"version" bson:"version"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}

// ContainerFilterParams holds query parameters for filtering containers.
type ContainerFilterParams struct {
	EncounterID string        `json:"encounterId,omitempty"`
	OwnerID     string        `json:"ownerId,omitempty"`
	Kind        ContainerKind `json:"kind,omitempty"`
}

// GenerateLootRequest holds parameters for procedural loot generation.
type GenerateLootRequest struct {
	EncounterID string `json:"encounterId"`
	CR          int    `json:"cr"`
	PartySize   int    `json:"partySize,omitempty"`
	Name        string `json:"name,omitempty"`
}
