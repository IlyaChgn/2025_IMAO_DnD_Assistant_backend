package apperrors

import "errors"

var (
	StartPosSizeError      = errors.New("start position or size error")
	UnknownAttackTypeError = errors.New("unknown attack type")
	MixedLangsError        = errors.New("mixed languages")

	ParseHitBonusError    = errors.New("error while parsing hit bonus")
	ParseDiceError        = errors.New("error while parsing dice count")
	ParseDamageBonusError = errors.New("error while parsing damage bonus")
)
