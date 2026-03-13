package handler

import (
	"encoding/json"
	"net/http"

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
//	@Description	Starts a new conversation between the authenticated user and the target user. If a conversation already exists it is returned. Both users must mutually follow each other. Returns JSON:API compliant response.
//	@Tags			conversations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		createConversationRequest	true	"Target user"
//	@Success		201		{object}	ConversationResponse	"Conversation created"
//	@Success		200		{object}	ConversationResponse	"Conversation already exists"
//	@Failure		400		{object}	JSONAPIErrorsResponse	"Bad request"
//	@Failure		401		{object}	JSONAPIErrorsResponse	"Unauthorized"
//	@Failure		403		{object}	JSONAPIErrorsResponse	"Users do not mutually follow each other"
//	@Failure		500		{object}	JSONAPIErrorsResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations [post]
func (h *ConversationHandler) Create(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.UserIDFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, response.MaxRequestBodySize)
	defer r.Body.Close()
	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			response.Error(w, http.StatusRequestEntityTooLarge, "Request Entity Too Large", "request body exceeds 1MB limit")
			return
		}
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
//	@Description	Returns a conversation by ID. The authenticated user must be a participant. Returns JSON:API compliant response.
//	@Tags			conversations
//	@Produce		json
//	@Param			id	path		string	true	"Conversation ID"
//	@Success		200	{object}	ConversationResponse	"Conversation"
//	@Failure		401	{object}	JSONAPIErrorsResponse	"Unauthorized"
//	@Failure		403	{object}	JSONAPIErrorsResponse	"Forbidden — not a participant"
//	@Failure		404	{object}	JSONAPIErrorsResponse	"Not found"
//	@Failure		500	{object}	JSONAPIErrorsResponse	"Internal server error"
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
//	@Description	Returns a paginated list of conversations for the authenticated user, ordered by most recent activity. Returns JSON:API compliant response with meta and links for pagination.
//	@Tags			conversations
//	@Produce		json
//	@Param			page[limit]		query		int		false	"Max results (default 20)"
//	@Param			page[offset]	query		int		false	"Offset for pagination"
//	@Success		200				{object}	ConversationListResponse	"Paginated list of conversations"
//	@Failure		401				{object}	JSONAPIErrorsResponse	"Unauthorized"
//	@Failure		500				{object}	JSONAPIErrorsResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/conversations [get]
func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	requesterID := middleware.UserIDFromContext(r.Context())
	pagination := response.ParsePagination(r.URL.Query(), conversation.DefaultConversationLimit, conversation.MaxConversationLimit)

	convs, total, err := h.listConversationsUC.Execute(r.Context(), requesterID, pagination.Limit, pagination.Offset)
	if err != nil {
		handleUseCaseError(w, err)
		return
	}

	response.Collection(w, response.NewConversationListResponse(convs), total, pagination.Limit, pagination.Offset, "/api/v1/conversations")
}
