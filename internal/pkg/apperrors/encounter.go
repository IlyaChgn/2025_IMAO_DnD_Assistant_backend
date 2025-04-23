package apperrors

import "errors"

var (
	PermissionDeniedError = errors.New("permission denied")
)
