package apperrors

import "errors"

var (
	CreatingVKReqError     = errors.New("something went wrong while creating request to VK API")
	VKResponseError        = errors.New("something went wrong while getting response from VK API")
	VKIncorrectResponse    = errors.New("vk server responded with incorrect response")
	ReadingVKResponseError = errors.New("something went wrong while reading response from VK API")
)
