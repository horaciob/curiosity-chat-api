package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/horaciobranciforte/curiosity-chat-api/internal/domain/entity"
	domerrors "github.com/horaciobranciforte/curiosity-chat-api/internal/domain/errors"
	"github.com/horaciobranciforte/curiosity-chat-api/internal/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

// ConversationRepository implements conversation.Repository and message.ConversationRepository.
type ConversationRepository struct {
	db *sqlx.DB
}

// NewConversationRepository creates a new ConversationRepository.
func NewConversationRepository(db *sqlx.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

type conversationRow struct {
	ID            string     `db:"id"`
	User1ID       string     `db:"user1_id"`
	User2ID       string     `db:"user2_id"`
	CreatedAt     time.Time  `db:"created_at"`
	LastMessageAt *time.Time `db:"last_message_at"`
}

func (r conversationRow) toEntity() *entity.Conversation {
	return &entity.Conversation{
		ID:            r.ID,
		User1ID:       r.User1ID,
		User2ID:       r.User2ID,
		CreatedAt:     r.CreatedAt,
		LastMessageAt: r.LastMessageAt,
	}
}

func (r *ConversationRepository) Create(ctx context.Context, c *entity.Conversation) error {
	q := `INSERT INTO conversations (id, user1_id, user2_id, created_at)
	      VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, c.ID, c.User1ID, c.User2ID, c.CreatedAt)
	return err
}

func (r *ConversationRepository) GetByID(ctx context.Context, id string) (*entity.Conversation, error) {
	q := `SELECT id, user1_id, user2_id, created_at, last_message_at
	      FROM conversations WHERE id = $1`
	var row conversationRow
	if err := r.db.GetContext(ctx, &row, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("conversation not found", domerrors.ErrConversationNotFound)
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *ConversationRepository) GetByParticipants(ctx context.Context, userA, userB string) (*entity.Conversation, error) {
	// Normalize so user1 < user2
	u1, u2 := userA, userB
	if u1 > u2 {
		u1, u2 = u2, u1
	}
	q := `SELECT id, user1_id, user2_id, created_at, last_message_at
	      FROM conversations WHERE user1_id = $1 AND user2_id = $2`
	var row conversationRow
	if err := r.db.GetContext(ctx, &row, q, u1, u2); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("conversation not found", domerrors.ErrConversationNotFound)
		}
		return nil, err
	}
	return row.toEntity(), nil
}

func (r *ConversationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*entity.Conversation, error) {
	q := `SELECT id, user1_id, user2_id, created_at, last_message_at
	      FROM conversations
	      WHERE user1_id = $1 OR user2_id = $1
	      ORDER BY COALESCE(last_message_at, created_at) DESC
	      LIMIT $2 OFFSET $3`
	var rows []conversationRow
	if err := r.db.SelectContext(ctx, &rows, q, userID, limit, offset); err != nil {
		return nil, err
	}
	convs := make([]*entity.Conversation, 0, len(rows))
	for _, row := range rows {
		convs = append(convs, row.toEntity())
	}
	return convs, nil
}

func (r *ConversationRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	q := `SELECT COUNT(*) FROM conversations WHERE user1_id = $1 OR user2_id = $1`
	var count int
	if err := r.db.GetContext(ctx, &count, q, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ConversationRepository) UpdateLastMessageAt(ctx context.Context, id string, t time.Time) error {
	q := `UPDATE conversations SET last_message_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, t, id)
	return err
}
