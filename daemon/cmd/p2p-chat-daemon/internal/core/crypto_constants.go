package core

// CryptoConfig holds constants for cryptographic operations.
// This helps centralize parameters.
type CryptoConfig struct {
	// For KDF (Argon2id)
	ArgonTime    uint32 // Iterations
	ArgonMemory  uint32 // KiB
	ArgonThreads uint8  // Parallelism
	ArgonKeyLen  uint32 // Desired key length in bytes (e.g., 32 for AES-256)
	SaltLen      int    // Bytes

	// For AES-GCM
	NonceLen int // Bytes (typically 12 for GCM)
}

// DefaultCryptoConfig provides sensible default parameters.
var DefaultCryptoConfig = CryptoConfig{
	ArgonTime:    1,
	ArgonMemory:  64 * 1024, // 64 MiB
	ArgonThreads: 4,
	ArgonKeyLen:  32, // AES-256
	SaltLen:      16,
	NonceLen:     12,
}
