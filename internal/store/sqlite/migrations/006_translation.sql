-- Translation cache for persisting translated text results
CREATE TABLE IF NOT EXISTS translation_cache (
    id              TEXT PRIMARY KEY,
    source_lang     TEXT NOT NULL,
    target_lang     TEXT NOT NULL,
    source_text     TEXT NOT NULL,
    translated_text TEXT NOT NULL,
    hash            TEXT NOT NULL UNIQUE,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL
);

-- Index on hash for O(1) lookups
CREATE INDEX IF NOT EXISTS idx_translation_cache_hash ON translation_cache(hash);
