package apperrors

import "errors"

var (
	InvalidInputError  = errors.New("invalid input data")
	InsertMongoDataErr = errors.New("failed to insert data into MongoDB")

	ReadFileError    = errors.New("failed to read file")
	InvalidJSONError = errors.New("invalid json format")

	UnmarashallingJSONError = errors.New("failed to unmarshal JSON")
)
