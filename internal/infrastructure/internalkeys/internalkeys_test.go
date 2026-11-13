package internalkeys

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNamesEveryCaller(t *testing.T) {
	s, err := Parse("legacy-key", " chat-api:k1 , poi-consumer:k2 ,")
	require.NoError(t, err)

	assert.Equal(t, []string{LegacyCaller, "chat-api", "poi-consumer"}, s.Callers())
	for key, want := range map[string]string{"legacy-key": LegacyCaller, "k1": "chat-api", "k2": "poi-consumer"} {
		got, ok := s.Match(key)
		assert.True(t, ok, key)
		assert.Equal(t, want, got)
	}
}

func TestUnknownAndEmptyKeysDoNotMatch(t *testing.T) {
	s, _ := Parse("legacy", "chat-api:k1")

	for _, provided := range []string{"", "k", "k10", "LEGACY"} {
		_, ok := s.Match(provided)
		assert.False(t, ok, provided)
	}
}

func TestACallerWithAnUnsetKeyIsDisabled(t *testing.T) {
	s, err := Parse("", "chat-api:,poi-consumer:k2")
	require.NoError(t, err)

	assert.Equal(t, []string{"poi-consumer"}, s.Callers())
	_, ok := s.Match("")
	assert.False(t, ok)
}

func TestParseRejectsMalformedEntriesWithoutLeakingTheKey(t *testing.T) {
	_, err := Parse("", "supersecretvalue")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "supersecretvalue")

	_, err = Parse("", ":k1")
	assert.Error(t, err)
}

func TestEmptySetsAcceptNothing(t *testing.T) {
	var nilSet *Set
	for _, s := range []*Set{nilSet, Single(""), {}} {
		assert.True(t, s.Empty())
		_, ok := s.Match("anything")
		assert.False(t, ok)
	}
}

func TestCallerTravelsInTheContext(t *testing.T) {
	_, ok := CallerFromContext(context.Background())
	assert.False(t, ok)

	got, ok := CallerFromContext(WithCaller(context.Background(), "chat-api"))
	assert.True(t, ok)
	assert.Equal(t, "chat-api", got)
}
