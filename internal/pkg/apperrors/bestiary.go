package apperrors

import "errors"

var (
	StartPosSizeError             = errors.New("start position or size error")
	UnknownDirectionError         = errors.New("unknown direction type")
	NotFoundError                 = errors.New("error job not found")
	ReceivedActionProcessingError = errors.New("error while actions processing in external gRPC service")
	ParsedActionsErr              = errors.New("missing parsed_actions_field")
	NilCreatureErr                = errors.New("nil creature")
	InvalidBase64Err              = errors.New("invalid Base64 format")

	ApiErr = errors.New("api error")
)
