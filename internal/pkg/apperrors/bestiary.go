package apperrors

import "errors"

var (
	StartPosSizeError      = errors.New("start position or size error")
	UnknownAttackTypeError = errors.New("unknown attack type")
	MixedLangsError        = errors.New("mixed languages")
	UnknownDirectionError  = errors.New("unknown direction type")

	ParseHitBonusError    = errors.New("error while parsing hit bonus")
	ParseDiceError        = errors.New("error while parsing dice count")
	ParseDamageBonusError = errors.New("error while parsing damage bonus")
	NotFoundError         = errors.New("error job not found")
)
