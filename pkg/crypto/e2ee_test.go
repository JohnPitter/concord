package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	kp1, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, kp1.PublicKey)
	assert.NotEqual(t, [32]byte{}, kp1.PrivateKey)

	kp2, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, kp1.PublicKey, kp2.PublicKey, "two key pairs should differ")
}

func TestE2EEEncryptDecrypt(t *testing.T) {
	alice, err := NewE2EEManager()
	require.NoError(t, err)

	bob, err := NewE2EEManager()
	require.NoError(t, err)

	// Exchange public keys
	require.NoError(t, alice.AddPeerKey("bob", bob.PublicKey()))
	require.NoError(t, bob.AddPeerKey("alice", alice.PublicKey()))

	// Alice encrypts for Bob
	plaintext := []byte("hello bob, this is a secret message")
	ciphertext, err := alice.Encrypt("bob", plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	// Bob decrypts
	decrypted, err := bob.Decrypt("alice", ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestE2EEBidirectional(t *testing.T) {
	alice, err := NewE2EEManager()
	require.NoError(t, err)
	bob, err := NewE2EEManager()
	require.NoError(t, err)

	require.NoError(t, alice.AddPeerKey("bob", bob.PublicKey()))
	require.NoError(t, bob.AddPeerKey("alice", alice.PublicKey()))

	// Alice → Bob
	ct1, err := alice.Encrypt("bob", []byte("from alice"))
	require.NoError(t, err)
	pt1, err := bob.Decrypt("alice", ct1)
	require.NoError(t, err)
	assert.Equal(t, "from alice", string(pt1))

	// Bob → Alice
	ct2, err := bob.Encrypt("alice", []byte("from bob"))
	require.NoError(t, err)
	pt2, err := alice.Decrypt("bob", ct2)
	require.NoError(t, err)
	assert.Equal(t, "from bob", string(pt2))
}

func TestE2EENoSessionKey(t *testing.T) {
	mgr, err := NewE2EEManager()
	require.NoError(t, err)

	_, err = mgr.Encrypt("unknown", []byte("test"))
	assert.ErrorIs(t, err, ErrNoSessionKey)

	_, err = mgr.Decrypt("unknown", []byte("test"))
	assert.ErrorIs(t, err, ErrNoSessionKey)
}

func TestE2EEDecryptInvalidData(t *testing.T) {
	alice, err := NewE2EEManager()
	require.NoError(t, err)
	bob, err := NewE2EEManager()
	require.NoError(t, err)

	require.NoError(t, alice.AddPeerKey("bob", bob.PublicKey()))
	require.NoError(t, bob.AddPeerKey("alice", alice.PublicKey()))

	// Too short (less than nonce)
	_, err = bob.Decrypt("alice", []byte{0x01, 0x02})
	assert.ErrorIs(t, err, ErrDecryptionFailed)

	// Tampered ciphertext
	ct, err := alice.Encrypt("bob", []byte("secret"))
	require.NoError(t, err)
	ct[len(ct)-1] ^= 0xFF // flip last byte
	_, err = bob.Decrypt("alice", ct)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestE2EERemovePeer(t *testing.T) {
	mgr, err := NewE2EEManager()
	require.NoError(t, err)

	peer, err := NewE2EEManager()
	require.NoError(t, err)

	require.NoError(t, mgr.AddPeerKey("peer1", peer.PublicKey()))
	assert.True(t, mgr.HasSessionKey("peer1"))
	assert.Equal(t, 1, mgr.PeerCount())

	mgr.RemovePeer("peer1")
	assert.False(t, mgr.HasSessionKey("peer1"))
	assert.Equal(t, 0, mgr.PeerCount())
}

func TestE2EEDifferentKeysCannotDecrypt(t *testing.T) {
	alice, err := NewE2EEManager()
	require.NoError(t, err)
	bob, err := NewE2EEManager()
	require.NoError(t, err)
	eve, err := NewE2EEManager()
	require.NoError(t, err)

	// Alice and Bob exchange keys
	require.NoError(t, alice.AddPeerKey("bob", bob.PublicKey()))
	require.NoError(t, bob.AddPeerKey("alice", alice.PublicKey()))

	// Eve tries to use her own key to decrypt
	require.NoError(t, eve.AddPeerKey("alice", alice.PublicKey()))

	ct, err := alice.Encrypt("bob", []byte("for bob only"))
	require.NoError(t, err)

	// Eve cannot decrypt Alice→Bob message
	_, err = eve.Decrypt("alice", ct)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}
