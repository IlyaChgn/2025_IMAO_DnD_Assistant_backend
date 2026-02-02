package apperrors

import "errors"

var (
	SessionNotExistsError   = errors.New("session does not exist")
	DeleteFromRedisError    = errors.New("something went wrong while deleting user from redis db")
	AddToRedisError         = errors.New("something went wrong while creating user session")
	MarshallingSessionError = errors.New("something went wrong while marshalling user session")

	UserDoesNotExistError = errors.New("user does not exist")

	IdentityNotFoundError = errors.New("identity not found")

	UnsupportedProviderError = errors.New("unsupported OAuth provider")

	OAuthProviderError = errors.New("OAuth provider error")
	VKApiError         = errors.New("VK API error")
	ClientError        = errors.New("client error")

	IdentityAlreadyLinkedError = errors.New("identity already linked to another user")
	LastIdentityError          = errors.New("cannot unlink last identity")
)
