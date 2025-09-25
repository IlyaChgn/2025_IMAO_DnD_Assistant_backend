package apperrors

import "errors"

var (
	TableNotFoundErr     = errors.New("table not found")
	PlayersNumErr        = errors.New("max players number had already reached")
	UserAlreadyExistsErr = errors.New("user already exists")
)
