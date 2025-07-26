package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/config"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	dbDriverName = "sqlite"
)

type DB struct {
	sqlDB *sql.DB
	dsn   string
	mu    sync.Mutex
}

func NewDB(config *config.Config) (*DB, error) {
	if config.P2P.DbPath == "" {
		return nil, errors.New("database data directory cannot be empty")
	}

	dbPath := config.P2P.DbPath
	log.Printf("Storage: Initializing database at %s", dbPath)

	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)

	dbHandle, err := sql.Open(dbDriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare database connection pool for %s: %w", dbPath, err)
	}

	if err := dbHandle.Ping(); err != nil {
		dbHandle.Close()
		return nil, fmt.Errorf("failed to connect to database %s: %w", dbPath, err)
	}

	dbHandle.SetMaxOpenConns(5)
	dbHandle.SetMaxIdleConns(2)
	dbHandle.SetConnMaxLifetime(time.Hour)

	database := &DB{
		sqlDB: dbHandle,
		dsn:   dsn,
	}

	if err := database.ensureCreation(); err != nil {
		database.Close()
		return nil, fmt.Errorf("database schema creation failed: %w", err)
	}

	log.Println("Storage: Database connection pool initialized and schema ready.")
	return database, nil
}

func (db *DB) ensureCreation() error {
	schemaSQL := `
		CREATE TABLE IF NOT EXISTS messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_peer_id TEXT NOT NULL,
			recipient_peer_id TEXT NOT NULL,
			send_time TEXT NOT NULL,
			content BLOB NOT NULL,  
			is_outgoing BOOLEAN NOT NULL
		);

		CREATE TABLE IF NOT EXISTS relationships (
			 peer_id TEXT PRIMARY KEY NOT NULL,
			 status TEXT NOT NULL DEFAULT 'None',
			 requested_at TEXT DEFAULT NULL,
			 approved_at TEXT DEFAULT NULL
		);

		CREATE TABLE IF NOT EXISTS group_keys (
			group_id TEXT PRIMARY KEY NOT NULL,
			group_key BLOB NOT NULL,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);

		CREATE TABLE IF NOT EXISTS group_members (
			group_id TEXT NOT NULL,
			peer_id TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS group_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id TEXT NOT NULL,
			sender_peer_id TEXT NOT NULL,
			content BLOB NOT NULL,
			sent_at INTEGER NOT NULL
		);
		
		CREATE TABLE IF NOT EXISTS display_names (
    		id INTEGER PRIMARY KEY AUTOINCREMENT,
    		entity_id TEXT NOT NULL,           -- peer_id or group_id
    		entity_type TEXT NOT NULL,         -- 'friend' or 'group'
    		display_name TEXT NOT NULL,
    		created_at INTEGER NOT NULL,
    		updated_at INTEGER NOT NULL,
    		UNIQUE(entity_id, entity_type)
		);

		CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages (recipient_peer_id);
		CREATE INDEX IF NOT EXISTS idx_relationships_peer_id ON relationships (peer_id);
		CREATE INDEX IF NOT EXISTS idx_display_names_entity ON display_names (entity_id, entity_type);

	`

	log.Println("Storage: Applying database schema...")
	_, err := db.sqlDB.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema SQL: %w", err)
	}
	log.Println("Storage: Schema applied successfully.")
	return nil
}

func (db *DB) Close() error {
	log.Println("Storage: Closing database connection pool...")
	if db.sqlDB == nil {
		return nil
	}
	err := db.sqlDB.Close()
	db.sqlDB = nil
	if err != nil {
		log.Printf("Storage: Error closing database: %v", err)
	} else {
		log.Println("Storage: Database closed.")
	}
	return err
}

func (db *DB) GetDB() *sql.DB {
	return db.sqlDB
}
