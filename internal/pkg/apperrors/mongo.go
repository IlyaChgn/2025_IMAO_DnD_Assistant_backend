package apperrors

import "errors"

var (
	FindMongoDataErr   = errors.New("failed to find data")
	DecodeMongoDataErr = errors.New("failed to decode data")
	UpdateMongoDataErr = errors.New("failed to update data")
	NoDocsErr          = errors.New("no documents found")
	InvalidIDErr       = errors.New("invalid ID format")
)
