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
	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"github.com/horaciobranciforte/curiosity-chat-api/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupMessageHandler() (*MessageHandler, *mocks.MessageRepositoryMock, *mocks.ConversationRepositoryMock, *mocks.FollowCheckerMock) {
	msgRepo := new(mocks.MessageRepositoryMock)
	convRepo := new(mocks.ConversationRepositoryMock)
	followChecker := new(mocks.FollowCheckerMock)
	sendUC := message.NewSendMessage(msgRepo, convRepo)
	getUC := message.NewGetMessages(msgRepo, convRepo)
	hub := ws.NewHub()
	go hub.Run()
	return NewMessageHandler(sendUC, getUC, hub, convRepo, msgRepo), msgRepo, convRepo, followChecker
}

func TestMessageHandlerSendTextSuccess(t *testing.T) {
	handler, msgRepo, convRepo, followChecker := setupMessageHandler()

	senderID := uuid.New().String()
	conv := entity.NewConversation(senderID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", mock.Anything, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", mock.Anything, conv.ID, mock.Anything).Return(nil)
	msgRepo.On("CountUnreadByConversationForUser", mock.Anything, conv.ID, mock.Anything).Return(1, nil)
	msgRepo.On("CountTotalUnreadForUser", mock.Anything, mock.Anything).Return(1, nil)

	body := map[string]string{"type": "text", "content": "Hello!"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID+"/messages", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, senderID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	msgRepo.AssertExpectations(t)
	convRepo.AssertExpectations(t)
}

func TestMessageHandlerSendPOIShareSuccess(t *testing.T) {
	handler, msgRepo, convRepo, followChecker := setupMessageHandler()

	senderID := uuid.New().String()
	poiID := uuid.New().String()
	conv := entity.NewConversation(senderID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", mock.Anything, senderID, mock.Anything).Return(true, nil)
	msgRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Message")).Return(nil)
	convRepo.On("UpdateLastMessageAt", mock.Anything, conv.ID, mock.Anything).Return(nil)
	msgRepo.On("CountUnreadByConversationForUser", mock.Anything, conv.ID, mock.Anything).Return(1, nil)
	msgRepo.On("CountTotalUnreadForUser", mock.Anything, mock.Anything).Return(1, nil)

	body := map[string]string{"type": "poi_share", "poi_id": poiID}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID+"/messages", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, senderID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func TestMessageHandlerSendInvalidBody(t *testing.T) {
	handler, _, _, _ := setupMessageHandler()

	convID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+convID+"/messages", bytes.NewReader([]byte("invalid json")))
	req = addUserIDToContext(req, uuid.New().String())
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", convID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestMessageHandlerSendConversationNotFound(t *testing.T) {
	handler, _, convRepo, _ := setupMessageHandler()

	convID := uuid.New().String()
	senderID := uuid.New().String()

	convRepo.On("GetByID", mock.Anything, convID).Return(nil, apperror.NotFound("not found", nil))

	body := map[string]string{"type": "text", "content": "Hello!"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+convID+"/messages", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, senderID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", convID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestMessageHandlerSendNotParticipant(t *testing.T) {
	handler, _, convRepo, _ := setupMessageHandler()

	conv := entity.NewConversation(uuid.New().String(), uuid.New().String())
	outsider := uuid.New().String()

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	body := map[string]string{"type": "text", "content": "Hello!"}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID+"/messages", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, outsider)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestMessageHandlerSendValidationError(t *testing.T) {
	handler, _, convRepo, followChecker := setupMessageHandler()

	senderID := uuid.New().String()
	conv := entity.NewConversation(senderID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	followChecker.On("AreFollowing", mock.Anything, senderID, mock.Anything).Return(true, nil)

	body := map[string]string{"type": "text", "content": ""}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+conv.ID+"/messages", bytes.NewReader(jsonBody))
	req = addUserIDToContext(req, senderID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.Send(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestMessageHandlerListSuccess(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())
	msgs := []*entity.Message{
		entity.NewTextMessage(conv.ID, requesterID, "hello"),
		entity.NewTextMessage(conv.ID, requesterID, "world"),
	}

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 50, 0).Return(msgs, nil)
	msgRepo.On("CountByConversation", mock.Anything, conv.ID).Return(2, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMessageHandlerListWithPagination(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 20, 10).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", mock.Anything, conv.ID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages?page[limit]=20&page[offset]=10", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMessageHandlerListNotParticipant(t *testing.T) {
	handler, _, convRepo, _ := setupMessageHandler()

	conv := entity.NewConversation(uuid.New().String(), uuid.New().String())
	outsider := uuid.New().String()

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages", nil)
	req = addUserIDToContext(req, outsider)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestMessageHandlerListInternalError(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 50, 0).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestMessageHandlerListNegativeLimit(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", mock.Anything, conv.ID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages?page[limit]=-5", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMessageHandlerListLimitExceedsMax(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 100, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", mock.Anything, conv.ID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages?page[limit]=500", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMessageHandlerListNegativeOffset(t *testing.T) {
	handler, msgRepo, convRepo, _ := setupMessageHandler()

	requesterID := uuid.New().String()
	conv := entity.NewConversation(requesterID, uuid.New().String())

	convRepo.On("GetByID", mock.Anything, conv.ID).Return(conv, nil)
	msgRepo.On("ListByConversation", mock.Anything, conv.ID, 50, 0).Return([]*entity.Message{}, nil)
	msgRepo.On("CountByConversation", mock.Anything, conv.ID).Return(0, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+conv.ID+"/messages?page[offset]=-10", nil)
	req = addUserIDToContext(req, requesterID)
	rr := httptest.NewRecorder()

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", conv.ID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))

	handler.List(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
