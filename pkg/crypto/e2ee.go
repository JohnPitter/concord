// Package crypto provides end-to-end encryption for Concord P2P messages.
// Uses X25519 for key exchange and AES-256-GCM for symmetric encryption.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

var (
	ErrInvalidKeySize    = errors.New("crypto: invalid key size")
	ErrDecryptionFailed  = errors.New("crypto: decryption failed")
	ErrNoPeerKey         = errors.New("crypto: no public key for peer")
	ErrNoSessionKey      = errors.New("crypto: no session key established")
)

// KeyPair holds an X25519 key pair.
type KeyPair struct {
	PrivateKey [32]byte
	PublicKey  [32]byte
}

// GenerateKeyPair creates a new X25519 key pair using crypto/rand.
func GenerateKeyPair() (*KeyPair, error) {
	var priv [32]byte
	if _, err := io.ReadFull(rand.Reader, priv[:]); err != nil {
		return nil, fmt.Errorf("crypto: generate key pair: %w", err)
	}
	// Clamp private key per X25519 spec
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("crypto: compute public key: %w", err)
	}

	kp := &KeyPair{}
	copy(kp.PrivateKey[:], priv[:])
	copy(kp.PublicKey[:], pub)
	return kp, nil
}

// E2EEManager manages encryption for P2P communication.
// Each peer exchange provides a shared secret via X25519 + HKDF.
type E2EEManager struct {
	mu          sync.RWMutex
	keyPair     *KeyPair
	peerKeys    map[string][32]byte // peerID -> public key
	sessionKeys map[string][]byte   // peerID -> derived 32-byte AES key
}

// NewE2EEManager creates a new E2EE manager with a fresh key pair.
func NewE2EEManager() (*E2EEManager, error) {
	kp, err := GenerateKeyPair()
	if err != nil {
		return nil, err
	}
	return &E2EEManager{
		keyPair:     kp,
		peerKeys:    make(map[string][32]byte),
		sessionKeys: make(map[string][]byte),
	}, nil
}

// PublicKey returns our public key for sharing with peers.
func (m *E2EEManager) PublicKey() [32]byte {
	return m.keyPair.PublicKey
}

// AddPeerKey registers a peer's public key and derives the session key.
func (m *E2EEManager) AddPeerKey(peerID string, pubKey [32]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.peerKeys[peerID] = pubKey

	// X25519 key agreement
	shared, err := curve25519.X25519(m.keyPair.PrivateKey[:], pubKey[:])
	if err != nil {
		return fmt.Errorf("crypto: key exchange with %s: %w", peerID, err)
	}

	// HKDF to derive a 32-byte AES key from the shared secret
	sessionKey, err := deriveKey(shared, []byte("concord-e2ee-v1"))
	if err != nil {
		return fmt.Errorf("crypto: derive session key: %w", err)
	}

	m.sessionKeys[peerID] = sessionKey
	return nil
}

// RemovePeer removes a peer's keys.
func (m *E2EEManager) RemovePeer(peerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peerKeys, peerID)
	delete(m.sessionKeys, peerID)
}

// Encrypt encrypts plaintext for a specific peer using their session key.
// Returns nonce (12 bytes) || ciphertext.
func (m *E2EEManager) Encrypt(peerID string, plaintext []byte) ([]byte, error) {
	m.mu.RLock()
	key, ok := m.sessionKeys[peerID]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrNoSessionKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext from a specific peer.
// Expects nonce (12 bytes) || ciphertext.
func (m *E2EEManager) Decrypt(peerID string, data []byte) ([]byte, error) {
	m.mu.RLock()
	key, ok := m.sessionKeys[peerID]
	m.mu.RUnlock()

	if !ok {
		return nil, ErrNoSessionKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// HasSessionKey checks if a session key exists for a peer.
func (m *E2EEManager) HasSessionKey(peerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.sessionKeys[peerID]
	return ok
}

// PeerCount returns the number of registered peers.
func (m *E2EEManager) PeerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peerKeys)
}

// deriveKey uses HKDF-SHA256 to derive a 32-byte key from shared secret.
func deriveKey(shared, info []byte) ([]byte, error) {
	h := hkdf.New(sha256.New, shared, nil, info)
	key := make([]byte, 32)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, err
	}
	return key, nil
}
