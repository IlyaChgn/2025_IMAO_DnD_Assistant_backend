package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	itemsinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/items"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

var validContainerKinds = map[models.ContainerKind]bool{
	models.ContainerKindCharacter: true,
	models.ContainerKindChest:     true,
	models.ContainerKindLoot:      true,
	models.ContainerKindStash:     true,
}

var validLayoutTypes = map[models.LayoutType]bool{
	models.LayoutTypeList:      true,
	models.LayoutTypeEquipment: true,
	models.LayoutTypeGrid:      true,
}

var validEquipmentSlots = map[models.EquipmentSlot]bool{
	models.SlotHead:      true,
	models.SlotNeck:      true,
	models.SlotShoulders: true,
	models.SlotChest:     true,
	models.SlotHands:     true,
	models.SlotWaist:     true,
	models.SlotLegs:      true,
	models.SlotFeet:      true,
	models.SlotRing1:     true,
	models.SlotRing2:     true,
	models.SlotMainHand:  true,
	models.SlotOffHand:   true,
}

type inventoryUsecases struct {
	repo     itemsinterfaces.InventoryRepository
	itemRepo itemsinterfaces.ItemDefinitionRepository
}

func NewInventoryUsecases(repo itemsinterfaces.InventoryRepository, itemRepo itemsinterfaces.ItemDefinitionRepository) itemsinterfaces.InventoryUsecases {
	return &inventoryUsecases{repo: repo, itemRepo: itemRepo}
}

func (uc *inventoryUsecases) GetContainer(ctx context.Context, id string) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidIDErr, 0, map[string]any{"id": id})
		return nil, apperrors.InvalidIDErr
	}

	container, err := uc.repo.GetContainer(ctx, id)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"id": id})
		return nil, err
	}

	return container, nil
}

func (uc *inventoryUsecases) GetContainers(ctx context.Context, filter models.ContainerFilterParams) ([]*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	containers, err := uc.repo.GetContainers(ctx, filter)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"filter": filter})
		return nil, err
	}

	return containers, nil
}

func (uc *inventoryUsecases) CreateContainer(ctx context.Context, container *models.InventoryContainer) (*models.InventoryContainer, error) {
	l := logger.FromContext(ctx)

	if container.Name == "" {
		l.UsecasesWarn(apperrors.EmptyContainerNameErr, 0, nil)
		return nil, apperrors.EmptyContainerNameErr
	}
	if !validContainerKinds[container.Kind] {
		l.UsecasesWarn(apperrors.InvalidContainerKindErr, 0, map[string]any{"kind": container.Kind})
		return nil, apperrors.InvalidContainerKindErr
	}
	if container.Kind == models.ContainerKindChest && container.EncounterID == "" {
		l.UsecasesWarn(apperrors.MissingEncounterIDErr, 0, nil)
		return nil, apperrors.MissingEncounterIDErr
	}
	if container.Layout != "" && !validLayoutTypes[container.Layout] {
		l.UsecasesWarn(apperrors.InvalidLayoutTypeErr, 0, map[string]any{"layout": container.Layout})
		return nil, apperrors.InvalidLayoutTypeErr
	}
	if container.Layout == "" {
		container.Layout = models.LayoutTypeList
	}

	created, err := uc.repo.CreateContainer(ctx, container)
	if err != nil {
		l.UsecasesError(err, 0, map[string]any{"name": container.Name})
		return nil, err
	}

	return created, nil
}

func (uc *inventoryUsecases) DeleteContainer(ctx context.Context, id string, userID int) error {
	l := logger.FromContext(ctx)

	_, err := uc.repo.GetContainer(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return err
	}

	if err := uc.repo.DeleteContainer(ctx, id); err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return err
	}

	return nil
}

func (uc *inventoryUsecases) ExecuteCommand(ctx context.Context, req *models.CommandRequest, userID int) (*models.CommandResponse, error) {
	l := logger.FromContext(ctx)

	if req.ContainerID == "" {
		l.UsecasesWarn(apperrors.InvalidCommandErr, userID, map[string]any{"reason": "empty containerId"})
		return nil, apperrors.InvalidCommandErr
	}

	container, err := uc.repo.GetContainer(ctx, req.ContainerID)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"containerId": req.ContainerID})
		return nil, err
	}

	if container.Version != req.Version {
		l.UsecasesWarn(apperrors.VersionConflictErr, userID, map[string]any{
			"expected": req.Version,
			"actual":   container.Version,
		})
		return nil, apperrors.VersionConflictErr
	}

	var patches []models.ContainerPatch
	equipmentChanged := false
	cmd := &req.Command

	switch cmd.Type {
	case models.CmdAdd:
		patches, err = uc.executeAdd(ctx, container, cmd)
	case models.CmdRemove:
		patches, equipmentChanged, err = uc.executeRemove(container, cmd)
	case models.CmdMove:
		if cmd.ToContainerID != "" && cmd.ToContainerID != req.ContainerID {
			return uc.executeCrossContainerMove(ctx, req, cmd, userID)
		}
		patches, err = uc.executeMove(container, cmd)
	case models.CmdSwap:
		patches, err = uc.executeSwap(container, cmd)
	case models.CmdSplit:
		patches, err = uc.executeSplit(ctx, container, cmd)
	case models.CmdMerge:
		patches, err = uc.executeMerge(container, cmd)
	case models.CmdEquip:
		patches, err = uc.executeEquip(ctx, container, cmd)
		equipmentChanged = err == nil
	case models.CmdUnequip:
		patches, err = uc.executeUnequip(container, cmd)
		equipmentChanged = err == nil
	case models.CmdUse:
		patches, err = uc.executeUse(ctx, container, cmd)
	case models.CmdUpdateCoins:
		patches, err = uc.executeUpdateCoins(container, cmd)
	default:
		l.UsecasesWarn(apperrors.InvalidCommandTypeErr, userID, map[string]any{"type": cmd.Type})
		return nil, apperrors.InvalidCommandTypeErr
	}

	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"command": cmd.Type})
		return nil, err
	}

	_, err = uc.repo.UpdateContainerWithVersion(ctx, container, req.Version)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"containerId": req.ContainerID})
		return nil, err
	}

	resp := &models.CommandResponse{
		Version: container.Version,
		Patches: patches,
	}

	if container.Kind == models.ContainerKindCharacter && equipmentChanged {
		stats := uc.ComputeCharacterStats(ctx, container)
		resp.ComputedStats = stats
	}

	return resp, nil
}

// executeAdd resolves the definition, creates a new ItemInstance, and appends it.
func (uc *inventoryUsecases) executeAdd(ctx context.Context, container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	if cmd.DefinitionID == "" {
		return nil, apperrors.InvalidCommandErr
	}

	def, err := uc.itemRepo.GetItemByID(ctx, cmd.DefinitionID)
	if err != nil {
		return nil, err
	}

	qty := cmd.Quantity
	if qty <= 0 {
		qty = 1
	}

	item := models.ItemInstance{
		ID:           uuid.New().String(),
		DefinitionID: def.ID.Hex(),
		Quantity:     qty,
		Placement:    models.ItemPlacement{Index: len(container.Items)},
		IsIdentified: true,
		AcquiredAt:   time.Now(),
	}
	if cmd.CustomName != "" {
		item.CustomName = cmd.CustomName
	}
	if def.Consumable != nil {
		charges := def.Consumable.MaxCharges
		item.Charges = &charges
	}

	container.Items = append(container.Items, item)

	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpAdd,
		Item:        &item,
	}}, nil
}

// executeRemove decrements quantity or removes an item entirely.
func (uc *inventoryUsecases) executeRemove(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, bool, error) {
	idx := findItemIndex(container, cmd.ItemID)
	if idx < 0 {
		return nil, false, apperrors.ItemNotInContainerErr
	}

	equipmentChanged := container.Items[idx].IsEquipped

	qty := cmd.Quantity
	if qty <= 0 || qty >= container.Items[idx].Quantity {
		// Copy before removal to avoid dangling pointer
		removedItem := container.Items[idx]
		if removedItem.IsEquipped && container.Equipment != nil {
			container.Equipment.SetSlot(removedItem.EquippedSlot, "")
		}
		container.Items = append(container.Items[:idx], container.Items[idx+1:]...)
		return []models.ContainerPatch{{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpRemove,
			Item:        &removedItem,
		}}, equipmentChanged, nil
	}

	container.Items[idx].Quantity -= qty
	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Item:        &container.Items[idx],
	}}, false, nil
}

// executeMove updates item placement within the same container.
func (uc *inventoryUsecases) executeMove(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	idx := findItemIndex(container, cmd.ItemID)
	if idx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	if cmd.ToPlacement == nil {
		return nil, apperrors.InvalidCommandErr
	}

	container.Items[idx].Placement = *cmd.ToPlacement

	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Item:        &container.Items[idx],
	}}, nil
}

// executeCrossContainerMove handles moving items between containers.
func (uc *inventoryUsecases) executeCrossContainerMove(ctx context.Context, req *models.CommandRequest, cmd *models.InventoryCommand, userID int) (*models.CommandResponse, error) {
	l := logger.FromContext(ctx)

	placement := models.ItemPlacement{}
	if cmd.ToPlacement != nil {
		placement = *cmd.ToPlacement
	}

	source, target, err := uc.repo.MoveItemAcrossContainers(ctx, req.ContainerID, req.Version, cmd.ToContainerID, cmd.ItemID, placement)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{
			"source": req.ContainerID,
			"target": cmd.ToContainerID,
			"itemID": cmd.ItemID,
		})
		return nil, err
	}

	patches := []models.ContainerPatch{
		{
			ContainerID: source.ID.Hex(),
			Version:     source.Version,
			Op:          models.PatchOpRemove,
			Item:        &models.ItemInstance{ID: cmd.ItemID},
		},
		{
			ContainerID: target.ID.Hex(),
			Version:     target.Version,
			Op:          models.PatchOpAdd,
			Item:        findItem(target, cmd.ItemID),
		},
	}

	return &models.CommandResponse{
		Version: source.Version,
		Patches: patches,
	}, nil
}

// executeSwap swaps placements of two items.
func (uc *inventoryUsecases) executeSwap(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	idxA := findItemIndex(container, cmd.ItemIDA)
	idxB := findItemIndex(container, cmd.ItemIDB)
	if idxA < 0 || idxB < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	container.Items[idxA].Placement, container.Items[idxB].Placement = container.Items[idxB].Placement, container.Items[idxA].Placement

	return []models.ContainerPatch{
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpUpdate,
			Item:        &container.Items[idxA],
		},
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpUpdate,
			Item:        &container.Items[idxB],
		},
	}, nil
}

// executeSplit splits a stackable item into two instances.
func (uc *inventoryUsecases) executeSplit(ctx context.Context, container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	idx := findItemIndex(container, cmd.ItemID)
	if idx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	item := &container.Items[idx]

	if cmd.SplitQuantity <= 0 || cmd.SplitQuantity >= item.Quantity {
		return nil, apperrors.InsufficientQuantityErr
	}

	// Verify stackable via definition
	def, err := uc.itemRepo.GetItemByID(ctx, item.DefinitionID)
	if err != nil {
		return nil, err
	}
	// Non-equipment items are stackable by default; equipment is not
	if def.Category == models.ItemCategoryEquipment {
		return nil, apperrors.ItemNotStackableErr
	}

	item.Quantity -= cmd.SplitQuantity

	placement := models.ItemPlacement{Index: len(container.Items)}
	if cmd.ToPlacement != nil {
		placement = *cmd.ToPlacement
	}

	newItem := models.ItemInstance{
		ID:           uuid.New().String(),
		DefinitionID: item.DefinitionID,
		Quantity:     cmd.SplitQuantity,
		Placement:    placement,
		CustomName:   item.CustomName,
		IsIdentified: item.IsIdentified,
		AcquiredAt:   time.Now(),
	}

	container.Items = append(container.Items, newItem)

	return []models.ContainerPatch{
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpUpdate,
			Item:        item,
		},
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpAdd,
			Item:        &newItem,
		},
	}, nil
}

// executeMerge merges two items of the same definition.
func (uc *inventoryUsecases) executeMerge(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	srcIdx := findItemIndex(container, cmd.SourceItemID)
	tgtIdx := findItemIndex(container, cmd.TargetItemID)
	if srcIdx < 0 || tgtIdx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	if container.Items[srcIdx].DefinitionID != container.Items[tgtIdx].DefinitionID {
		return nil, apperrors.InvalidCommandErr
	}

	// Copy source before removal, merge quantity into target
	srcCopy := container.Items[srcIdx]
	container.Items[tgtIdx].Quantity += srcCopy.Quantity

	// Remove source — adjust tgtIdx if it shifts
	container.Items = append(container.Items[:srcIdx], container.Items[srcIdx+1:]...)
	if tgtIdx > srcIdx {
		tgtIdx--
	}

	return []models.ContainerPatch{
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpRemove,
			Item:        &srcCopy,
		},
		{
			ContainerID: container.ID.Hex(),
			Version:     container.Version + 1,
			Op:          models.PatchOpUpdate,
			Item:        &container.Items[tgtIdx],
		},
	}, nil
}

// executeEquip equips an item into a slot.
func (uc *inventoryUsecases) executeEquip(ctx context.Context, container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	idx := findItemIndex(container, cmd.ItemID)
	if idx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	item := &container.Items[idx]

	// Resolve slot from definition if not given
	slot := cmd.Slot
	if slot == "" {
		def, err := uc.itemRepo.GetItemByID(ctx, item.DefinitionID)
		if err != nil {
			return nil, err
		}
		if def.Equipment != nil && def.Equipment.Slot != "" {
			slot = models.EquipmentSlot(def.Equipment.Slot)
		} else {
			return nil, apperrors.InvalidCommandErr
		}
	}

	if !validEquipmentSlots[slot] {
		return nil, apperrors.InvalidCommandErr
	}

	if container.Equipment == nil {
		container.Equipment = &models.EquippedSlots{}
	}

	// Auto-unequip existing item in slot
	patches := make([]models.ContainerPatch, 0, 2)
	existingID := container.Equipment.GetSlot(slot)
	if existingID != "" && existingID != item.ID {
		existingIdx := findItemIndex(container, existingID)
		if existingIdx >= 0 {
			container.Items[existingIdx].IsEquipped = false
			container.Items[existingIdx].EquippedSlot = ""
			patches = append(patches, models.ContainerPatch{
				ContainerID: container.ID.Hex(),
				Version:     container.Version + 1,
				Op:          models.PatchOpUpdate,
				Item:        &container.Items[existingIdx],
			})
		}
	}

	item.IsEquipped = true
	item.EquippedSlot = slot
	container.Equipment.SetSlot(slot, item.ID)

	patches = append(patches, models.ContainerPatch{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Item:        item,
	})

	return patches, nil
}

// executeUnequip unequips an item from its slot.
func (uc *inventoryUsecases) executeUnequip(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	// Find item by itemId or by slot
	var idx int
	if cmd.ItemID != "" {
		idx = findItemIndex(container, cmd.ItemID)
	} else if cmd.Slot != "" && container.Equipment != nil {
		existingID := container.Equipment.GetSlot(cmd.Slot)
		if existingID == "" {
			return nil, apperrors.ItemNotEquippedErr
		}
		idx = findItemIndex(container, existingID)
	} else {
		return nil, apperrors.InvalidCommandErr
	}

	if idx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	item := &container.Items[idx]
	if !item.IsEquipped {
		return nil, apperrors.ItemNotEquippedErr
	}

	if container.Equipment != nil {
		container.Equipment.SetSlot(item.EquippedSlot, "")
	}

	item.IsEquipped = false
	item.EquippedSlot = ""

	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Item:        item,
	}}, nil
}

// executeUse uses a consumable item (decrements charges/quantity).
func (uc *inventoryUsecases) executeUse(ctx context.Context, container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	idx := findItemIndex(container, cmd.ItemID)
	if idx < 0 {
		return nil, apperrors.ItemNotInContainerErr
	}

	def, err := uc.itemRepo.GetItemByID(ctx, container.Items[idx].DefinitionID)
	if err != nil {
		return nil, err
	}

	if def.Category != models.ItemCategoryConsumable && def.Consumable == nil {
		return nil, apperrors.ItemNotConsumableErr
	}

	// Decrement charges if present
	item := &container.Items[idx]
	if item.Charges != nil && *item.Charges > 0 {
		*item.Charges--
		if *item.Charges <= 0 {
			// Copy before removal to avoid dangling pointer
			removedItem := container.Items[idx]
			container.Items = append(container.Items[:idx], container.Items[idx+1:]...)
			return []models.ContainerPatch{{
				ContainerID: container.ID.Hex(),
				Version:     container.Version + 1,
				Op:          models.PatchOpRemove,
				Item:        &removedItem,
			}}, nil
		}
	} else {
		// Decrement quantity
		item.Quantity--
		if item.Quantity <= 0 {
			removedItem := container.Items[idx]
			container.Items = append(container.Items[:idx], container.Items[idx+1:]...)
			return []models.ContainerPatch{{
				ContainerID: container.ID.Hex(),
				Version:     container.Version + 1,
				Op:          models.PatchOpRemove,
				Item:        &removedItem,
			}}, nil
		}
	}

	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Item:        &container.Items[idx],
	}}, nil
}

// executeUpdateCoins adds delta coins to the container.
func (uc *inventoryUsecases) executeUpdateCoins(container *models.InventoryContainer, cmd *models.InventoryCommand) ([]models.ContainerPatch, error) {
	if cmd.Coins == nil {
		return nil, apperrors.InvalidCommandErr
	}

	// Validate before mutating to keep container consistent on error
	newCp := container.Coins.Cp + cmd.Coins.Cp
	newSp := container.Coins.Sp + cmd.Coins.Sp
	newEp := container.Coins.Ep + cmd.Coins.Ep
	newGp := container.Coins.Gp + cmd.Coins.Gp
	newPp := container.Coins.Pp + cmd.Coins.Pp

	if newCp < 0 || newSp < 0 || newEp < 0 || newGp < 0 || newPp < 0 {
		return nil, apperrors.NegativeCoinsErr
	}

	container.Coins.Cp = newCp
	container.Coins.Sp = newSp
	container.Coins.Ep = newEp
	container.Coins.Gp = newGp
	container.Coins.Pp = newPp

	return []models.ContainerPatch{{
		ContainerID: container.ID.Hex(),
		Version:     container.Version + 1,
		Op:          models.PatchOpUpdate,
		Coins:       &container.Coins,
	}}, nil
}

// findItemIndex returns the index of an item in the container by its ID, or -1 if not found.
func findItemIndex(container *models.InventoryContainer, itemID string) int {
	for i := range container.Items {
		if container.Items[i].ID == itemID {
			return i
		}
	}
	return -1
}

// findItem returns a pointer to an item in the container by its ID, or nil.
func findItem(container *models.InventoryContainer, itemID string) *models.ItemInstance {
	idx := findItemIndex(container, itemID)
	if idx < 0 {
		return nil
	}
	return &container.Items[idx]
}

