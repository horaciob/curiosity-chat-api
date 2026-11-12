package conversation

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// SummaryRepository provides per-conversation list data.
type SummaryRepository interface {
	LastMessagePreviews(ctx context.Context, conversationIDs []string) (map[string]string, error)
	UnreadCountsForUser(ctx context.Context, userID string, conversationIDs []string) (map[string]int, error)
}

// NameResolver resolves user display names (user-api).
type NameResolver interface {
	Names(ctx context.Context, userIDs []string) (map[string]string, error)
}

// Summarize fills the fields a conversation list row shows: the other person's name, the
// last message and the unread count. The app used to fetch each name separately (showing
// raw IDs meanwhile) and had no preview or per-chat unread count at all.
//
// Best effort: a failure in one source leaves those fields empty rather than failing the
// whole list.
type Summarize struct {
	repo  SummaryRepository
	names NameResolver // optional
}

// NewSummarize creates the use case. names may be nil.
func NewSummarize(repo SummaryRepository, names NameResolver) *Summarize {
	return &Summarize{repo: repo, names: names}
}

// Execute fills the summary fields of convs in place, relative to userID.
func (uc *Summarize) Execute(ctx context.Context, userID string, convs []*entity.Conversation) {
	if len(convs) == 0 {
		return
	}
	ids := make([]string, 0, len(convs))
	otherIDs := make([]string, 0, len(convs))
	for _, c := range convs {
		ids = append(ids, c.ID)
		if other, err := c.OtherUserID(userID); err == nil {
			otherIDs = append(otherIDs, other)
		}
	}

	if previews, err := uc.repo.LastMessagePreviews(ctx, ids); err == nil {
		for _, c := range convs {
			if p, ok := previews[c.ID]; ok {
				preview := p
				c.LastMessagePreview = &preview
			}
		}
	} else {
		logger.Warn("conversation summary: previews failed", zap.Error(err))
	}

	if counts, err := uc.repo.UnreadCountsForUser(ctx, userID, ids); err == nil {
		for _, c := range convs {
			c.UnreadCount = counts[c.ID]
		}
	} else {
		logger.Warn("conversation summary: unread counts failed", zap.Error(err))
	}

	if uc.names == nil {
		return
	}
	names, err := uc.names.Names(ctx, otherIDs)
	if err != nil {
		logger.Warn("conversation summary: names failed", zap.Error(err))
		return
	}
	for _, c := range convs {
		if other, err := c.OtherUserID(userID); err == nil {
			c.OtherUserName = names[other]
		}
	}
}
