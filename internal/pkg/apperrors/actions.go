package apperrors

import "errors"

var (
	InvalidActionTypeErr    = errors.New("invalid action type")
	MissingAbilityErr       = errors.New("ability field is required")
	MissingDCErr            = errors.New("dc field is required for saving throw")
	MissingDiceExprErr      = errors.New("dice field is required for custom roll")
	InvalidDiceExprErr      = errors.New("invalid dice expression")
	MissingCharacterIDErr   = errors.New("characterId is required")
	ParticipantNotFoundErr  = errors.New("participant not found in encounter")
	EncounterNotFoundErr    = errors.New("encounter not found")
	MissingWeaponIDErr      = errors.New("weaponId is required for weapon attack")
	InsufficientSlotsErr    = errors.New("insufficient spell slots")
	WeaponNotFoundErr       = errors.New("weapon not found on character")
	SpellNotKnownErr        = errors.New("spell not known or prepared")
	FeatureUsesExhaustedErr = errors.New("feature uses exhausted")
	MissingSpellIDErr       = errors.New("spellId is required for spell cast")
	MissingFeatureIDErr     = errors.New("featureId is required for use feature")
)
