package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"time"
)

// MessageRepository defines the operations for persisting chat messages.
type MessageRepository interface {
	// Store saves a new message to the database.
	Store(ctx context.Context, msg types.StoredMessage) (id int64, err error)

	StoreGroupMessage(ctx context.Context, msg types.StoredGroupMessage) error

	GetGroupMessages(ctx context.Context, groupID string, limit int, before time.Time) ([]types.StoredGroupMessage, error)

	// GetMessagesByPeerID retrieves messages exchanged with a specific peer, ordered by timestamp
	GetMessagesByPeerID(ctx context.Context, peerID string, limit int) ([]types.StoredMessage, error)
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
func (r *sqliteMessageRepository) Store(ctx context.Context, msg types.StoredMessage) (int64, error) {
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
		msg.SenderPeerID,
		msg.RecipientPeerId,
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

// StoreGroupMessage saves a new group message to the group_messages table.
func (r *sqliteMessageRepository) StoreGroupMessage(ctx context.Context, msg types.StoredGroupMessage) error {
	sqlStmt := `
		INSERT INTO group_messages (group_id, sender_peer_id, content, sent_at)
		VALUES (?, ?, ?, ?);
	`
	// Ensure SentAt is not zero, default to Now() if it is.
	sentAtTimestamp := msg.SentAt.Unix()
	if msg.SentAt.IsZero() {
		sentAtTimestamp = time.Now().Unix()
	}

	// Assuming EncryptedContent is []byte and DB column `content` is BLOB
	_, err := r.db.ExecContext(ctx, sqlStmt,
		msg.GroupID,
		msg.SenderPeerID,
		msg.EncryptedContent, // Store raw bytes if column is BLOB
		sentAtTimestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to insert group message for group %s from sender %s: %w", msg.GroupID, msg.SenderPeerID, err)
	}
	log.Printf("Storage: Stored group message for group %s from %s", msg.GroupID, msg.SenderPeerID)
	return nil
}

// GetGroupMessages retrieves recent messages for a specific group.
// Returns messages ordered by most recent first.
func (r *sqliteMessageRepository) GetGroupMessages(ctx context.Context, groupID string, limit int, before time.Time) ([]types.StoredGroupMessage, error) {
	if limit <= 0 {
		limit = 50
	}
	beforeTimestamp := before.Unix()
	if before.IsZero() {
		beforeTimestamp = time.Now().Add(100 * 365 * 24 * time.Hour).Unix()
	}

	querySQL := `
		SELECT group_id, sender_peer_id, content, sent_at
		FROM group_messages
		WHERE group_id = ? AND sent_at < ?
		ORDER BY sent_at DESC
		LIMIT ?;
	`
	rows, err := r.db.QueryContext(ctx, querySQL, groupID, beforeTimestamp, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query group messages for %s: %w", groupID, err)
	}
	defer rows.Close()

	var messages []types.StoredGroupMessage
	for rows.Next() {
		var msg types.StoredGroupMessage
		var sentAtUnix int64

		var encryptedContentBytes []byte

		err := rows.Scan(
			&msg.GroupID,
			&msg.SenderPeerID,
			&encryptedContentBytes, // Scan into byte slice if DB column is BLOB
			&sentAtUnix,
		)
		if err != nil {
			log.Printf("Storage: Error scanning group message row for group %s: %v", groupID, err)
			continue
		}

		msg.EncryptedContent = encryptedContentBytes
		msg.SentAt = time.Unix(sentAtUnix, 0)

		messages = append([]types.StoredGroupMessage{msg}, messages...)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating group message rows for %s: %w", groupID, err)
	}

	log.Printf("Storage: Retrieved %d messages for group %s", len(messages), groupID)
	return messages, nil
}

// GetMessagesByPeerID retrieves messages exchanged with a specific peer.
// Returns messages ordered by timestamp ascending (oldest first).
func (r *sqliteMessageRepository) GetMessagesByPeerID(ctx context.Context, peerID string, limit int) ([]types.StoredMessage, error) {
	log.Printf("Storage: Retrieving messages for peer %s", peerID)

	if peerID == "" {
		return nil, errors.New("peerID cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}

	// This query retrieves messages where:
	// 1. The specified peer is either the sender or recipient
	// 2. Orders by send_time in ascending order (oldest first)
	// 3. Limits the number of results
	querySQL := `
		SELECT id, sender_peer_id, recipient_peer_id, send_time, content, is_outgoing
		FROM messages
		WHERE (recipient_peer_id = ? OR sender_peer_id = ?)
		ORDER BY send_time ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, querySQL, peerID, peerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages for peer %s: %w", peerID, err)
	}
	defer rows.Close()

	var messages []types.StoredMessage
	for rows.Next() {
		var msg types.StoredMessage
		var sendTimeStr string

		err := rows.Scan(
			&msg.ID,
			&msg.SenderPeerID,
			&msg.RecipientPeerId,
			&sendTimeStr,
			&msg.Content,
			&msg.IsOutgoing,
		)
		if err != nil {
			log.Printf("Storage: Error scanning message row for peer %s: %v", peerID, err)
			continue
		}

		// Parse the send_time string into a time.Time
		sendTime, err := time.Parse(time.RFC3339, sendTimeStr)
		if err != nil {
			log.Printf("Storage: Error parsing send_time for message %d: %v", msg.ID, err)
			// Use current time as a fallback
			sendTime = time.Now()
		}
		msg.SendTime = sendTime

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows for peer %s: %w", peerID, err)
	}

	log.Printf("Storage: Retrieved %d messages for peer %s", len(messages), peerID)
	return messages, nil
}
