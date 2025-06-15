package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core/crypto_utils"

	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/argon2"
)

const (
	keyPermissions = 0600 // Read/write only
	// Argon2id parameters
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // AES-256
	saltLen      = 16
	nonceLen     = 12
)

// GenerateAndSaveEncryptedKey generates a new Ed25519 key, encrypts it with a password, and saves it.
func GenerateAndSaveEncryptedKey(keyPath string, password []byte) (crypto.PrivKey, error) {
	log.Printf("Generating new private key at %s", keyPath)
	privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	if err := SaveEncryptedKey(keyPath, privKey, password); err != nil {
		_ = os.Remove(keyPath)
		return nil, err
	}

	log.Printf("Successfully generated and saved encrypted key to %s", keyPath)
	return privKey, nil
}

// SaveEncryptedKey encrypts the given private key with the password and saves it to keyPath.
func SaveEncryptedKey(keyPath string, privKey crypto.PrivKey, password []byte) error {
	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}

	privKeyBytes, err := crypto.MarshalPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Generate salt for KDF
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key from password using Argon2id
	derivedKey := argon2.IDKey(password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Create AES cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM AEAD cipher
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create AES-GCM cipher: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, nonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the key bytes
	ciphertext := aesgcm.Seal(nil, nonce, privKeyBytes, nil)

	// Write salt, nonce, ciphertext to file
	file, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, keyPermissions)
	if err != nil {
		return fmt.Errorf("failed to open key file for writing: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(salt); err != nil {
		return fmt.Errorf("failed to write salt: %w", err)
	}
	if _, err := file.Write(nonce); err != nil {
		return fmt.Errorf("failed to write nonce: %w", err)
	}
	if _, err := file.Write(ciphertext); err != nil {
		return fmt.Errorf("failed to write ciphertext: %w", err)
	}

	return nil
}

// LoadAndDecryptKey loads an encrypted key from keyPath and decrypts it using the password.
func LoadAndDecryptKey(keyPath string, password []byte) (crypto.PrivKey, []byte, error) {
	if len(password) == 0 {
		return nil, nil, fmt.Errorf("password cannot be empty")
	}

	file, err := os.Open(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("key file not found at %s", keyPath)
		}
		return nil, nil, fmt.Errorf("failed to open key file %s: %w", keyPath, err)
	}
	defer file.Close()

	// Read salt, nonce, and ciphertext
	salt := make([]byte, saltLen)
	nonce := make([]byte, nonceLen)

	if _, err := io.ReadFull(file, salt); err != nil {
		return nil, nil, fmt.Errorf("failed to read salt from key file: %w", err)
	}
	if _, err := io.ReadFull(file, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to read nonce from key file: %w", err)
	}

	ciphertext, err := io.ReadAll(file) // Read the rest as ciphertext
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read ciphertext from key file: %w", err)
	}

	if len(ciphertext) == 0 {
		return nil, nil, fmt.Errorf("key file %s appears corrupted (empty ciphertext)", keyPath)
	}

	// Derive decryption key
	derivedKey := argon2.IDKey(password, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	// Create AES cipher block
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES cipher for decryption: %w", err)
	}

	// Create GCM AEAD cipher
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AES-GCM cipher for decryption: %w", err)
	}

	// Decrypt the ciphertext
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Common cause: incorrect password or corrupted file
		return nil, nil, fmt.Errorf("failed to decrypt key (incorrect password or corrupted file): %w", err)
	}

	// Unmarshal the decrypted bytes back into a private key
	privKey, err := crypto.UnmarshalPrivateKey(plaintext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal decrypted key bytes: %w", err)
	}

	configDir, err := os.UserConfigDir()

	salt, err = crypto_utils.GetDatabaseFieldSalt(configDir, core.DefaultCryptoConfig)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database field salt: %w", err)
	}

	key := crypto_utils.DeriveKeyFromPassword(password, salt, core.DefaultCryptoConfig)

	log.Printf("Successfully loaded and decrypted key from %s", keyPath)
	return privKey, key, nil
}

// Helper to check if key exists
func KeyExists(keyPath string) bool {
	info, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
