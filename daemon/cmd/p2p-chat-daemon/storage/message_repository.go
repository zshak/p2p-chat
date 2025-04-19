package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
)

// MessageRepository defines the operations for persisting chat messages.
type MessageRepository interface {
	// Store saves a new message to the database.
	Store(ctx context.Context, msg types.ChatMessage) (id int64, err error)
}

// --- SQLite Implementation ---

type sqliteMessageRepository struct {
	db *sql.DB
}

// NewSQLiteMessageRepository creates a new repository instance.
func NewSQLiteMessageRepository(database *DB) (MessageRepository, error) {
	if database == nil {
		return nil, errors.New("database connection is required for message repository")
	}
	return &sqliteMessageRepository{db: database.GetDB()}, nil
}

// Store saves a message, ensuring the conversation exists.
func (r *sqliteMessageRepository) Store(ctx context.Context, msg types.ChatMessage) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	msgSQL := `
		INSERT INTO messages (sender_peer_id, recipient_peer_id, send_time, content, is_outgoing)
		VALUES (?, ?, ?, ?, ?);
`
	res, err := tx.ExecContext(ctx, msgSQL,
		msg.RecipientPeerId,
		msg.SenderPeerID,
		msg.SendTime,
		msg.Content,
		msg.IsOutgoing,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to insert message for conversation %s: %w", msg.RecipientPeerId, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		// This driver might not support LastInsertId well, or table lacks AUTOINCREMENT? Check schema.
		log.Printf("WARN: Could not get LastInsertId after message store: %v", err)
		// Still commit, but return 0 for ID
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit message store transaction: %w", err)
	}

	return id, nil
}
