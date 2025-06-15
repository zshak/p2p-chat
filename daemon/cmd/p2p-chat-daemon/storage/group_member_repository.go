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
	AddMembers(ctx context.Context, groupID string, peerIDs []string) error
	GetGroupsWithMembers(ctx context.Context) (map[string][]string, error)
	GetGroups(ctx context.Context) ([]GroupInfo, error)
}

// GroupInfo represents a group with its members
type GroupInfo struct {
	GroupID string   `json:"group_id"`
	Members []string `json:"members"`
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

// GetGroupsWithMembers returns a map where keys are group IDs and values are lists of peer IDs that are members of those groups
func (r *sqliteGroupMemberRepository) GetGroupsWithMembers(ctx context.Context) (map[string][]string, error) {
	// Initialize the result map
	result := make(map[string][]string)

	// Query to get all group members
	rows, err := r.db.QueryContext(ctx, "SELECT group_id, peer_id FROM group_members ORDER BY group_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer rows.Close()

	// Process each row
	for rows.Next() {
		var groupID, peerID string
		if err := rows.Scan(&groupID, &peerID); err != nil {
			return nil, fmt.Errorf("failed to scan group member row: %w", err)
		}

		// Append the peer ID to the appropriate group in the result map
		result[groupID] = append(result[groupID], peerID)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating group members: %w", err)
	}

	return result, nil
}

// GetGroups returns a list of GroupInfo objects containing group IDs and their members
func (r *sqliteGroupMemberRepository) GetGroups(ctx context.Context) ([]GroupInfo, error) {
	// Get the map of groups with their members
	groupsMap, err := r.GetGroupsWithMembers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups with members: %w", err)
	}

	// Convert the map to a slice of GroupInfo
	var groups []GroupInfo
	for groupID, members := range groupsMap {
		groups = append(groups, GroupInfo{
			GroupID: groupID,
			Members: members,
		})
	}

	log.Printf("Storage: Retrieved %d groups", len(groups))
	return groups, nil
}
