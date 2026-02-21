package p2p

import (
	"crypto/sha256"
	"fmt"
	"math/big"
)

// roomWordlist is used to generate human-readable room codes.
// In production, a larger embedded wordlist would be used.
var roomWordlist = []string{
	"alpha", "bravo", "cobra", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey",
	"xray", "yankee", "zulu", "amber", "blaze", "cedar", "dawn",
	"ember", "forge", "grove", "haven", "ivory", "jade", "knot", "lunar",
}

// RoomCode generates a deterministic, human-readable room code from the Host's peer ID.
// The code has the form "word-NNNN" (e.g., "alpha-4271").
// Complexity: O(1).
func (h *Host) RoomCode() string {
	id := h.host.ID().String()
	hash := sha256.Sum256([]byte(id))
	n := new(big.Int).SetBytes(hash[:4])
	wordIdx := new(big.Int).Mod(n, big.NewInt(int64(len(roomWordlist)))).Int64()
	numPart := new(big.Int).Mod(new(big.Int).Rsh(n, 10), big.NewInt(9000)).Int64() + 1000
	return fmt.Sprintf("%s-%d", roomWordlist[wordIdx], numPart)
}

// RoomRendezvous returns the DHT rendezvous string for a given room code.
// Use this as the argument to Host.FindPeers to join a room.
func RoomRendezvous(code string) string {
	return "concord-room/" + code
}
