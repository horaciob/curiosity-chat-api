package apperror

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	sentinel := errors.New("underlying error")
	err := NewError(TypeNotFound, "resource not found", sentinel)

	assert.Equal(t, TypeNotFound, err.Type)
	assert.Equal(t, "resource not found", err.Message)
	assert.Equal(t, sentinel, err.Err)
}

func TestNewErrorWithoutUnderlying(t *testing.T) {
	err := NewError(TypeValidation, "invalid input", nil)

	assert.Equal(t, TypeValidation, err.Type)
	assert.Equal(t, "invalid input", err.Message)
	assert.Nil(t, err.Err)
}

func TestErrorWithCode(t *testing.T) {
	original := NewError(TypeConflict, "duplicate entry", errors.New("db error"))
	modified := original.WithCode("DUPLICATE_001")

	// Original should not be modified
	assert.Empty(t, original.Code)
	// Modified should have the code
	assert.Equal(t, "DUPLICATE_001", modified.Code)
	assert.Equal(t, original.Type, modified.Type)
	assert.Equal(t, original.Message, modified.Message)
	assert.Equal(t, original.Err, modified.Err)
}

func TestErrorMessageWithUnderlying(t *testing.T) {
	sentinel := errors.New("database connection failed")
	err := NewError(TypeInternal, "server error", sentinel)

	msg := err.Error()
	assert.Contains(t, msg, "INTERNAL")
	assert.Contains(t, msg, "server error")
	assert.Contains(t, msg, "database connection failed")
}

func TestErrorMessageWithoutUnderlying(t *testing.T) {
	err := NewError(TypeNotFound, "user not found", nil)

	msg := err.Error()
	assert.Equal(t, "NOT_FOUND: user not found", msg)
}

func TestErrorUnwrap(t *testing.T) {
	sentinel := errors.New("underlying error")
	err := NewError(TypeValidation, "validation failed", sentinel)

	unwrapped := err.Unwrap()
	assert.Equal(t, sentinel, unwrapped)
}

func TestErrorUnwrapNil(t *testing.T) {
	err := NewError(TypeNotFound, "not found", nil)

	unwrapped := err.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name       string
		errType    Type
		wantStatus int
	}{
		{"not found", TypeNotFound, http.StatusNotFound},
		{"validation", TypeValidation, http.StatusBadRequest},
		{"bad request", TypeBadRequest, http.StatusBadRequest},
		{"conflict", TypeConflict, http.StatusConflict},
		{"already exists", TypeAlreadyExists, http.StatusConflict},
		{"unauthorized", TypeUnauthorized, http.StatusUnauthorized},
		{"forbidden", TypeForbidden, http.StatusForbidden},
		{"internal", TypeInternal, http.StatusInternalServerError},
		{"unknown type", Type("UNKNOWN"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.errType, "test", nil)
			assert.Equal(t, tt.wantStatus, err.HTTPStatus())
		})
	}
}

func TestConstructorFunctions(t *testing.T) {
	sentinel := errors.New("test error")

	tests := []struct {
		name     string
		fn       func(string, error) *Error
		wantType Type
		wantMsg  string
		wantCode int
	}{
		{"NotFound", NotFound, TypeNotFound, "user not found", http.StatusNotFound},
		{"Validation", Validation, TypeValidation, "invalid input", http.StatusBadRequest},
		{"Conflict", Conflict, TypeConflict, "duplicate", http.StatusConflict},
		{"AlreadyExists", AlreadyExists, TypeAlreadyExists, "exists", http.StatusConflict},
		{"Internal", Internal, TypeInternal, "internal error", http.StatusInternalServerError},
		{"Unauthorized", Unauthorized, TypeUnauthorized, "unauthorized", http.StatusUnauthorized},
		{"Forbidden", Forbidden, TypeForbidden, "forbidden", http.StatusForbidden},
		{"BadRequest", BadRequest, TypeBadRequest, "bad request", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.wantMsg, sentinel)
			assert.Equal(t, tt.wantType, err.Type)
			assert.Equal(t, tt.wantMsg, err.Message)
			assert.Equal(t, sentinel, err.Err)
			assert.Equal(t, tt.wantCode, err.HTTPStatus())
		})
	}
}

func TestConstructorFunctionsWithNilError(t *testing.T) {
	err := NotFound("not found", nil)
	assert.Nil(t, err.Err)
	assert.Equal(t, "NOT_FOUND: not found", err.Error())
}

func TestIsType(t *testing.T) {
	err := NotFound("test", nil)

	assert.True(t, IsType(err, TypeNotFound))
	assert.False(t, IsType(err, TypeValidation))
	assert.False(t, IsType(err, TypeConflict))
}

func TestIsTypeWithRegularError(t *testing.T) {
	err := errors.New("regular error")

	assert.False(t, IsType(err, TypeNotFound))
	assert.False(t, IsType(err, TypeValidation))
}

func TestIsTypeWithNil(t *testing.T) {
	assert.False(t, IsType(nil, TypeNotFound))
}

func TestIsNotFound(t *testing.T) {
	notFoundErr := NotFound("test", nil)
	validationErr := Validation("test", nil)

	assert.True(t, IsNotFound(notFoundErr))
	assert.False(t, IsNotFound(validationErr))
	assert.False(t, IsNotFound(errors.New("regular")))
	assert.False(t, IsNotFound(nil))
}

func TestIsConflict(t *testing.T) {
	conflictErr := Conflict("test", nil)
	validationErr := Validation("test", nil)

	assert.True(t, IsConflict(conflictErr))
	assert.False(t, IsConflict(validationErr))
	assert.False(t, IsConflict(errors.New("regular")))
	assert.False(t, IsConflict(nil))
}

func TestIsValidation(t *testing.T) {
	validationErr := Validation("test", nil)
	notFoundErr := NotFound("test", nil)

	assert.True(t, IsValidation(validationErr))
	assert.False(t, IsValidation(notFoundErr))
	assert.False(t, IsValidation(errors.New("regular")))
	assert.False(t, IsValidation(nil))
}

func TestIsForbidden(t *testing.T) {
	forbiddenErr := Forbidden("test", nil)
	notFoundErr := NotFound("test", nil)

	assert.True(t, IsForbidden(forbiddenErr))
	assert.False(t, IsForbidden(notFoundErr))
	assert.False(t, IsForbidden(errors.New("regular")))
	assert.False(t, IsForbidden(nil))
}

func TestIsTypeHelperVariations(t *testing.T) {
	// Test wrapping with standard errors
	wrappedNotFound := errors.New("wrapped")
	appErr := NotFound("original", wrappedNotFound)

	assert.True(t, IsNotFound(appErr))
	assert.True(t, errors.Is(appErr, wrappedNotFound))
}

func TestErrorTypesAreDistinct(t *testing.T) {
	assert.NotEqual(t, TypeNotFound, TypeValidation)
	assert.NotEqual(t, TypeConflict, TypeAlreadyExists)
	assert.NotEqual(t, TypeInternal, TypeUnauthorized)
	assert.NotEqual(t, TypeForbidden, TypeBadRequest)
}

func TestValidateUUIDSuccess(t *testing.T) {
	validUUID := uuid.New().String()
	err := ValidateUUID(validUUID, "conversation ID", errors.New("invalid"))
	assert.NoError(t, err)
}

func TestValidateUUIDEmpty(t *testing.T) {
	err := ValidateUUID("", "user ID", errors.New("required"))
	assert.Error(t, err)
	assert.True(t, IsValidation(err))
	assert.Contains(t, err.Error(), "user ID is required")
}

func TestValidateUUIDInvalid(t *testing.T) {
	err := ValidateUUID("not-a-uuid", "conversation ID", errors.New("invalid"))
	assert.Error(t, err)
	assert.True(t, IsValidation(err))
	assert.Contains(t, err.Error(), "invalid conversation ID format")
}

func TestValidateUUIDInvalidFormats(t *testing.T) {
	invalidUUIDs := []string{
		"123",
		"too-short",
		"12345678-1234-1234-1234-123456789abc-extra",
		"not-valid-uuid",
		"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
	}

	sentinelErr := errors.New("invalid")
	for _, id := range invalidUUIDs {
		t.Run("invalid_"+id, func(t *testing.T) {
			err := ValidateUUID(id, "ID", sentinelErr)
			assert.True(t, IsValidation(err), "expected validation error for %s", id)
		})
	}
}

func TestValidateUUIDPreservesSentinel(t *testing.T) {
	sentinel := errors.New("my sentinel")
	err := ValidateUUID("", "field", sentinel)

	assert.ErrorIs(t, err, sentinel)
}

func TestErrorImplementsErrorInterface(t *testing.T) {
	var _ error = NewError(TypeNotFound, "test", nil)
	var _ error = Validation("test", nil)
	var _ error = NotFound("test", nil)
}
