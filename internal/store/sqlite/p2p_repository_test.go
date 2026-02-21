package sqlite

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP2PRepo_SaveAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	migrator := NewMigrator(db, db.logger)
	require.NoError(t, migrator.Migrate(ctx))

	repo := NewP2PRepo(db)

	msg := P2PMessage{
		ID: "test-1", PeerID: "peer-abc",
		Direction: "sent", Content: "hello", SentAt: "2026-02-21T10:00:00Z",
	}
	require.NoError(t, repo.SaveMessage(ctx, msg))

	msgs, err := repo.GetMessages(ctx, "peer-abc", 10)
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
	assert.Equal(t, "hello", msgs[0].Content)
	assert.Equal(t, "sent", msgs[0].Direction)
}

func TestP2PRepo_GetEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	migrator := NewMigrator(db, db.logger)
	require.NoError(t, migrator.Migrate(ctx))

	repo := NewP2PRepo(db)
	msgs, err := repo.GetMessages(ctx, "no-such-peer", 10)
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestP2PRepo_OrderBySentAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	migrator := NewMigrator(db, db.logger)
	require.NoError(t, migrator.Migrate(ctx))

	repo := NewP2PRepo(db)

	// Insert messages out of order
	msgs := []P2PMessage{
		{ID: "m3", PeerID: "peer-1", Direction: "received", Content: "third", SentAt: "2026-02-21T12:00:00Z"},
		{ID: "m1", PeerID: "peer-1", Direction: "sent", Content: "first", SentAt: "2026-02-21T10:00:00Z"},
		{ID: "m2", PeerID: "peer-1", Direction: "received", Content: "second", SentAt: "2026-02-21T11:00:00Z"},
	}
	for _, m := range msgs {
		require.NoError(t, repo.SaveMessage(ctx, m))
	}

	result, err := repo.GetMessages(ctx, "peer-1", 10)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, "first", result[0].Content)
	assert.Equal(t, "second", result[1].Content)
	assert.Equal(t, "third", result[2].Content)
}

func TestP2PRepo_LimitResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	migrator := NewMigrator(db, db.logger)
	require.NoError(t, migrator.Migrate(ctx))

	repo := NewP2PRepo(db)

	for i := 0; i < 5; i++ {
		msg := P2PMessage{
			ID: fmt.Sprintf("m%d", i), PeerID: "peer-1",
			Direction: "sent", Content: fmt.Sprintf("msg-%d", i),
			SentAt: fmt.Sprintf("2026-02-21T10:%02d:00Z", i),
		}
		require.NoError(t, repo.SaveMessage(ctx, msg))
	}

	result, err := repo.GetMessages(ctx, "peer-1", 3)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}
