package apperrors

import "errors"

var (
	ItemNotFoundErr        = errors.New("item not found")
	ContainerNotFoundErr   = errors.New("container not found")
	NotImplementedErr      = errors.New("not implemented")
	ItemNotOwnedErr        = errors.New("item not owned by user")
	ItemNotCustomErr       = errors.New("cannot modify non-custom item")
	InvalidItemCategoryErr = errors.New("invalid item category")
	InvalidItemRarityErr   = errors.New("invalid item rarity")
	DuplicateEngNameErr    = errors.New("item with this engName already exists")
)
