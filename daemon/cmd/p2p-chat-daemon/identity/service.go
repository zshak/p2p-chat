package identity

import (
	"errors"
	"github.com/libp2p/go-libp2p/core/crypto"
	"p2p-chat-daemon/cmd/config"
	"sync"
)

// Service handles identity management (key loading, generation)
type Service struct {
	cfg     *config.P2PConfig
	keyPath string
	mutex   sync.RWMutex
	privKey crypto.PrivKey
}

func NewService(cfg *config.P2PConfig) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	return &Service{
		cfg:     cfg,
		keyPath: cfg.PrivateKeyPath,
	}, nil
}

func (s *Service) KeyExists() bool {
	return KeyExists(s.keyPath)
}

// LoadAndDecrypt attempts to load the key
// Decrypts the key and stores it internally.
func (s *Service) LoadAndDecrypt(password []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.privKey != nil {
		return errors.New("key already loaded")
	}

	key, err := LoadAndDecryptKey(s.keyPath, password)
	if err != nil {
		return err
	}
	s.privKey = key
	return nil
}

// GenerateAndEncrypt generates encryption key from user password
// and saves it in the keyPath
func (s *Service) GenerateAndEncrypt(password []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.privKey != nil {
		return errors.New("key already exists in memory")
	}
	if KeyExists(s.keyPath) {
		return errors.New("key file already exists on disk")
	}

	key, err := GenerateAndSaveEncryptedKey(s.keyPath, password)
	if err != nil {
		return err
	}
	s.privKey = key
	return nil
}

// GetPrivateKey returns the loaded private key. Returns nil if not loaded/decrypted.
func (s *Service) GetPrivateKey() crypto.PrivKey {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.privKey
}
