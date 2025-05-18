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

// RelationshipRepository defines the operations for persisting  messages.
type RelationshipRepository interface {
	Store(ctx context.Context, relationship types.FriendRelationship) error
	UpdateStatus(ctx context.Context, relationship types.FriendRelationship) error
	GetRelationByPeerId(ctx context.Context, peerId string) (types.FriendRelationship, error)
	GetAcceptedRelations(ctx context.Context) ([]types.FriendRelationship, error)
}

// --- SQLite Implementation ---

type sqliteRelationshipRepository struct {
	db *sql.DB
}

// NewSQLiteRelationshipRepository creates a new repository instance.
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

func (r *sqliteRelationshipRepository) UpdateStatus(ctx context.Context, relationship types.FriendRelationship) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	msgSQL := `
		UPDATE relationships 
		SET status = ?,
			approved_at = ?
		
		WHERE peer_id = ?;
`

	approvedAtStr := relationship.ApprovedAt.Format(time.RFC3339)

	_, err = tx.ExecContext(ctx, msgSQL,
		relationship.Status,
		approvedAtStr,
		relationship.PeerID,
	)

	if err != nil {
		return fmt.Errorf("failed to update relationship with %s: %w", relationship.PeerID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit message store transaction: %w", err)
	}

	return nil
}

func (r *sqliteRelationshipRepository) GetRelationByPeerId(ctx context.Context, peerId string) (types.FriendRelationship, error) {
	var rel types.FriendRelationship
	var statusStr string
	var requestedAtStr, approvedAtStr sql.NullString // For reading nullable TEXT dates

	sqlStmt := `SELECT peer_id, status, requested_at, approved_at
                FROM relationships WHERE peer_id = ?;`

	err := r.db.QueryRowContext(ctx, sqlStmt, peerId).Scan(
		&rel.PeerID,
		&statusStr,
		&requestedAtStr,
		&approvedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.FriendRelationship{}, sql.ErrNoRows
		}

		return types.FriendRelationship{}, fmt.Errorf("failed to get relationship for %s: %w", peerId, err)
	}

	rel.Status = stringToFriendStatus(statusStr)

	if requestedAtStr.Valid {
		t, err := time.Parse(time.RFC3339Nano, requestedAtStr.String)
		if err == nil {
			rel.RequestedAt = t
		} else {
			log.Printf("WARN: Could not parse requested_at '%s' for peer %s: %v", requestedAtStr.String, peerId, err)
		}
	}
	if approvedAtStr.Valid {
		t, err := time.Parse(time.RFC3339Nano, approvedAtStr.String)
		if err == nil {
			rel.ApprovedAt = t
		} else {
			log.Printf("WARN: Could not parse approved_at '%s' for peer %s: %v", approvedAtStr.String, peerId, err)
		}
	}

	return rel, nil
}

func (r *sqliteRelationshipRepository) GetAcceptedRelations(ctx context.Context) ([]types.FriendRelationship, error) {
	sqlStmt := `SELECT peer_id, status, requested_at, approved_at
                FROM relationships WHERE Status = ?
				order by peer_id ASC;`

	rows, err := r.db.QueryContext(ctx, sqlStmt, types.FriendStatusApproved)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []types.FriendRelationship{}, sql.ErrNoRows
		}

		return []types.FriendRelationship{}, fmt.Errorf("failed to get relationships %w", err)
	}

	var friends []types.FriendRelationship
	for rows.Next() {
		var rel types.FriendRelationship
		var requestedAtStr, approvedAtStr sql.NullString
		var statusText string

		var errScan error
		errScan = rows.Scan(
			&rel.PeerID,
			&statusText,
			&requestedAtStr,
			&approvedAtStr,
		)
		rel.Status = stringToFriendStatus(statusText)

		t, err := time.Parse(time.RFC3339Nano, requestedAtStr.String)
		if err == nil {
			rel.RequestedAt = t
		} else {
			log.Printf("WARN: Could not parse requested_at '%s' for peer %s: %v", requestedAtStr.String, rel.PeerID, err)
		}

		t, err = time.Parse(time.RFC3339Nano, approvedAtStr.String)
		if err == nil {
			rel.ApprovedAt = t
		} else {
			log.Printf("WARN: Could not parse approved_at '%s' for peer %s: %v", approvedAtStr.String, rel.PeerID, err)
		}

		if errScan != nil {
			log.Printf("Storage: Error scanning approved friend row: %v", errScan)
			return nil, fmt.Errorf("error scanning approved friend row: %w", errScan)
		}

		friends = append(friends, rel)
	}

	return friends, nil
}

func stringToFriendStatus(s string) types.FriendStatus {
	switch s {
	case "1":
		return types.FriendStatusSent
	case "2":
		return types.FriendStatusPending
	case "3":
		return types.FriendStatusApproved
	case "4":
		return types.FriendStatusRejected
	default:
		log.Printf("WARN: Unknown friend status string '%s' from DB, defaulting to None.", s)
		return types.FriendStatusNone // Default or error
	}
}
