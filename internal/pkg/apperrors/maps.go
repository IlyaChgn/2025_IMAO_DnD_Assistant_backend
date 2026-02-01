package apperrors

import "errors"

var (
	MapNotFoundError        = errors.New("map not found")
	MapPermissionDenied     = errors.New("permission denied for map")
	InvalidMapNameError     = errors.New("invalid map name")
	InvalidSchemaVersion    = errors.New("invalid schema version")
	InvalidDimensionsError  = errors.New("invalid map dimensions")
	InvalidPlacementError   = errors.New("invalid placement")
	MapValidationError      = errors.New("map validation failed")
	InvalidUserIDError      = errors.New("invalid user ID")
)
