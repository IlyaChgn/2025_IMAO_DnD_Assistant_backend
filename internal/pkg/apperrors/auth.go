package apperrors

import "errors"

var (
	SessionNotExistsError   = errors.New("session does not exist")
	DeleteFromRedisError    = errors.New("something went wrong while deleting user from redis db")
	AddToRedisError         = errors.New("something went wrong while creating user session")
	MarshallingSessionError = errors.New("something went wrong while marshalling user session")

	UserDoesNotExistError = errors.New("user does not exist")

	VKApiError  = errors.New("VK API error")
	ClientError = errors.New("client error")
)
