package apperror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// Type represents the type of application error.
type Type string

const (
	TypeNotFound      Type = "NOT_FOUND"
	TypeValidation    Type = "VALIDATION"
	TypeConflict      Type = "CONFLICT"
	TypeUnauthorized  Type = "UNAUTHORIZED"
	TypeForbidden     Type = "FORBIDDEN"
	TypeInternal      Type = "INTERNAL"
	TypeBadRequest    Type = "BAD_REQUEST"
	TypeAlreadyExists Type = "ALREADY_EXISTS"
)

// Error represents an application error with type and HTTP status.
type Error struct {
	Type    Type
	Message string
	Code    string
	Err     error
}

func NewError(errType Type, message string, err error) *Error {
	return &Error{Type: errType, Message: message, Err: err}
}

func (e *Error) WithCode(code string) *Error {
	cp := *e
	cp.Code = code
	return &cp
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

func (e *Error) HTTPStatus() int {
	switch e.Type {
	case TypeNotFound:
		return http.StatusNotFound
	case TypeValidation, TypeBadRequest:
		return http.StatusBadRequest
	case TypeConflict, TypeAlreadyExists:
		return http.StatusConflict
	case TypeUnauthorized:
		return http.StatusUnauthorized
	case TypeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func NotFound(message string, err error) *Error   { return NewError(TypeNotFound, message, err) }
func Validation(message string, err error) *Error { return NewError(TypeValidation, message, err) }
func Conflict(message string, err error) *Error   { return NewError(TypeConflict, message, err) }
func AlreadyExists(message string, err error) *Error {
	return NewError(TypeAlreadyExists, message, err)
}
func Internal(message string, err error) *Error     { return NewError(TypeInternal, message, err) }
func Unauthorized(message string, err error) *Error { return NewError(TypeUnauthorized, message, err) }
func Forbidden(message string, err error) *Error    { return NewError(TypeForbidden, message, err) }
func BadRequest(message string, err error) *Error   { return NewError(TypeBadRequest, message, err) }

func IsType(err error, t Type) bool {
	var appErr *Error
	return errors.As(err, &appErr) && appErr.Type == t
}

func IsNotFound(err error) bool   { return IsType(err, TypeNotFound) }
func IsConflict(err error) bool   { return IsType(err, TypeConflict) }
func IsValidation(err error) bool { return IsType(err, TypeValidation) }
func IsForbidden(err error) bool  { return IsType(err, TypeForbidden) }

// ValidateUUID checks if the provided string is a valid UUID.
// Returns a validation error if invalid or empty.
func ValidateUUID(id string, fieldName string, sentinelErr error) error {
	if id == "" {
		return Validation(fmt.Sprintf("%s is required", fieldName), sentinelErr)
	}
	if _, err := uuid.Parse(id); err != nil {
		return Validation(fmt.Sprintf("invalid %s format", fieldName), sentinelErr)
	}
	return nil
}
