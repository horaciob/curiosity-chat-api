package handler

import (
	"context"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/infrastructure/logger"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/usecase/message"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/ws"
	"go.uber.org/zap"
)

// deliverNewMessage fans a saved message out, the same way whether it arrived over HTTP
// or the WebSocket: the sender's other devices, the recipient, a push when they have no
// open connection, the "delivered" receipt when they do, and their unread counters. The
// WebSocket path used to skip the counters and drop share_intent.
func deliverNewMessage(hub *ws.Hub, pusher ChatPusher, convRepo message.ConversationRepository, msgRepo message.Repository, senderID string, msg *entity.Message) {
	// The request context ends once the response is written; this outlives it.
	ctx, cancel := context.WithTimeout(context.Background(), wsOperationTimeout)
	defer cancel()

	outgoing := ws.OutgoingMessage{
		ID:             msg.ID,
		Type:           msg.Type,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		Content:        msg.Content,
		POIID:          msg.POIID,
		ShareIntent:    msg.ShareIntent,
		Status:         msg.Status,
		CreatedAt:      msg.CreatedAt,
	}
	hub.BroadcastTo(senderID, outgoing) // multi-device echo

	conv, err := convRepo.GetByID(ctx, msg.ConversationID)
	if err != nil {
		logger.Warn("deliver: conversation not found", zap.String("conversation_id", msg.ConversationID), zap.Error(err))
		return
	}
	recipient, err := conv.OtherUserID(senderID)
	if err != nil {
		logger.Warn("deliver: no other participant", zap.String("conversation_id", msg.ConversationID), zap.Error(err))
		return
	}
	hub.BroadcastTo(recipient, outgoing)
	pushIfOffline(hub, pusher, recipient, msg)

	if hub.IsOnline(recipient) {
		if err := msgRepo.UpdateStatus(ctx, msg.ID, "delivered"); err == nil {
			hub.BroadcastJSON(senderID, ws.DeliveredEvent{Type: "delivered", ConversationID: msg.ConversationID, MessageID: msg.ID})
		}
	}

	if convUnread, err := msgRepo.CountUnreadByConversationForUser(ctx, msg.ConversationID, recipient); err == nil {
		total, _ := msgRepo.CountTotalUnreadForUser(ctx, recipient)
		hub.BroadcastJSON(recipient, ws.UnreadCountEvent{
			Type: "unread_count_update", ConversationID: msg.ConversationID, UnreadCount: convUnread, TotalUnread: total,
		})
	}
}
