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

type KeyRepository interface {
	Store(ctx context.Context, key types.GroupKey) error

	GetKey(ctx context.Context, groupID string) (*types.GroupKey, error)
}

type sqliteKeyRepository struct {
	db *sql.DB
}

func NewSQLiteKeyRepository(database *DB) (KeyRepository, error) {
	if database == nil {
		return nil, errors.New("database connection is required for message repository")
	}
	return &sqliteKeyRepository{db: database.GetDB()}, nil
}

func (r *sqliteKeyRepository) Store(ctx context.Context, key types.GroupKey) error {
	sqlStmt := `
		REPLACE INTO group_keys (group_id, group_key, name, created_at)
		VALUES (?, ?, ?, ?);
	`
	_, err := r.db.ExecContext(ctx, sqlStmt,
		key.GroupId,
		key.Key,
		key.Name,
		time.Now().Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to store key for group %s: %w", key.GroupId, err)
	}
	log.Printf("Storage: Stored/Replaced key for group %s", key.GroupId)

	return nil
}

func (r *sqliteKeyRepository) GetKey(ctx context.Context, groupID string) (*types.GroupKey, error) {
	sqlStmt := `SELECT group_id, group_key, name, created_at FROM group_keys WHERE group_id = ?;`
	var gk types.GroupKey
	var createdAtUnix int64

	err := r.db.QueryRowContext(ctx, sqlStmt, groupID).Scan(
		&gk.GroupId,
		&gk.Key,
		&gk.Name,
		&createdAtUnix,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("Storage: Key not found for group %s", groupID)
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get key for group %s: %w", groupID, err)
	}
	gk.CreatedAt = time.Unix(createdAtUnix, 0)

	return &gk, nil
}
