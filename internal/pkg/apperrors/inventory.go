package apperrors

import "errors"

var (
	InvalidCommandErr       = errors.New("invalid command")
	InvalidCommandTypeErr   = errors.New("invalid command type")
	ContainerFullErr        = errors.New("container is full")
	ItemNotStackableErr     = errors.New("item is not stackable")
	InsufficientQuantityErr = errors.New("insufficient quantity")
	SlotOccupiedErr         = errors.New("equipment slot is occupied")
	ItemNotEquippedErr      = errors.New("item is not equipped")
	ItemNotConsumableErr    = errors.New("item is not consumable")
	InvalidContainerKindErr = errors.New("invalid container kind")
	InvalidLayoutTypeErr    = errors.New("invalid layout type")
	EmptyContainerNameErr   = errors.New("container name must not be empty")
	NegativeCoinsErr        = errors.New("coin amount cannot be negative")
	ItemNotInContainerErr   = errors.New("item not found in container")
	MissingEncounterIDErr   = errors.New("encounterId is required")
	InvalidCRErr            = errors.New("CR must be between 0 and 30")
)
