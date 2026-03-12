package apperror

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
	}

	sentinelErr := errors.New("invalid")
	for _, id := range invalidUUIDs {
		err := ValidateUUID(id, "ID", sentinelErr)
		assert.True(t, IsValidation(err), "expected validation error for %s", id)
	}
}
