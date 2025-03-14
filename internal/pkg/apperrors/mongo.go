package apperrors

import "errors"

var (
	FindMongoDataErr   = errors.New("failed to find data")
	DecodeMongoDataErr = errors.New("failed to decode data")
	NoDocsErr          = errors.New("no documents found")
)
