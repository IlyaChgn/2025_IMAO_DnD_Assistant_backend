package models

// CommandType represents the type of inventory command.
type CommandType string

const (
	CmdAdd         CommandType = "ADD"
	CmdRemove      CommandType = "REMOVE"
	CmdMove        CommandType = "MOVE"
	CmdSwap        CommandType = "SWAP"
	CmdSplit       CommandType = "SPLIT"
	CmdMerge       CommandType = "MERGE"
	CmdEquip       CommandType = "EQUIP"
	CmdUnequip     CommandType = "UNEQUIP"
	CmdUse         CommandType = "USE"
	CmdUpdateCoins CommandType = "UPDATE_COINS"
)

// InventoryCommand is a flat struct discriminated by Type.
type InventoryCommand struct {
	Type CommandType `json:"type"`

	// ADD
	DefinitionID string `json:"definitionId,omitempty"`
	Quantity     int    `json:"quantity,omitempty"`
	CustomName   string `json:"customName,omitempty"`

	// REMOVE, EQUIP, UNEQUIP, USE, MOVE, SWAP, SPLIT, MERGE
	ItemID string `json:"itemId,omitempty"`

	// MOVE
	ToContainerID string         `json:"toContainerId,omitempty"`
	ToPlacement   *ItemPlacement `json:"toPlacement,omitempty"`

	// SWAP
	ItemIDA string `json:"itemIdA,omitempty"`
	ItemIDB string `json:"itemIdB,omitempty"`

	// SPLIT
	SplitQuantity int `json:"splitQuantity,omitempty"`

	// MERGE
	SourceItemID string `json:"sourceItemId,omitempty"`
	TargetItemID string `json:"targetItemId,omitempty"`

	// EQUIP / UNEQUIP
	Slot EquipmentSlot `json:"slot,omitempty"`

	// UPDATE_COINS
	Coins *Coins `json:"coins,omitempty"`
}

// CommandRequest is the top-level request for the command endpoint.
type CommandRequest struct {
	ContainerID string           `json:"containerId"`
	Version     int              `json:"version"`
	Command     InventoryCommand `json:"command"`
}

// PatchOp represents the type of change in a patch.
type PatchOp string

const (
	PatchOpAdd    PatchOp = "add"
	PatchOpRemove PatchOp = "remove"
	PatchOpUpdate PatchOp = "update"
)

// ContainerPatch describes a single mutation applied to a container.
type ContainerPatch struct {
	ContainerID string        `json:"containerId"`
	Version     int           `json:"version"`
	Op          PatchOp       `json:"op"`
	Item        *ItemInstance `json:"item,omitempty"`
	Coins       *Coins        `json:"coins,omitempty"`
}

// CommandResponse is the response from the command endpoint.
type CommandResponse struct {
	Version       int                     `json:"version"`
	Patches       []ContainerPatch        `json:"patches"`
	ComputedStats *ComputedCharacterStats `json:"computedStats,omitempty"`
}
