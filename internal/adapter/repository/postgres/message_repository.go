package postgres

import (
	"context"
	"time"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	"github.com/jmoiron/sqlx"
)

// MessageRepository implements message.Repository.
type MessageRepository struct {
	db *sqlx.DB
}

// NewMessageRepository creates a new MessageRepository.
func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

type messageRow struct {
	ID             string    `db:"id"`
	ConversationID string    `db:"conversation_id"`
	SenderID       string    `db:"sender_id"`
	Type           string    `db:"type"`
	Content        *string   `db:"content"`
	POIID          *string   `db:"poi_id"`
	CreatedAt      time.Time `db:"created_at"`
}

func (r messageRow) toEntity() *entity.Message {
	return &entity.Message{
		ID:             r.ID,
		ConversationID: r.ConversationID,
		SenderID:       r.SenderID,
		Type:           r.Type,
		Content:        r.Content,
		POIID:          r.POIID,
		CreatedAt:      r.CreatedAt,
	}
}

func (r *MessageRepository) Create(ctx context.Context, m *entity.Message) error {
	q := `INSERT INTO messages (id, conversation_id, sender_id, type, content, poi_id, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, q, m.ID, m.ConversationID, m.SenderID, m.Type, m.Content, m.POIID, m.CreatedAt)
	return err
}

func (r *MessageRepository) ListByConversation(ctx context.Context, conversationID string, limit, offset int) ([]*entity.Message, error) {
	q := `SELECT id, conversation_id, sender_id, type, content, poi_id, created_at
	      FROM messages
	      WHERE conversation_id = $1
	      ORDER BY created_at DESC
	      LIMIT $2 OFFSET $3`
	var rows []messageRow
	if err := r.db.SelectContext(ctx, &rows, q, conversationID, limit, offset); err != nil {
		return nil, err
	}
	msgs := make([]*entity.Message, 0, len(rows))
	for _, row := range rows {
		msgs = append(msgs, row.toEntity())
	}
	return msgs, nil
}

func (r *MessageRepository) CountByConversation(ctx context.Context, conversationID string) (int, error) {
	q := `SELECT COUNT(*) FROM messages WHERE conversation_id = $1`
	var count int
	if err := r.db.GetContext(ctx, &count, q, conversationID); err != nil {
		return 0, err
	}
	return count, nil
}
