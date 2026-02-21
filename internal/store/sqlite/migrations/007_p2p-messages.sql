-- P2P direct messages (local only, not synced to server)
CREATE TABLE IF NOT EXISTS p2p_messages (
  id        TEXT PRIMARY KEY,
  peer_id   TEXT NOT NULL,
  direction TEXT NOT NULL CHECK(direction IN ('sent','received')),
  content   TEXT NOT NULL,
  sent_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_p2p_messages_peer ON p2p_messages(peer_id, sent_at);
