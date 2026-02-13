package apperrors

import "errors"

var (
	InvalidFeatureSourceErr = errors.New("invalid feature source")
	InvalidFeatureLevelErr  = errors.New("invalid feature level")
	FeatureNotFoundErr      = errors.New("feature not found")
)
