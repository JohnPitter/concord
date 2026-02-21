# Concord Security Model

This document describes the security architecture of Concord, covering authentication, authorization, encryption, input validation, rate limiting, and file security.

---

## Table of Contents

- [Authentication](#authentication)
- [Authorization (RBAC)](#authorization-rbac)
- [End-to-End Encryption](#end-to-end-encryption)
- [Token Security](#token-security)
- [Input Validation & Sanitization](#input-validation--sanitization)
- [Rate Limiting & Brute Force Protection](#rate-limiting--brute-force-protection)
- [File Security](#file-security)
- [Cryptographic Primitives](#cryptographic-primitives)
- [OWASP Top 10 Mitigations](#owasp-top-10-mitigations)

---

## Authentication

### GitHub Device Flow (RFC 8628)

Concord uses the OAuth 2.0 Device Authorization Grant as defined in [RFC 8628](https://www.rfc-editor.org/rfc/rfc8628). This is ideal for desktop apps where the user authenticates via their browser.

**Flow:**

```
Client                       GitHub                       Browser
  |                            |                            |
  |-- POST /login/device/code ->|                            |
  |<- device_code + user_code --|                            |
  |                            |                            |
  |              (show user_code to user)                    |
  |                            |<-- user enters code --------|
  |                            |--- authorize ------>        |
  |                            |                            |
  |-- POST /login/oauth/access_token (poll) -->|            |
  |<- authorization_pending ---|                            |
  |-- POST (poll) ------------>|                            |
  |<- access_token ------------|                            |
  |                            |                            |
  |-- GET /user (with token) -->|                            |
  |<- user profile ------------|                            |
```

**Implementation details** (source: `internal/auth/github.go`):

- Client ID configured via `config.Auth.GitHubClientID`
- Scope: `read:user` (minimal GitHub permissions)
- HTTP client timeout: 10 seconds
- Polling interval: minimum 5 seconds (increased by 5 on `slow_down`)
- Handles all error states: `authorization_pending`, `slow_down`, `expired_token`, `access_denied`

### JWT Lifecycle

After GitHub authentication, Concord issues its own JWT tokens (source: `internal/auth/jwt.go`):

| Token | Lifetime | Issuer | Purpose |
|---|---|---|---|
| Access Token | 15 minutes | `concord` | API authentication |
| Refresh Token | 30 days | `concord-refresh` | Token renewal |

**JWT structure:**

- Algorithm: HS256 (HMAC-SHA256)
- Secret: minimum 32 characters (enforced at startup)
- Library: `github.com/golang-jwt/jwt/v5`

**Claims:**

```go
type Claims struct {
    UserID   string `json:"uid"`  // e.g. "gh_12345678"
    GitHubID int64  `json:"gid"`  // GitHub numeric ID
    Username string `json:"usr"`  // GitHub login
    jwt.RegisteredClaims         // exp, iat, iss
}
```

**Token validation:**
- Verifies HMAC signature (rejects unexpected signing methods)
- Checks expiration
- Validates issuer (`concord` for access, `concord-refresh` for refresh)

### Session Management

Sessions are stored in the local SQLite database (source: `internal/auth/repository.go`):

- Session ID: UUID v4
- User ID: deterministic from GitHub ID (`gh_{github_id}`)
- Refresh token: encrypted at rest with AES-256-GCM
- Refresh token hash: SHA-256 (for lookups without decryption)
- Expiration: 30 days
- Expired sessions are automatically cleaned

**Token refresh flow:**
1. Retrieve encrypted refresh token from DB
2. Decrypt with AES-256-GCM using key derived from JWT secret (SHA-256)
3. Validate refresh token JWT
4. Generate new access + refresh token pair
5. Return new tokens to frontend

---

## Authorization (RBAC)

Concord implements Role-Based Access Control with 4 roles and 6 permissions (source: `internal/server/permissions.go`).

### Roles (highest to lowest)

| Role | Hierarchy | Description |
|---|---|---|
| `owner` | 4 | Server creator. Full control. |
| `admin` | 3 | Trusted management. Cannot manage server settings. |
| `moderator` | 2 | Content moderation. |
| `member` | 1 | Basic participation. |

### Permission Matrix

| Permission | Owner | Admin | Moderator | Member |
|---|---|---|---|---|
| `PermManageServer` (rename/delete server) | Yes | -- | -- | -- |
| `PermManageChannels` (create/edit/delete channels) | Yes | Yes | -- | -- |
| `PermManageMembers` (kick/change roles) | Yes | Yes | -- | -- |
| `PermCreateInvite` (generate invite codes) | Yes | Yes | Yes | Yes |
| `PermSendMessages` (send text messages) | Yes | Yes | Yes | Yes |
| `PermManageMessages` (delete others' messages) | Yes | Yes | Yes | -- |

### Hierarchy Enforcement

Role modifications are strictly hierarchical:
- Cannot kick a member with equal or higher role
- Cannot promote a member to your own role level or above
- Cannot modify a member with equal or higher role
- Owner role cannot be assigned directly via role update

Permission checks use O(1) map lookups.

---

## End-to-End Encryption

All P2P communication is encrypted end-to-end using modern cryptographic primitives (source: `pkg/crypto/e2ee.go`).

### Key Exchange: X25519

- Each peer generates an X25519 key pair at startup
- Private key is clamped per the X25519 specification:
  ```
  priv[0]  &= 248
  priv[31] &= 127
  priv[31] |= 64
  ```
- Public keys are exchanged via the signaling server
- Shared secret computed: `X25519(myPrivateKey, peerPublicKey)`

### Key Derivation: HKDF-SHA256

The raw X25519 shared secret is passed through HKDF to derive a 32-byte AES key:

```
sessionKey = HKDF-SHA256(sharedSecret, salt=nil, info="concord-e2ee-v1")
```

This produces a uniformly random 256-bit key suitable for AES-GCM.

### Symmetric Encryption: AES-256-GCM

Once the session key is established:

- Algorithm: AES-256-GCM (Galois/Counter Mode)
- Nonce: 12 bytes, randomly generated per message
- Wire format: `nonce (12 bytes) || ciphertext || GCM tag (16 bytes)`
- Provides both confidentiality and integrity (AEAD)

### Per-Peer Key Management

The `E2EEManager` maintains:
- One X25519 key pair per client instance
- Per-peer public keys: `map[peerID][32]byte`
- Per-peer session keys: `map[peerID][]byte` (32-byte AES keys)
- Thread-safe with RWMutex
- Keys removed when peer disconnects

### Security Properties

- **Forward secrecy**: New key pair generated each session
- **Per-peer isolation**: Each peer pair has a unique session key
- **Authentication**: X25519 key exchange provides mutual authentication when public keys are verified via the signaling server
- **Integrity**: AES-GCM provides authenticated encryption

---

## Token Security

### Refresh Token Encryption at Rest

Refresh tokens stored in SQLite are encrypted (source: `internal/auth/service.go`):

1. Encryption key derived: `SHA-256(JWT_SECRET)` -> 32-byte key
2. Refresh token encrypted: `AES-256-GCM(refreshToken, derivedKey)`
3. Stored as base64-encoded ciphertext
4. SHA-256 hash of plaintext stored separately for lookups

### Application-Level Encryption

The `CryptoManager` (source: `internal/security/crypto.go`) provides two AEAD ciphers:

| Cipher | Use Case | Key Size | Nonce Size |
|---|---|---|---|
| AES-256-GCM | Refresh token encryption, general data | 32 bytes | 12 bytes |
| ChaCha20-Poly1305 | Alternative AEAD, mobile-friendly | 32 bytes | 12 bytes |

Both ciphers prepend the nonce to the ciphertext for self-contained decryption.

### Password Hashing

For future use, Argon2id is configured (source: `internal/security/crypto.go`):

- Algorithm: Argon2id (winner of the Password Hashing Competition)
- Parameters: `t=1, m=65536 (64 MB), p=4, keyLen=32`
- Format: `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`
- Verification uses constant-time comparison to prevent timing attacks

---

## Input Validation & Sanitization

### Validator (source: `internal/security/validation.go`)

| Validation | Rules |
|---|---|
| Username | 3-32 chars, `[a-zA-Z0-9_-]` only |
| Email | RFC 5321 max 254 chars, simple regex (avoids ReDoS) |
| URL | Max 2048 chars, `http`/`https` only, blocks private IP ranges (SSRF prevention) |
| Text Input | Valid UTF-8, no null bytes, max 10KB |
| Channel Name | 2-64 chars, `[a-zA-Z0-9 _-]` only |
| Server Name | 2-100 chars, valid UTF-8, no control characters |
| File Extension | Checked against whitelist |

**SSRF Prevention:**
URLs are validated against private IP ranges:
- `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- `169.254.0.0/16` (link-local)
- `fc00::/7` (IPv6 ULA), `fe80::/10` (IPv6 link-local)
- Loopback and link-local addresses

### Sanitizer (source: `internal/security/sanitizer.go`)

| Function | Purpose |
|---|---|
| `SanitizeHTML` | Escapes all HTML, then selectively unescapes safe tags (`b`, `i`, `u`, `em`, `strong`, `code`, `pre`). Removes all HTML attributes. Max 50KB. |
| `StripHTML` | Removes all HTML tags entirely |
| `SanitizeMarkdown` | Removes `javascript:`, `data:`, `vbscript:` URLs from markdown links. Removes `on*` event handlers. |
| `SanitizeFilename` | Removes path separators (`/`, `\`), `..`, null bytes, control characters. Max 255 chars. |
| `SanitizePath` | Removes `../`, `..\`, null bytes. Normalizes to forward slashes. |
| `SanitizeUsername` | HTML-escapes, removes control characters. Max 32 chars. |
| `SanitizeMessage` | Removes null bytes, normalizes whitespace, HTML-escapes. Max 5000 chars. |
| `RemoveControlCharacters` | Strips all control chars except `\n`, `\t`, `\r` |
| `EscapeShellArg` | Single-quote escaping for shell arguments (prefer `exec.Command` instead) |

### SQL Injection Prevention

- All database queries use parameterized queries (`?` placeholders)
- `SanitizeSQL` exists only for logging/display purposes (never for queries)
- `ContainsSQLKeywords` provides defense-in-depth detection

---

## Rate Limiting & Brute Force Protection

### Token Bucket Rate Limiter (source: `internal/security/rate_limiter.go`)

- Algorithm: Token bucket with configurable rate, interval, and capacity
- Per-key (e.g., per IP, per user) rate limiting
- Thread-safe with RWMutex
- Background cleanup of inactive buckets every 10 minutes (TTL: 1 hour)
- Supports `Allow(key)` for single requests and `AllowN(key, n)` for batch
- `WaitIfNeeded(ctx, key)` blocks until a token is available or context cancels

### Brute Force Protector (source: `internal/security/rate_limiter.go`)

- Tracks failed attempts per key
- Exponential backoff lockout: `lockoutPeriod * 2^(attempts - maxAttempts)`
- Maximum lockout: 24 hours
- Successful attempt resets the counter
- Background cleanup of stale trackers every hour (TTL: 24 hours)
- Returns `(allowed, retryAfter, error)` for informative error messages

---

## File Security

### File Scanner (source: `internal/files/scanner.go`)

Every uploaded file is validated before storage:

1. **Size check**: Maximum 50 MB (`MaxFileSize = 50 << 20`)
2. **Empty file check**: Zero-byte files are rejected
3. **Extension blocklist**: Dangerous executable extensions are blocked
4. **MIME type detection**: Content-based detection using first 512 bytes (`http.DetectContentType`)
5. **MIME whitelist**: Only approved MIME types are accepted

### Blocked Extensions

| Extension | Blocked |
|---|---|
| `.exe`, `.bat`, `.cmd`, `.com`, `.msi` | Yes |
| `.scr`, `.pif`, `.vbs`, `.ps1` | Yes |
| `.dll`, `.sys`, `.drv`, `.cpl` | Yes |
| `.inf`, `.reg` | Yes |
| `.js`, `.sh` | No (allowed for code sharing) |

### Allowed MIME Types

**Images:** `image/jpeg`, `image/png`, `image/gif`, `image/webp`, `image/svg+xml`

**Documents:** `application/pdf`, `text/plain`, `text/csv`, `text/markdown`, `application/json`

**Archives:** `application/zip`, `application/gzip`, `application/x-tar`, `application/x-7z-compressed`

**Audio:** `audio/mpeg`, `audio/ogg`, `audio/wav`, `audio/webm`, `audio/flac`

**Video:** `video/mp4`, `video/webm`, `video/ogg`

**Code:** `text/html`, `text/css`, `text/javascript`, `application/javascript`, `application/octet-stream`

### File Integrity

- Every file is hashed with SHA-256 on upload
- Every chunk is individually hashed with SHA-256
- On reassembly, full-file hash is verified against the offer hash
- Content-addressed deduplication: files with the same SHA-256 hash reuse storage

### Storage Security

- Files stored outside the database in a dedicated directory
- File permissions: `0600` (owner read/write only) for temp files
- Path traversal prevented by `SanitizeFilename` and `filepath.Base`

---

## Cryptographic Primitives

Summary of all cryptographic operations used in Concord:

| Operation | Algorithm | Key Size | Library |
|---|---|---|---|
| JWT signing | HMAC-SHA256 | 256-bit | `golang-jwt/jwt/v5` |
| Refresh token encryption | AES-256-GCM | 256-bit | `crypto/aes` + `crypto/cipher` |
| P2P key exchange | X25519 | 256-bit | `golang.org/x/crypto/curve25519` |
| P2P key derivation | HKDF-SHA256 | 256-bit | `golang.org/x/crypto/hkdf` |
| P2P message encryption | AES-256-GCM | 256-bit | `crypto/aes` + `crypto/cipher` |
| Password hashing | Argon2id | 256-bit | `golang.org/x/crypto/argon2` |
| General encryption | ChaCha20-Poly1305 | 256-bit | `golang.org/x/crypto/chacha20poly1305` |
| File integrity | SHA-256 | 256-bit | `crypto/sha256` |
| Random generation | `crypto/rand` | -- | `crypto/rand` |
| Invite code generation | `crypto/rand` + base32 | 40-bit | `crypto/rand` |

---

## OWASP Top 10 Mitigations

### A01:2021 - Broken Access Control

- RBAC with 4-tier role hierarchy
- Permission checks on every service method
- Hierarchy enforcement prevents privilege escalation
- Only server owner can delete server
- Only message author can edit (or manager can delete)

### A02:2021 - Cryptographic Failures

- All sensitive data encrypted at rest (AES-256-GCM)
- P2P messages encrypted end-to-end (X25519 + AES-256-GCM)
- JWT secret minimum 32 characters
- No plaintext storage of tokens or secrets
- Argon2id for password hashing (if needed)

### A03:2021 - Injection

- All SQL queries use parameterized statements
- HTML sanitization on user content
- Markdown sanitization removes `javascript:` URLs
- Input validation rejects control characters and null bytes

### A04:2021 - Insecure Design

- Clean Architecture enforces separation of concerns
- Domain layer has zero external dependencies
- Interfaces define contracts between layers
- Security components are reusable and testable

### A05:2021 - Security Misconfiguration

- Secure defaults for all configuration
- JWT secret length enforced
- SQLite WAL mode and FK constraints enabled by default
- Health checks validate component state

### A06:2021 - Vulnerable and Outdated Components

- `govulncheck` in CI pipeline
- Dependencies pinned in `go.mod`
- Pure Go SQLite (no CGo, smaller attack surface)

### A07:2021 - Identification and Authentication Failures

- GitHub OAuth (delegated authentication to trusted provider)
- Token bucket rate limiting
- Brute force protection with exponential backoff
- Session expiration (30 days)
- Automatic cleanup of expired sessions

### A08:2021 - Software and Data Integrity Failures

- SHA-256 file integrity verification
- Per-chunk hash verification during P2P transfer
- JWT signature verification on every token use
- Constant-time hash comparison for password verification

### A09:2021 - Security Logging and Monitoring Failures

- Structured logging with zerolog (every auth attempt, permission check, error)
- Prometheus metrics for auth, DB, P2P, cache, and HTTP
- Health checks for all infrastructure components
- Log sanitization (no tokens or secrets in logs)

### A10:2021 - Server-Side Request Forgery (SSRF)

- URL validation blocks private IP ranges
- Only `http`/`https` schemes allowed
- Loopback, link-local, and RFC 1918 addresses blocked
