package apperrors

import "errors"

var (
	ItemNotFoundErr     = errors.New("item not found")
	ContainerNotFoundErr = errors.New("container not found")
	NotImplementedErr   = errors.New("not implemented")
)
