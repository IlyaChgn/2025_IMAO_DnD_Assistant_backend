package apperrors

import "errors"

var (
	TxError = errors.New("error during transaction")

	QueryError = errors.New("error during query")
	ScanError  = errors.New("error during scanning row")
)
