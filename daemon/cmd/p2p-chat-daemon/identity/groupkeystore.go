package identity

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/core/types"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/storage"
	"sync"
)

const (
	groupKeySize   = 32 // AES-256
	groupNonceSize = 12 // AES-GCM standard nonce size
)

// GroupKeyStore manages symmetric keys for groups.
type GroupKeyStore struct {
	mu      sync.RWMutex
	keyRepo storage.KeyRepository
	ctx     context.Context
}

// NewGroupKeyStore creates a new GroupKeyStore.
func NewGroupKeyStore(keyRepo storage.KeyRepository, ctx context.Context) *GroupKeyStore {
	return &GroupKeyStore{
		keyRepo: keyRepo,
		ctx:     ctx,
	}
}

// GenerateNewKey creates a new symmetric key for a group and stores it.
// In a real app, this would only be done by a group admin/creator.
func (s *GroupKeyStore) GenerateNewKey(groupID string) ([]byte, error) {
	key := make([]byte, groupKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate group key for %s: %w", groupID, err)
	}

	s.keyRepo.Store(s.ctx, types.GroupKey{
		Key:     key,
		GroupId: groupID,
	})

	return key, nil
}

// GetKey retrieves the symmetric encryption key for a given groupID.
// It returns the key and a boolean indicating if the key was found.
func (s *GroupKeyStore) GetKey(groupID string) ([]byte, bool) {

	key, err := s.keyRepo.GetKey(s.ctx, groupID)

	if err != nil {
		return nil, false
	}

	return key.Key, true
}

// Encrypt encrypts plaintext using the group's key.
// Returns ciphertext (nonce prefixed).
func (s *GroupKeyStore) Encrypt(groupID string, plaintext []byte) ([]byte, error) {
	key, ok := s.GetKey(groupID) // GetKey now returns a copy
	if !ok {
		return nil, fmt.Errorf("no key found for group %s to encrypt", groupID)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher for group %s: %w", groupID, err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM for group %s: %w", groupID, err)
	}

	nonce := make([]byte, groupNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce for group %s encryption: %w", groupID, err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// Decrypt decrypts ciphertext (nonce prefixed) using the group's key.
func (s *GroupKeyStore) Decrypt(groupID string, ciphertextWithNonce []byte) ([]byte, error) {
	key, ok := s.GetKey(groupID) // GetKey now returns a copy
	if !ok {
		return nil, fmt.Errorf("no key found for group %s to decrypt", groupID)
	}

	if len(ciphertextWithNonce) < groupNonceSize {
		return nil, errors.New("ciphertext too short to contain nonce")
	}
	nonce := ciphertextWithNonce[:groupNonceSize]
	ciphertext := ciphertextWithNonce[groupNonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher for group %s decryption: %w", groupID, err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM for group %s decryption: %w", groupID, err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt/authenticate message for group %s: %w", groupID, err)
	}

	return plaintext, nil
}
