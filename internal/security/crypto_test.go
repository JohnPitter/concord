package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCryptoManager(t *testing.T) {
	cm := NewCryptoManager()
	assert.NotNil(t, cm)
	assert.Equal(t, uint32(1), cm.argon2Time)
	assert.Equal(t, uint32(64*1024), cm.argon2Memory)
	assert.Equal(t, uint8(4), cm.argon2Threads)
	assert.Equal(t, uint32(32), cm.argon2KeyLen)
}

func TestCryptoManager_HashPassword(t *testing.T) {
	cm := NewCryptoManager()

	t.Run("hashes password successfully", func(t *testing.T) {
		password := "MySecurePassword123!"
		hash, err := cm.HashPassword(password)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.Contains(t, hash, "$argon2id$")
	})

	t.Run("produces different hashes for same password", func(t *testing.T) {
		password := "SamePassword"
		hash1, err1 := cm.HashPassword(password)
		hash2, err2 := cm.HashPassword(password)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, hash1, hash2, "different salts should produce different hashes")
	})
}

func TestCryptoManager_VerifyPassword(t *testing.T) {
	cm := NewCryptoManager()

	t.Run("verifies correct password", func(t *testing.T) {
		password := "CorrectPassword123"
		hash, err := cm.HashPassword(password)
		require.NoError(t, err)

		valid, err := cm.VerifyPassword(password, hash)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("rejects incorrect password", func(t *testing.T) {
		password := "CorrectPassword123"
		hash, err := cm.HashPassword(password)
		require.NoError(t, err)

		valid, err := cm.VerifyPassword("WrongPassword", hash)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("returns error for invalid hash format", func(t *testing.T) {
		_, err := cm.VerifyPassword("password", "invalid_hash")
		assert.Error(t, err)
	})
}

func TestCryptoManager_Encrypt_Decrypt(t *testing.T) {
	cm := NewCryptoManager()

	t.Run("encrypts and decrypts successfully", func(t *testing.T) {
		plaintext := []byte("Secret message that needs encryption")
		key, err := GenerateKey(32)
		require.NoError(t, err)

		encrypted, err := cm.Encrypt(plaintext, key)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := cm.Decrypt(encrypted, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("fails with wrong key", func(t *testing.T) {
		plaintext := []byte("Secret message")
		key1, _ := GenerateKey(32)
		key2, _ := GenerateKey(32)

		encrypted, err := cm.Encrypt(plaintext, key1)
		require.NoError(t, err)

		_, err = cm.Decrypt(encrypted, key2)
		assert.Error(t, err, "decryption should fail with wrong key")
	})

	t.Run("fails with invalid key size", func(t *testing.T) {
		plaintext := []byte("test")
		invalidKey := []byte("short")

		_, err := cm.Encrypt(plaintext, invalidKey)
		assert.Error(t, err)
	})
}

func TestCryptoManager_EncryptAES_DecryptAES(t *testing.T) {
	cm := NewCryptoManager()

	t.Run("encrypts and decrypts with AES-GCM", func(t *testing.T) {
		plaintext := []byte("AES encrypted message")
		key, err := GenerateKey(32)
		require.NoError(t, err)

		encrypted, err := cm.EncryptAES(plaintext, key)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		decrypted, err := cm.DecryptAES(encrypted, key)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("fails with wrong key", func(t *testing.T) {
		plaintext := []byte("test")
		key1, _ := GenerateKey(32)
		key2, _ := GenerateKey(32)

		encrypted, err := cm.EncryptAES(plaintext, key1)
		require.NoError(t, err)

		_, err = cm.DecryptAES(encrypted, key2)
		assert.Error(t, err)
	})
}

func TestGenerateKey(t *testing.T) {
	t.Run("generates key of correct size", func(t *testing.T) {
		key, err := GenerateKey(32)
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("generates different keys each time", func(t *testing.T) {
		key1, _ := GenerateKey(32)
		key2, _ := GenerateKey(32)
		assert.NotEqual(t, key1, key2)
	})
}

func TestGenerateToken(t *testing.T) {
	t.Run("generates token successfully", func(t *testing.T) {
		token, err := GenerateToken(32)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generates different tokens", func(t *testing.T) {
		token1, _ := GenerateToken(32)
		token2, _ := GenerateToken(32)
		assert.NotEqual(t, token1, token2)
	})
}

func TestHash(t *testing.T) {
	t.Run("produces consistent hash", func(t *testing.T) {
		input := []byte("test input")
		hash1 := Hash(input)
		hash2 := Hash(input)
		assert.Equal(t, hash1, hash2)
	})

	t.Run("produces different hash for different input", func(t *testing.T) {
		hash1 := Hash([]byte("input1"))
		hash2 := Hash([]byte("input2"))
		assert.NotEqual(t, hash1, hash2)
	})
}

func TestHashString(t *testing.T) {
	t.Run("produces consistent hash", func(t *testing.T) {
		hash1 := HashString("test")
		hash2 := HashString("test")
		assert.Equal(t, hash1, hash2)
	})

	t.Run("produces hex string", func(t *testing.T) {
		hash := HashString("test")
		assert.Len(t, hash, 64) // SHA-256 hex = 64 characters
	})
}

func TestSecureRandom(t *testing.T) {
	t.Run("generates random bytes", func(t *testing.T) {
		bytes, err := SecureRandom(16)
		require.NoError(t, err)
		assert.Len(t, bytes, 16)
	})

	t.Run("generates different values", func(t *testing.T) {
		bytes1, _ := SecureRandom(16)
		bytes2, _ := SecureRandom(16)
		assert.NotEqual(t, bytes1, bytes2)
	})
}
