-- Persistent direct messages between friends
CREATE TABLE IF NOT EXISTS friend_messages (
    id          TEXT PRIMARY KEY,
    sender_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (sender_id <> receiver_id)
);

CREATE INDEX IF NOT EXISTS idx_friend_messages_sender_receiver_time
    ON friend_messages(sender_id, receiver_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_friend_messages_receiver_sender_time
    ON friend_messages(receiver_id, sender_id, created_at DESC, id DESC);
