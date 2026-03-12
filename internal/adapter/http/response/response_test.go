package response

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePaginationDefaults(t *testing.T) {
	q := url.Values{}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationCustomValues(t *testing.T) {
	q := url.Values{
		"page[limit]":  []string{"50"},
		"page[offset]": []string{"100"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 100, result.Offset)
}

func TestParsePaginationLimitExceedsMax(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"200"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 100, result.Limit)
}

func TestParsePaginationNegativeLimit(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"-10"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
}

func TestParsePaginationNegativeOffset(t *testing.T) {
	q := url.Values{
		"page[offset]": []string{"-50"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationInvalidValues(t *testing.T) {
	q := url.Values{
		"page[limit]":  []string{"invalid"},
		"page[offset]": []string{"invalid"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
	assert.Equal(t, 0, result.Offset)
}

func TestParsePaginationZeroLimit(t *testing.T) {
	q := url.Values{
		"page[limit]": []string{"0"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 20, result.Limit)
}

func TestParsePaginationLargeOffset(t *testing.T) {
	q := url.Values{
		"page[offset]": []string{"999999"},
	}
	result := ParsePagination(q, 20, 100)

	assert.Equal(t, 999999, result.Offset)
}

func TestParsePaginationDifferentDefaults(t *testing.T) {
	q := url.Values{}
	result := ParsePagination(q, 50, 200)

	assert.Equal(t, 50, result.Limit)
}
