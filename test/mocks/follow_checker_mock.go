package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// FollowCheckerMock is a mock implementation of conversation.FollowChecker.
type FollowCheckerMock struct {
	mock.Mock
}

func (m *FollowCheckerMock) AreFollowing(ctx context.Context, userA, userB string) (bool, error) {
	args := m.Called(ctx, userA, userB)
	return args.Bool(0), args.Error(1)
}
