package apperrors

import "errors"

var (
	TxStartError  = errors.New("failed to start transaction")
	TxError       = errors.New("error during transaction")
	TxCommitError = errors.New("failed to commit transaction")

	QueryError = errors.New("error during query")
	ScanError  = errors.New("error during scanning row")
)
