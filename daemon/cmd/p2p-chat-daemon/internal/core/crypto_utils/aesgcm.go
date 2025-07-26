package crypto_utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core"
)

// EncryptDataWithKey encrypts data using a provided key and returns (nonce || ciphertext).
func EncryptDataWithKey(key []byte, plaintext []byte, cfg core.CryptoConfig) ([]byte, error) {
	if len(key) != int(cfg.ArgonKeyLen) {
		return nil, fmt.Errorf("invalid key length: got %d, want %d", len(key), cfg.ArgonKeyLen)
	}
	if plaintext == nil {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM: %w", err)
	}

	nonce := make([]byte, cfg.NonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// DecryptDataWithKey decrypts (nonce || ciphertext) using a provided key.
func DecryptDataWithKey(key []byte, ciphertextWithNonce []byte, cfg core.CryptoConfig) ([]byte, error) {
	if len(key) != int(cfg.ArgonKeyLen) {
		return nil, fmt.Errorf("invalid key length")
	}
	if len(ciphertextWithNonce) < cfg.NonceLen {
		if len(ciphertextWithNonce) == 0 {
			return nil, nil
		}
		return nil, errors.New("ciphertext too short to contain nonce")
	}

	nonce := ciphertextWithNonce[:cfg.NonceLen]
	ciphertext := ciphertextWithNonce[cfg.NonceLen:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher for decryption: %w", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM for decryption: %w", err)
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt/authenticate data: %w", err)
	}
	return plaintext, nil
}
