package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"time"
)

// RelationshipRepository defines the operations for persisting  messages.
type RelationshipRepository interface {
	Store(ctx context.Context, relationship types.FriendRelationship) error
}

// --- SQLite Implementation ---

type sqliteRelationshipRepository struct {
	db *sql.DB
}

// NewSQLiteMessageRepository creates a new repository instance.
func NewSQLiteRelationshipRepository(database *DB) (RelationshipRepository, error) {
	if database == nil {
		return nil, errors.New("database connection is required for relationship repo")
	}
	return &sqliteRelationshipRepository{db: database.GetDB()}, nil
}

// Store saves a message, ensuring the conversation exists.
func (r *sqliteRelationshipRepository) Store(ctx context.Context, relationship types.FriendRelationship) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	msgSQL := `
		INSERT INTO relationships (peer_id, status, requested_at, approved_at)
		VALUES (?, ?, ?, ?);
`

	var requestedAtStr, approvedAtStr interface{}
	if !relationship.RequestedAt.IsZero() {
		requestedAtStr = relationship.RequestedAt.Format(time.RFC3339)
	} else {
		requestedAtStr = nil
	}

	if !relationship.ApprovedAt.IsZero() {
		approvedAtStr = relationship.ApprovedAt.Format(time.RFC3339)
	} else {
		approvedAtStr = nil
	}

	_, err = tx.ExecContext(ctx, msgSQL,
		relationship.PeerID,
		relationship.Status,
		requestedAtStr,
		approvedAtStr,
	)

	if err != nil {
		return fmt.Errorf("failed to insert relationship with %s: %w", relationship.PeerID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit message store transaction: %w", err)
	}

	return nil
}
