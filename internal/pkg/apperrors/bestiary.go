package apperrors

import "errors"

var (
	StartPosSizeError             = errors.New("start position or size error")
	UnknownDirectionError         = errors.New("unknown direction type")
	NotFoundError                 = errors.New("error job not found")
	ReceivedActionProcessingError = errors.New("error while actions processing in external gRPC service")
)
