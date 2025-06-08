package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// GroupMemberRepository defines operations for managing group memberships.
type GroupMemberRepository interface {
	AddMembers(ctx context.Context, groupID string, peerIDs []string) error // New method
}

// --- SQLite Implementation ---

type sqliteGroupMemberRepository struct {
	db *sql.DB
}

// NewSQLiteGroupMemberRepository creates a new group membership repository.
func NewSQLiteGroupMemberRepository(database *DB) (GroupMemberRepository, error) {
	if database == nil || database.GetDB() == nil {
		return nil, errors.New("database connection required for group member repository")
	}
	return &sqliteGroupMemberRepository{db: database.GetDB()}, nil
}

// AddMembers adds multiple peers to a group within a single transaction.
// Uses INSERT OR IGNORE to be idempotent for each member.
func (r *sqliteGroupMemberRepository) AddMembers(ctx context.Context, groupID string, peerIDs []string) error {
	if groupID == "" {
		return errors.New("groupID cannot be empty")
	}
	if len(peerIDs) == 0 {
		return nil // Nothing to add
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for adding members to group %s: %w", groupID, err)
	}
	// Defer rollback in case of error, commit explicitly on success
	defer tx.Rollback()

	// Prepare the statement for inserting members
	// Using INSERT OR IGNORE to avoid errors if a member already exists in the group.
	// This makes the operation idempotent for individual members.
	stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO group_members (group_id, peer_id) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement for adding members to group %s: %w", groupID, err)
	}
	defer stmt.Close()

	addedCount := 0
	for _, peerID := range peerIDs {
		if peerID == "" {
			log.Printf("Storage: WARN - Skipping empty peerID while adding members to group %s", groupID)
			continue
		}
		// Execute the prepared statement for each peer
		res, err := stmt.ExecContext(ctx, groupID, peerID)
		if err != nil {
			// Rollback will happen due to defer
			return fmt.Errorf("failed to add member %s to group %s: %w", peerID, groupID, err)
		}
		rowsAffected, _ := res.RowsAffected() // Check if a row was actually inserted
		if rowsAffected > 0 {
			addedCount++
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for adding members to group %s: %w", groupID, err)
	}

	log.Printf("Storage: Added %d new member(s) to group %s. Total attempted: %d", addedCount, groupID, len(peerIDs))
	return nil
}
