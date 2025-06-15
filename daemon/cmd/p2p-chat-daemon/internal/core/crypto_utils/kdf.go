package crypto_utils

import (
	"golang.org/x/crypto/argon2"
	"p2p-chat-daemon/cmd/p2p-chat-daemon/internal/core" // For CryptoConfig
)

// DeriveKeyFromPassword uses Argon2id to derive a cryptographic key from a password and salt.
func DeriveKeyFromPassword(password []byte, salt []byte, cfg core.CryptoConfig) []byte {
	return argon2.IDKey(
		password,
		salt,
		cfg.ArgonTime,
		cfg.ArgonMemory,
		cfg.ArgonThreads,
		cfg.ArgonKeyLen,
	)
}
