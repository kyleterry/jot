package errors

import (
	"fmt"
	"net/http"
)

type ErrorType int

const (
	ErrorTypeInvalidPassword ErrorType = iota
	ErrorTypeNotFound
	ErrorTypeUnknown
	ErrorTypeETagMismatch
)

type StoreError struct {
	Type       ErrorType
	Message    string
	StatusCode int
	Causes     []error
}

func (se *StoreError) WithCause(errs ...error) *StoreError {
	se.Causes = append(se.Causes, errs...)

	return se
}

func (se StoreError) Error() string {
	return se.Message
}

func IsStoreError(err error) bool {
	_, ok := err.(*StoreError)

	return ok
}

func NewInvalidPasswordError() *StoreError {
	return &StoreError{
		Type:       ErrorTypeInvalidPassword,
		Message:    "invalid password",
		StatusCode: http.StatusUnauthorized,
	}
}

func NewETagMismatchError() *StoreError {
	return &StoreError{
		Type:       ErrorTypeETagMismatch,
		Message:    "etag mismatch",
		StatusCode: http.StatusPreconditionFailed,
	}
}

func NewUnknownError(msg string) *StoreError {
	return &StoreError{
		Type:       ErrorTypeUnknown,
		Message:    msg,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewNotFoundError(key string) *StoreError {
	return &StoreError{
		Type:       ErrorTypeNotFound,
		Message:    fmt.Sprintf("could not find jot under key: %s", key),
		StatusCode: http.StatusNotFound,
	}
}
