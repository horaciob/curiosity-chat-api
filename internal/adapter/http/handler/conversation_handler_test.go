package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/conversation"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupConversationHandler() (*ConversationHandler, *mocks.ConversationRepositoryMock) {
	repo := new(mocks.ConversationRepositoryMock)
	createUC := conversation.NewCreateConversation(repo)
	getUC := conversation.NewGetConversation(repo)
	listUC := conversation.NewListConversations(repo)
	return NewConversationHandler(createUC, getUC, listUC), repo
}

func addUserIDToContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	return req.WithContext(ctx)
}

func TestConversationHandlerCreateSuccess(t *testing.T) {
	handler, repo := setupConversationHandler()

	userA := uuid.New().String()
	userB := uuid.New().String()

	repo.On("GetByParticipants", mock.Anything, userA, userB).Return(nil, apperror.NotFound("not found", nil))
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	body := map[string]string{"target_user_id": userB}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, userA)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	repo.AssertExpectations(t)
}

func TestConversationHandlerCreateInvalidBody(t *testing.T) {
	handler, _ := setupConversationHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader([]byte("invalid json")))
	req = addUserIDToContext(req, uuid.New().String())
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestConversationHandlerCreateValidationError(t *testing.T) {
	handler, _ := setupConversationHandler()

	body := map[string]string{"target_user_id": "not-a-uuid"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, uuid.New().String())
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestConversationHandlerCreateNotFound(t *testing.T) {
	handler, repo := setupConversationHandler()

	userA := uuid.New().String()
	userB := uuid.New().String()

	repo.On("GetByParticipants", mock.Anything, userA, userB).Return(nil, apperror.NotFound("not found", nil))
	repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Conversation")).Return(nil)

	body := map[string]string{"target_user_id": userB}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, userA)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestConversationHandlerGetSuccess(t *testing.T) {
	handler, repo := setupConversationHandler()

	userA := uuid.New().String()
	userB := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	repo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID, nil)
	req = addUserIDToContext(req, userA)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Get(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestConversationHandlerGetNotFound(t *testing.T) {
	handler, repo := setupConversationHandler()

	convID := uuid.New().String()
	userID := uuid.New().String()

	repo.On("GetByID", mock.Anything, convID).Return(nil, apperror.NotFound("not found", nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+convID, nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", convID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Get(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestConversationHandlerGetForbidden(t *testing.T) {
	handler, repo := setupConversationHandler()

	userA := uuid.New().String()
	userB := uuid.New().String()
	outsider := uuid.New().String()
	conv := entity.NewConversation(userA, userB)

	repo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID, nil)
	req = addUserIDToContext(req, outsider)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Get(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestConversationHandlerListSuccess(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()
	convs := []*entity.Conversation{
		entity.NewConversation(userID, uuid.New().String()),
		entity.NewConversation(userID, uuid.New().String()),
	}

	repo.On("ListByUser", mock.Anything, userID, 20, 0).Return(convs, nil)
	repo.On("CountByUser", mock.Anything, userID).Return(2, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestConversationHandlerListWithPagination(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()

	repo.On("ListByUser", mock.Anything, userID, 10, 20).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", mock.Anything, userID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?page[limit]=10&page[offset]=20", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	repo.AssertExpectations(t)
}

func TestConversationHandlerListInternalError(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()

	repo.On("ListByUser", mock.Anything, userID, 20, 0).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestConversationHandlerListNegativeLimit(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()

	repo.On("ListByUser", mock.Anything, userID, 20, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", mock.Anything, userID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?page[limit]=-5", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestConversationHandlerListLimitExceedsMax(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()

	repo.On("ListByUser", mock.Anything, userID, 100, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", mock.Anything, userID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?page[limit]=500", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestConversationHandlerListNegativeOffset(t *testing.T) {
	handler, repo := setupConversationHandler()

	userID := uuid.New().String()

	repo.On("ListByUser", mock.Anything, userID, 20, 0).Return([]*entity.Conversation{}, nil)
	repo.On("CountByUser", mock.Anything, userID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations?page[offset]=-10", nil)
	req = addUserIDToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
