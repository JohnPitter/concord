package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// CryptoManager handles encryption and hashing operations
type CryptoManager struct {
	// Argon2 parameters (recommended defaults)
	argon2Time    uint32
	argon2Memory  uint32
	argon2Threads uint8
	argon2KeyLen  uint32
}

// NewCryptoManager creates a new crypto manager with secure defaults
func NewCryptoManager() *CryptoManager {
	return &CryptoManager{
		argon2Time:    1,      // Number of iterations
		argon2Memory:  64*1024, // 64 MB
		argon2Threads: 4,      // Number of threads
		argon2KeyLen:  32,     // 256-bit key
	}
}

// HashPassword hashes a password using Argon2id
// Argon2id is the recommended password hashing algorithm (winner of Password Hashing Competition)
// Complexity: O(memory * time) - intentionally slow to prevent brute force
func (cm *CryptoManager) HashPassword(password string) (string, error) {
	// Generate random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash password
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		cm.argon2Time,
		cm.argon2Memory,
		cm.argon2Threads,
		cm.argon2KeyLen,
	)

	// Encode salt and hash to base64
	saltEncoded := base64.RawStdEncoding.EncodeToString(salt)
	hashEncoded := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	formatted := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		cm.argon2Memory, cm.argon2Time, cm.argon2Threads,
		saltEncoded, hashEncoded,
	)

	return formatted, nil
}

// VerifyPassword verifies a password against its hash
// Complexity: O(memory * time) - same as hashing
func (cm *CryptoManager) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	var memory, time uint32
	var threads uint8
	var saltEncoded, hashEncoded string

	_, err := fmt.Sscanf(encodedHash, "$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		&memory, &time, &threads, &saltEncoded, &hashEncoded,
	)
	if err != nil {
		return false, fmt.Errorf("invalid hash format: %w", err)
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(saltEncoded)
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashEncoded)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the input password with the same parameters
	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	// Constant-time comparison to prevent timing attacks
	if len(actualHash) != len(expectedHash) {
		return false, nil
	}

	var diff byte
	for i := range actualHash {
		diff |= actualHash[i] ^ expectedHash[i]
	}

	return diff == 0, nil
}

// Encrypt encrypts data using ChaCha20-Poly1305 AEAD cipher
// ChaCha20-Poly1305 is modern, fast, and secure
// Complexity: O(n) where n is the length of plaintext
func (cm *CryptoManager) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	result := append(nonce, ciphertext...)

	return result, nil
}

// Decrypt decrypts data using ChaCha20-Poly1305 AEAD cipher
// Complexity: O(n) where n is the length of ciphertext
func (cm *CryptoManager) Decrypt(encrypted []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	if len(encrypted) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := encrypted[:aead.NonceSize()]
	ciphertext := encrypted[aead.NonceSize():]

	// Decrypt and verify
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptAES encrypts data using AES-256-GCM
// AES-GCM is widely supported and hardware-accelerated on most CPUs
// Complexity: O(n) where n is the length of plaintext
func (cm *CryptoManager) EncryptAES(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	result := append(nonce, ciphertext...)

	return result, nil
}

// DecryptAES decrypts data using AES-256-GCM
// Complexity: O(n) where n is the length of ciphertext
func (cm *CryptoManager) DecryptAES(encrypted []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(encrypted) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := encrypted[:aead.NonceSize()]
	ciphertext := encrypted[aead.NonceSize():]

	// Decrypt and verify
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// GenerateKey generates a cryptographically secure random key
// Complexity: O(1)
func GenerateKey(size int) ([]byte, error) {
	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateToken generates a cryptographically secure random token
// Complexity: O(1)
func GenerateToken(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Hash creates a SHA-256 hash of the input
// Complexity: O(n) where n is the length of input
func Hash(input []byte) []byte {
	hash := sha256.Sum256(input)
	return hash[:]
}

// HashString creates a SHA-256 hash of a string and returns it as hex
// Complexity: O(n) where n is the length of input
func HashString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// SecureRandom generates cryptographically secure random bytes
// Complexity: O(1)
func SecureRandom(n int) ([]byte, error) {
	bytes := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}
