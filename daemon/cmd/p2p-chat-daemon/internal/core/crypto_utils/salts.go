package crypto_utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath" // For joining paths

	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core" // For CryptoConfig
)

const (
	identityKeySaltFilename   = "identity.salt"
	databaseFieldSaltFilename = "dbfield.salt"
	filePermissions           = 0600 // Read/write only for user
)

// EnsureSaltFileExists generates and saves a salt file if it doesn't exist,
// otherwise it loads the existing salt.
// appDataDir is the base directory for storing these salt files.
func EnsureSaltFileExists(appDataDir string, saltFilename string, saltLen int) ([]byte, error) {
	appDataDir = filepath.Join(appDataDir, "p2p-chat-daemon")

	if appDataDir == "" {
		return nil, errors.New("appDataDir cannot be empty")
	}
	if err := os.MkdirAll(appDataDir, 0700); err != nil { // Ensure directory exists
		return nil, fmt.Errorf("could not create app data sub-directory %s: %w", appDataDir, err)
	}

	filePath := filepath.Join(appDataDir, saltFilename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File does not exist, generate and save new salt
		log.Printf("CryptoUtils: Salt file %s not found, generating new one.", filePath)
		salt := make([]byte, saltLen)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, fmt.Errorf("failed to generate new salt for %s: %w", saltFilename, err)
		}
		if err := os.WriteFile(filePath, salt, filePermissions); err != nil {
			return nil, fmt.Errorf("failed to write new salt file %s: %w", filePath, err)
		}
		log.Printf("CryptoUtils: New salt generated and saved to %s", filePath)
		return salt, nil
	} else if err != nil {
		// Other error stating the file (e.g., permission issue)
		return nil, fmt.Errorf("error stating salt file %s: %w", filePath, err)
	}

	// File exists, load it
	salt, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing salt file %s: %w", filePath, err)
	}
	if len(salt) != saltLen {
		return nil, fmt.Errorf("existing salt file %s has incorrect length: got %d, want %d", filePath, len(salt), saltLen)
	}
	log.Printf("CryptoUtils: Loaded existing salt from %s", filePath)
	return salt, nil
}

// GetIdentityFileSalt ensures and returns the salt for identity key file encryption.
func GetIdentityFileSalt(appDataDir string, cfg core.CryptoConfig) ([]byte, error) {
	return EnsureSaltFileExists(appDataDir, identityKeySaltFilename, cfg.SaltLen)
}

// GetDatabaseFieldSalt ensures and returns the salt for database field encryption.
func GetDatabaseFieldSalt(appDataDir string, cfg core.CryptoConfig) ([]byte, error) {
	return EnsureSaltFileExists(appDataDir, databaseFieldSaltFilename, cfg.SaltLen)
}
