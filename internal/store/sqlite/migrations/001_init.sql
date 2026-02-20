-- Initial schema for Concord
-- Creates all core tables for users, servers, channels, messages, and attachments

-- Users table
-- Stores user information from GitHub OAuth
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,
    github_id   INTEGER UNIQUE NOT NULL,
    username    TEXT NOT NULL,
    avatar_url  TEXT,
    email       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_github_id ON users(github_id);
CREATE INDEX idx_users_username ON users(username);

-- Servers (guilds) table
-- Represents Discord-like servers/communities
CREATE TABLE IF NOT EXISTS servers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    icon_url    TEXT,
    owner_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invite_code TEXT UNIQUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_servers_owner_id ON servers(owner_id);
CREATE INDEX idx_servers_invite_code ON servers(invite_code);

-- Channels table
-- Text and voice channels within servers
CREATE TABLE IF NOT EXISTS channels (
    id          TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    type        TEXT NOT NULL CHECK(type IN ('text', 'voice')),
    position    INTEGER DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_channels_server_id ON channels(server_id);
CREATE INDEX idx_channels_type ON channels(type);
CREATE INDEX idx_channels_position ON channels(server_id, position);

-- Messages table
-- Stores all text messages with full-text search support
CREATE TABLE IF NOT EXISTS messages (
    id          TEXT PRIMARY KEY,
    channel_id  TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    type        TEXT DEFAULT 'text' CHECK(type IN ('text', 'file', 'system')),
    reply_to_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
    edited_at   DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_channel_created ON messages(channel_id, created_at DESC);
CREATE INDEX idx_messages_author ON messages(author_id);
CREATE INDEX idx_messages_reply_to ON messages(reply_to_id);

-- Full-text search virtual table for messages
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    content=messages,
    content_rowid=rowid
);

-- Triggers to keep FTS index in sync with messages table
CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content) VALUES (new.rowid, new.content);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
    INSERT INTO messages_fts(rowid, content) VALUES (new.rowid, new.content);
END;

-- Attachments table
-- Stores file metadata for file messages
CREATE TABLE IF NOT EXISTS attachments (
    id          TEXT PRIMARY KEY,
    message_id  TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename    TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL,
    mime_type   TEXT NOT NULL,
    hash        TEXT NOT NULL,  -- SHA-256 for integrity verification
    local_path  TEXT,           -- Local cache path
    remote_url  TEXT,           -- URL if stored on server
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attachments_message_id ON attachments(message_id);
CREATE INDEX idx_attachments_hash ON attachments(hash);

-- Server members table
-- Junction table for users and servers with roles
CREATE TABLE IF NOT EXISTS server_members (
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT DEFAULT 'member' CHECK(role IN ('owner', 'admin', 'moderator', 'member')),
    nickname    TEXT,
    muted       BOOLEAN DEFAULT 0,
    deafened    BOOLEAN DEFAULT 0,
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_server_members_user ON server_members(user_id);
CREATE INDEX idx_server_members_role ON server_members(server_id, role);

-- Channel members table
-- Tracks who is currently in a voice channel
CREATE TABLE IF NOT EXISTS channel_members (
    channel_id  TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    muted       BOOLEAN DEFAULT 0,
    deafened    BOOLEAN DEFAULT 0,
    speaking    BOOLEAN DEFAULT 0,
    PRIMARY KEY (channel_id, user_id)
);

CREATE INDEX idx_channel_members_user ON channel_members(user_id);

-- Direct messages table
-- For 1-on-1 conversations outside of servers
CREATE TABLE IF NOT EXISTS direct_messages (
    id          TEXT PRIMARY KEY,
    user1_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_message_at DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user1_id, user2_id),
    CHECK(user1_id < user2_id)  -- Ensure consistent ordering
);

CREATE INDEX idx_direct_messages_user1 ON direct_messages(user1_id);
CREATE INDEX idx_direct_messages_user2 ON direct_messages(user2_id);

-- DM messages table
CREATE TABLE IF NOT EXISTS dm_messages (
    id          TEXT PRIMARY KEY,
    dm_id       TEXT NOT NULL REFERENCES direct_messages(id) ON DELETE CASCADE,
    author_id   TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    type        TEXT DEFAULT 'text' CHECK(type IN ('text', 'file')),
    reply_to_id TEXT REFERENCES dm_messages(id) ON DELETE SET NULL,
    edited_at   DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_dm_messages_dm_created ON dm_messages(dm_id, created_at DESC);
CREATE INDEX idx_dm_messages_author ON dm_messages(author_id);

-- Sessions table
-- Stores encrypted refresh tokens for authentication
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    device_name TEXT,
    ip_address  TEXT,
    user_agent  TEXT,
    expires_at  DATETIME NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Invites table
-- Tracks server invite links
CREATE TABLE IF NOT EXISTS invites (
    code        TEXT PRIMARY KEY,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    created_by  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    max_uses    INTEGER DEFAULT 0,  -- 0 = unlimited
    uses        INTEGER DEFAULT 0,
    expires_at  DATETIME,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invites_server_id ON invites(server_id);
CREATE INDEX idx_invites_expires_at ON invites(expires_at);

-- Settings table
-- User-specific settings and preferences
CREATE TABLE IF NOT EXISTS user_settings (
    user_id     TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme       TEXT DEFAULT 'void',
    language    TEXT DEFAULT 'en',
    notifications_enabled BOOLEAN DEFAULT 1,
    sound_enabled BOOLEAN DEFAULT 1,
    voice_input_device TEXT,
    voice_output_device TEXT,
    voice_input_volume INTEGER DEFAULT 100,
    voice_output_volume INTEGER DEFAULT 100,
    translation_enabled BOOLEAN DEFAULT 0,
    translation_lang TEXT DEFAULT 'en',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- P2P peers table (optional, for caching peer information)
CREATE TABLE IF NOT EXISTS p2p_peers (
    peer_id     TEXT PRIMARY KEY,
    user_id     TEXT REFERENCES users(id) ON DELETE CASCADE,
    multiaddr   TEXT NOT NULL,
    last_seen   DATETIME DEFAULT CURRENT_TIMESTAMP,
    connection_type TEXT CHECK(connection_type IN ('direct', 'hole_punch', 'relay')),
    latency_ms  INTEGER,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_p2p_peers_user_id ON p2p_peers(user_id);
CREATE INDEX idx_p2p_peers_last_seen ON p2p_peers(last_seen);

-- Cache table for translation results
CREATE TABLE IF NOT EXISTS translation_cache (
    hash        TEXT PRIMARY KEY,  -- Hash of (source_text + source_lang + target_lang)
    source_text TEXT NOT NULL,
    source_lang TEXT NOT NULL,
    target_lang TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    access_count INTEGER DEFAULT 1
);

CREATE INDEX idx_translation_cache_langs ON translation_cache(source_lang, target_lang);
CREATE INDEX idx_translation_cache_accessed ON translation_cache(accessed_at);
