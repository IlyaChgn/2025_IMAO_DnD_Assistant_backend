package apperrors

import "errors"

var (
	InvalidSpellLevelErr  = errors.New("invalid spell level")
	InvalidSpellSchoolErr = errors.New("invalid spell school")
	SpellNotFoundErr      = errors.New("spell not found")
)
