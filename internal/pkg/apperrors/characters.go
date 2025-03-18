package apperrors

import "errors"

var (
	// Общие ошибки
	InvalidInputError  = errors.New("invalid input data")
	PlugErr            = errors.New("something went wrong")
	InsertMongoDataErr = errors.New("failed to insert data into MongoDB")

	// Ошибки, связанные с персонажами
	CharacterNotFoundErr      = errors.New("character not found")
	CharacterDataInvalidErr   = errors.New("invalid character data")
	CharacterAlreadyExistsErr = errors.New("character already exists")
	CharacterUpdateFailedErr  = errors.New("failed to update character")
	CharacterDeleteFailedErr  = errors.New("failed to delete character")
)
