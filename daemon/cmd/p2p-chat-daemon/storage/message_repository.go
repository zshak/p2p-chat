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

type MessageRepository interface {
	Store(ctx context.Context, msg types.StoredMessage) (id int64, err error)
	StoreGroupMessage(ctx context.Context, msg types.StoredGroupMessage) error
	GetGroupMessages(ctx context.Context, groupID string, limit int, before time.Time) ([]types.StoredGroupMessage, error)
	GetMessagesByPeerID(ctx context.Context, peerID string, limit int) ([]types.StoredMessage, error)
}

type sqliteMessageRepository struct {
	db *sql.DB
}

func NewSQLiteMessageRepository(database *DB) (MessageRepository, error) {
	if database == nil {
		return nil, errors.New("database connection is required for message repository")
	}
	return &sqliteMessageRepository{db: database.GetDB()}, nil
}

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
		log.Printf("WARN: Could not get LastInsertId after message store: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit message store transaction: %w", err)
	}

	return id, nil
}

func (r *sqliteMessageRepository) StoreGroupMessage(ctx context.Context, msg types.StoredGroupMessage) error {
	sqlStmt := `
		INSERT INTO group_messages (group_id, sender_peer_id, content, sent_at)
		VALUES (?, ?, ?, ?);
	`
	sentAtTimestamp := msg.SentAt.Unix()
	if msg.SentAt.IsZero() {
		sentAtTimestamp = time.Now().Unix()
	}

	_, err := r.db.ExecContext(ctx, sqlStmt,
		msg.GroupID,
		msg.SenderPeerID,
		msg.EncryptedContent,
		sentAtTimestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to insert group message for group %s from sender %s: %w", msg.GroupID, msg.SenderPeerID, err)
	}
	log.Printf("Storage: Stored group message for group %s from %s", msg.GroupID, msg.SenderPeerID)
	return nil
}

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
			&encryptedContentBytes,
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

func (r *sqliteMessageRepository) GetMessagesByPeerID(ctx context.Context, peerID string, limit int) ([]types.StoredMessage, error) {
	log.Printf("Storage: Retrieving messages for peer %s", peerID)

	if peerID == "" {
		return nil, errors.New("peerID cannot be empty")
	}

	if limit <= 0 {
		limit = 50
	}

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

		sendTime, err := time.Parse(time.RFC3339, sendTimeStr)
		if err != nil {
			log.Printf("Storage: Error parsing send_time for message %d: %v", msg.ID, err)
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
