package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/middleware"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/adapter/http/response"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/conversation"
)

// ConversationHandler handles conversation endpoints.
type ConversationHandler struct {
	createConversationUC *conversation.CreateConversation
	getConversationUC    *conversation.GetConversation
	listConversationsUC  *conversation.ListConversations
}

// NewConversationHandler creates a new ConversationHandler.
func NewConversationHandler(
	createConversationUC *conversation.CreateConversation,
	getConversationUC *conversation.GetConversation,
	listConversationsUC *conversation.ListConversations,
) *ConversationHandler {
	return &ConversationHandler{
		createConversationUC: createConversationUC,
		getConversationUC:    getConversationUC,
		listConversationsUC:  listConversationsUC,
	}
}

type createConversationRequest struct {
	TargetUserID string `json:"target_user_id"`
}

// Create handles POST /api/v1/conversations.
//
//	@Summary		Create or get a conversation
//	@Description	Starts a new conversation between the authenticated user and the target user. If a conversation already exists it is returned. Both users must mutually follow each other.
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createConversationRequest	true	"Target user"
//	@Success		201		{object}	object						"Conversation created"
//	@Success		200		{object}	object						"Conversation already exists"
//	@Failure		400		{object}	object						"Bad request"
//	@Failure		401		{object}	object						"Unauthorized"
//	@Failure		403		{object}	object						"Users do not mutually follow each other"
//	@Failure		500		{object}	object						"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations [post]
func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.UserIDFromContext(r.Context())

	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	conv, err := h.createConversationUC.Execute(r.Context(), requesterID, req.TargetUserID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response.Created(w, response.NewConversationResponse(conv))
}

// Get handles GET /api/v1/conversations/{id}.
//
//	@Summary		Get a conversation
//	@Description	Returns a conversation by ID. The authenticated user must be a participant.
//	@Tags			conversations
//	@Produce		json
//	@Param			id	path		string	true	"Conversation ID"
//	@Success		200	{object}	object	"Conversation"
//	@Failure		401	{object}	object	"Unauthorized"
//	@Failure		403	{object}	object	"Forbidden — not a participant"
//	@Failure		404	{object}	object	"Not found"
//	@Failure		500	{object}	object	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations/{id} [get]
func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	requesterID := middleware.UserIDFromContext(r.Context())

	conv, err := h.getConversationUC.Execute(r.Context(), id, requesterID)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response.Success(w, response.NewConversationResponse(conv))
}

// List handles GET /api/v1/conversations.
//
//	@Summary		List conversations
//	@Description	Returns a paginated list of conversations for the authenticated user, ordered by most recent activity.
//	@Tags			conversations
//	@Produce		json
//	@Param			page[limit]		query		int		false	"Max results (default 20)"
//	@Param			page[offset]	query		int		false	"Offset for pagination"
//	@Success		200				{object}	object	"Paginated list of conversations"
//	@Failure		401				{object}	object	"Unauthorized"
//	@Failure		500				{object}	object	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations [get]
func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.UserIDFromContext(r.Context())
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("page[limit]"))
	offset, _ := strconv.Atoi(q.Get("page[offset]"))
	if limit == 0 {
		limit = 20
	}

	convs, total, err := h.listConversationsUC.Execute(r.Context(), requesterID, limit, offset)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response.Collection(w, response.NewConversationListResponse(convs), total, limit, offset, "/api/v1/conversations")
}
