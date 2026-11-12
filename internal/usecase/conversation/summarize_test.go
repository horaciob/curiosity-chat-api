package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSummaryRepo struct {
	previews map[string]string
	unread   map[string]int
	err      error
}

func (s stubSummaryRepo) LastMessagePreviews(context.Context, []string) (map[string]string, error) {
	return s.previews, s.err
}
func (s stubSummaryRepo) UnreadCountsForUser(context.Context, string, []string) (map[string]int, error) {
	return s.unread, s.err
}

type stubNames map[string]string

func (s stubNames) Names(context.Context, []string) (map[string]string, error) { return s, nil }

func TestSummarizeFillsRowFieldsForRequester(t *testing.T) {
	convs := []*entity.Conversation{
		{ID: "c1", User1ID: "me", User2ID: "ana"},
		{ID: "c2", User1ID: "luis", User2ID: "me"},
	}
	uc := NewSummarize(
		stubSummaryRepo{previews: map[string]string{"c1": "hola"}, unread: map[string]int{"c2": 4}},
		stubNames{"ana": "Ana", "luis": "Luis"},
	)

	uc.Execute(context.Background(), "me", convs)

	require.NotNil(t, convs[0].LastMessagePreview)
	assert.Equal(t, "hola", *convs[0].LastMessagePreview)
	assert.Nil(t, convs[1].LastMessagePreview, "no messages yet")
	assert.Equal(t, 0, convs[0].UnreadCount)
	assert.Equal(t, 4, convs[1].UnreadCount)
	assert.Equal(t, "Ana", convs[0].OtherUserName)
	assert.Equal(t, "Luis", convs[1].OtherUserName)
}

func TestSummarizeIsBestEffort(t *testing.T) {
	convs := []*entity.Conversation{{ID: "c1", User1ID: "me", User2ID: "ana"}}
	uc := NewSummarize(stubSummaryRepo{err: errors.New("db down")}, stubNames{"ana": "Ana"})

	uc.Execute(context.Background(), "me", convs)

	assert.Nil(t, convs[0].LastMessagePreview)
	assert.Equal(t, "Ana", convs[0].OtherUserName, "names still resolve when the DB summary fails")
}
