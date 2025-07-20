package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// DisplayName represents a custom display name for a friend or group
type DisplayName struct {
	ID          int64     `json:"id"`
	EntityID    string    `json:"entity_id"`   // peer_id or group_id
	EntityType  string    `json:"entity_type"` // 'friend' or 'group'
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DisplayNameRepository defines operations for managing display names
type DisplayNameRepository interface {
	Store(ctx context.Context, displayName DisplayName) error
	Update(ctx context.Context, entityID, entityType, newDisplayName string) error
	GetByEntity(ctx context.Context, entityID, entityType string) (*DisplayName, error)
	Delete(ctx context.Context, entityID, entityType string) error
	GetAllByType(ctx context.Context, entityType string) ([]DisplayName, error)
}

// --- SQLite Implementation ---

type sqliteDisplayNameRepository struct {
	db *sql.DB
}

// NewSQLiteDisplayNameRepository creates a new display name repository
func NewSQLiteDisplayNameRepository(database *DB) (DisplayNameRepository, error) {
	if database == nil || database.GetDB() == nil {
		return nil, errors.New("database connection required for display name repository")
	}
	return &sqliteDisplayNameRepository{db: database.GetDB()}, nil
}

// Store saves a new display name
func (r *sqliteDisplayNameRepository) Store(ctx context.Context, displayName DisplayName) error {
	now := time.Now()
	sqlStmt := `
		INSERT OR REPLACE INTO display_names (entity_id, entity_type, display_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, sqlStmt,
		displayName.EntityID,
		displayName.EntityType,
		displayName.DisplayName,
		now.Unix(),
		now.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to store display name for %s %s: %w", displayName.EntityType, displayName.EntityID, err)
	}

	log.Printf("Storage: Stored display name '%s' for %s %s", displayName.DisplayName, displayName.EntityType, displayName.EntityID)
	return nil
}

// Update modifies an existing display name
func (r *sqliteDisplayNameRepository) Update(ctx context.Context, entityID, entityType, newDisplayName string) error {
	sqlStmt := `
		UPDATE display_names 
		SET display_name = ?, updated_at = ?
		WHERE entity_id = ? AND entity_type = ?
	`

	result, err := r.db.ExecContext(ctx, sqlStmt, newDisplayName, time.Now().Unix(), entityID, entityType)
	if err != nil {
		return fmt.Errorf("failed to update display name for %s %s: %w", entityType, entityID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	log.Printf("Storage: Updated display name to '%s' for %s %s", newDisplayName, entityType, entityID)
	return nil
}

// GetByEntity retrieves a display name for a specific entity
func (r *sqliteDisplayNameRepository) GetByEntity(ctx context.Context, entityID, entityType string) (*DisplayName, error) {
	sqlStmt := `
		SELECT id, entity_id, entity_type, display_name, created_at, updated_at
		FROM display_names
		WHERE entity_id = ? AND entity_type = ?
	`

	var dn DisplayName
	var createdAtUnix, updatedAtUnix int64

	err := r.db.QueryRowContext(ctx, sqlStmt, entityID, entityType).Scan(
		&dn.ID,
		&dn.EntityID,
		&dn.EntityType,
		&dn.DisplayName,
		&createdAtUnix,
		&updatedAtUnix,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get display name for %s %s: %w", entityType, entityID, err)
	}

	dn.CreatedAt = time.Unix(createdAtUnix, 0)
	dn.UpdatedAt = time.Unix(updatedAtUnix, 0)

	return &dn, nil
}

// Delete removes a display name
func (r *sqliteDisplayNameRepository) Delete(ctx context.Context, entityID, entityType string) error {
	sqlStmt := `DELETE FROM display_names WHERE entity_id = ? AND entity_type = ?`

	result, err := r.db.ExecContext(ctx, sqlStmt, entityID, entityType)
	if err != nil {
		return fmt.Errorf("failed to delete display name for %s %s: %w", entityType, entityID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	log.Printf("Storage: Deleted display name for %s %s", entityType, entityID)
	return nil
}

// GetAllByType retrieves all display names for a specific entity type
func (r *sqliteDisplayNameRepository) GetAllByType(ctx context.Context, entityType string) ([]DisplayName, error) {
	sqlStmt := `
		SELECT id, entity_id, entity_type, display_name, created_at, updated_at
		FROM display_names
		WHERE entity_type = ?
		ORDER BY display_name ASC
	`

	rows, err := r.db.QueryContext(ctx, sqlStmt, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to query display names for type %s: %w", entityType, err)
	}
	defer rows.Close()

	var displayNames []DisplayName
	for rows.Next() {
		var dn DisplayName
		var createdAtUnix, updatedAtUnix int64

		err := rows.Scan(
			&dn.ID,
			&dn.EntityID,
			&dn.EntityType,
			&dn.DisplayName,
			&createdAtUnix,
			&updatedAtUnix,
		)
		if err != nil {
			log.Printf("Storage: Error scanning display name row: %v", err)
			continue
		}

		dn.CreatedAt = time.Unix(createdAtUnix, 0)
		dn.UpdatedAt = time.Unix(updatedAtUnix, 0)
		displayNames = append(displayNames, dn)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating display name rows: %w", err)
	}

	return displayNames, nil
}
