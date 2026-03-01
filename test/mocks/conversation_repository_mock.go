package mocks

import (
	"context"
	"time"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

// ConversationRepositoryMock is a mock implementation of conversation.Repository.
type ConversationRepositoryMock struct {
	mock.Mock
}

func (m *ConversationRepositoryMock) Create(ctx context.Context, c *entity.Conversation) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *ConversationRepositoryMock) GetByID(ctx context.Context, id string) (*entity.Conversation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *ConversationRepositoryMock) GetByParticipants(ctx context.Context, userA, userB string) (*entity.Conversation, error) {
	args := m.Called(ctx, userA, userB)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Conversation), args.Error(1)
}

func (m *ConversationRepositoryMock) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Conversation, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Conversation), args.Error(1)
}

func (m *ConversationRepositoryMock) CountByUser(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *ConversationRepositoryMock) UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error {
	args := m.Called(ctx, id, t)
	return args.Error(0)
}
