package mocks

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

// MessageRepositoryMock is a mock implementation of message.Repository.
type MessageRepositoryMock struct {
	mock.Mock
}

func (m *MessageRepositoryMock) Create(ctx context.Context, msg *entity.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MessageRepositoryMock) ListByConversation(ctx context.Context, conversationID string, limit, offset int) ([]*entity.Message, error) {
	args := m.Called(ctx, conversationID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Message), args.Error(1)
}

func (m *MessageRepositoryMock) CountByConversation(ctx context.Context, conversationID string) (int, error) {
	args := m.Called(ctx, conversationID)
	return args.Int(0), args.Error(1)
}
