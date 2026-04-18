package apperr

import "errors"

const(
	CodeInternalError = "INTERNAL_ERROR"
	CodeInvalidInput = "INVALID_INPUT"
	CodeNotFound = "NOT_FOUND"
)

var ErrNotFound = errors.New("not found")

type Error struct{
	Code string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func New(code, message string) *Error {
	return &Error{Code: code, Message: message}
}