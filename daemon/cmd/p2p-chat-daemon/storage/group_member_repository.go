package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

type GroupMemberRepository interface {
	AddMembers(ctx context.Context, groupID string, peerIDs []string) error
	GetGroupsWithMembers(ctx context.Context) (map[string][]string, error)
	GetGroups(ctx context.Context) ([]GroupInfo, error)
}

type GroupInfo struct {
	GroupID string   `json:"group_id"`
	Members []string `json:"members"`
	Name    string   `json:"name"`
}

type sqliteGroupMemberRepository struct {
	db *sql.DB
}

func NewSQLiteGroupMemberRepository(database *DB) (GroupMemberRepository, error) {
	if database == nil || database.GetDB() == nil {
		return nil, errors.New("database connection required for group member repository")
	}
	return &sqliteGroupMemberRepository{db: database.GetDB()}, nil
}

func (r *sqliteGroupMemberRepository) AddMembers(ctx context.Context, groupID string, peerIDs []string) error {
	if groupID == "" {
		return errors.New("groupID cannot be empty")
	}
	if len(peerIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for adding members to group %s: %w", groupID, err)
	}
	defer tx.Rollback()

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
		res, err := stmt.ExecContext(ctx, groupID, peerID)
		if err != nil {
			return fmt.Errorf("failed to add member %s to group %s: %w", peerID, groupID, err)
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected > 0 {
			addedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for adding members to group %s: %w", groupID, err)
	}

	log.Printf("Storage: Added %d new member(s) to group %s. Total attempted: %d", addedCount, groupID, len(peerIDs))
	return nil
}

func (r *sqliteGroupMemberRepository) GetGroupsWithMembers(ctx context.Context) (map[string][]string, error) {
	result := make(map[string][]string)

	rows, err := r.db.QueryContext(ctx, "SELECT group_id, peer_id FROM group_members ORDER BY group_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query group members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var groupID, peerID string
		if err := rows.Scan(&groupID, &peerID); err != nil {
			return nil, fmt.Errorf("failed to scan group member row: %w", err)
		}

		result[groupID] = append(result[groupID], peerID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating group members: %w", err)
	}

	return result, nil
}

func (r *sqliteGroupMemberRepository) GetGroups(ctx context.Context) ([]GroupInfo, error) {
	groupsMap, err := r.GetGroupsWithMembers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups with members: %w", err)
	}

	var groups []GroupInfo
	for groupID, members := range groupsMap {

		name, err := r.GetGroupName(ctx, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get group name for ID %s: %w", groupID, err)
		}
		groups = append(groups, GroupInfo{
			GroupID: groupID,
			Members: members,
			Name:    name,
		})
	}

	log.Printf("Storage: Retrieved %d groups", len(groups))
	return groups, nil
}

func (r *sqliteGroupMemberRepository) GetGroupName(ctx context.Context, groupID string) (string, error) {
	var name string
	query := `SELECT name FROM group_keys WHERE group_id = $1`
	err := r.db.QueryRowContext(ctx, query, groupID).Scan(&name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("group name not found for ID %s", groupID)
		}
		return "", fmt.Errorf("database query failed for group name: %w", err)
	}
	return name, nil
}
