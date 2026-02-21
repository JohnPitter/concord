# Concord REST API Reference

> Base URL: `http://{host}:{port}/api/v1`

All endpoints return JSON. Errors use standard HTTP status codes with a JSON body:

```json
{
  "error": "description of what went wrong"
}
```

---

## Table of Contents

- [Health & Metrics](#health--metrics)
- [Authentication](#authentication)
- [Servers](#servers)
- [Channels](#channels)
- [Members](#members)
- [Invites](#invites)
- [Messages](#messages)
- [WebSocket](#websocket)

---

## Health & Metrics

### `GET /health`

Returns the health status of the application and all registered components.

**Auth required:** No

**Response** `200 OK`:

```json
{
  "status": "healthy",
  "timestamp": "2026-02-20T12:00:00Z",
  "components": {
    "sqlite": {
      "name": "sqlite",
      "status": "healthy",
      "message": "OK",
      "timestamp": "2026-02-20T12:00:00Z",
      "duration_ms": 1200000
    }
  },
  "version": "0.1.0",
  "uptime_seconds": 3600000000000
}
```

**Status values:** `healthy`, `degraded`, `unhealthy`, `unknown`

---

### `GET /metrics`

Exposes Prometheus metrics in text exposition format.

**Auth required:** No

**Response** `200 OK`: Prometheus text format

Metric namespaces include:

| Namespace | Description |
|---|---|
| `concord_voice_*` | Voice channel users, connections, latency, packets lost, jitter buffer |
| `concord_messages_*` | Messages sent/received, delivery latency |
| `concord_p2p_*` | Connection type, duration, active connections, peers discovered, relay usage |
| `concord_files_*` | Uploads, downloads, transfer bytes, transfer duration |
| `concord_translation_*` | Requests, latency, errors, cache hits |
| `concord_servers_*` | Created, active, members |
| `concord_auth_*` | Attempts, successful, failed, active sessions |
| `concord_db_*` | Query duration, connections, errors |
| `concord_cache_*` | Hits, misses, evictions, size |
| `concord_http_*` | Requests total, request duration, response size |

---

## Authentication

Concord uses the GitHub Device Flow (RFC 8628) for authentication. The flow is:

1. Client calls `POST /api/v1/auth/device-code` to get a device code and user code.
2. User navigates to `https://github.com/login/device` and enters the user code.
3. Client polls `POST /api/v1/auth/token` with the device code until authorization completes.
4. Client receives a JWT access token (15 min) and refresh token (30 days).
5. When the access token expires, client calls `POST /api/v1/auth/refresh`.

### `POST /api/v1/auth/device-code`

Initiates the GitHub Device Flow. Returns codes for the user to authorize.

**Auth required:** No

**Request body:** None

**Response** `200 OK`:

```json
{
  "device_code": "3584d83530557fdd1f46af8289938c8ef79f9dc5",
  "user_code": "WDJB-MJHT",
  "verification_uri": "https://github.com/login/device",
  "expires_in": 899,
  "interval": 5
}
```

| Field | Type | Description |
|---|---|---|
| `device_code` | string | Code used to poll for the token (do not show to user) |
| `user_code` | string | Code the user enters on GitHub |
| `verification_uri` | string | URL the user should visit |
| `expires_in` | int | Seconds until the device code expires |
| `interval` | int | Minimum polling interval in seconds |

**Error codes:**

| Status | Cause |
|---|---|
| 500 | Failed to contact GitHub API |

---

### `POST /api/v1/auth/token`

Polls for the access token after the user authorizes on GitHub. Creates or updates the local user and session.

**Auth required:** No

**Request body:**

```json
{
  "device_code": "3584d83530557fdd1f46af8289938c8ef79f9dc5",
  "interval": 5
}
```

**Response** `200 OK`:

```json
{
  "authenticated": true,
  "user": {
    "id": "gh_12345678",
    "github_id": 12345678,
    "username": "octocat",
    "display_name": "The Octocat",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
    "created_at": "2026-02-20T12:00:00Z",
    "updated_at": "2026-02-20T12:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": 1740060900
}
```

**JWT claims structure:**

| Claim | Type | Description |
|---|---|---|
| `uid` | string | Concord user ID (`gh_{github_id}`) |
| `gid` | int64 | GitHub user ID |
| `usr` | string | GitHub username |
| `iss` | string | `concord` (access) or `concord-refresh` (refresh) |
| `exp` | int64 | Expiration timestamp |
| `iat` | int64 | Issued-at timestamp |

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Missing device_code |
| 403 | User denied authorization |
| 408 | Device code expired |
| 500 | Internal error (GitHub API, DB, JWT) |

---

### `POST /api/v1/auth/refresh`

Refreshes an expired access token using a valid refresh token. Generates a new token pair.

**Auth required:** No (uses refresh token)

**Request body:**

```json
{
  "user_id": "gh_12345678"
}
```

**Response** `200 OK`:

```json
{
  "authenticated": true,
  "user": {
    "id": "gh_12345678",
    "github_id": 12345678,
    "username": "octocat"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": 1740060900
}
```

**Response** `200 OK` (session expired):

```json
{
  "authenticated": false
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Missing user_id |
| 401 | Refresh token expired or invalid |
| 500 | Decryption or DB error |

---

## Servers

### `POST /api/v1/servers`

Creates a new server. The creator becomes the owner. A default `#general` text channel and a `General` voice channel are created automatically.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "name": "My Gaming Server",
  "owner_id": "gh_12345678"
}
```

**Validation:**
- `name` is required, trimmed, max 100 characters

**Response** `201 Created`:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Gaming Server",
  "icon_url": "",
  "owner_id": "gh_12345678",
  "invite_code": "abcdefgh",
  "created_at": "2026-02-20T12:00:00Z"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty name, name too long |
| 401 | Not authenticated |
| 500 | DB error |

---

### `GET /api/v1/servers`

Returns all servers the authenticated user belongs to.

**Auth required:** Yes (Bearer token)

**Query parameters:**

| Param | Type | Description |
|---|---|---|
| `user_id` | string | User ID to list servers for |

**Response** `200 OK`:

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Gaming Server",
    "icon_url": "",
    "owner_id": "gh_12345678",
    "invite_code": "abcdefgh",
    "created_at": "2026-02-20T12:00:00Z"
  }
]
```

---

### `GET /api/v1/servers/{id}`

Returns a single server by ID.

**Auth required:** Yes (Bearer token)

**Response** `200 OK`:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Gaming Server",
  "icon_url": "",
  "owner_id": "gh_12345678",
  "invite_code": "abcdefgh",
  "created_at": "2026-02-20T12:00:00Z"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 404 | Server not found |

---

### `PUT /api/v1/servers/{id}`

Updates a server's name and icon. Requires `PermManageServer` (owner only).

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "user_id": "gh_12345678",
  "name": "Renamed Server",
  "icon_url": "https://example.com/icon.png"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty name |
| 403 | Insufficient permissions |
| 404 | Server not found |

---

### `DELETE /api/v1/servers/{id}`

Deletes a server. Only the server owner can delete.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "user_id": "gh_12345678"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 403 | Not the server owner |
| 404 | Server not found |

---

## Channels

### `POST /api/v1/servers/{id}/channels`

Creates a new channel within a server. Requires `PermManageChannels` (owner or admin).

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "user_id": "gh_12345678",
  "name": "announcements",
  "type": "text"
}
```

**Validation:**
- `name` is required, trimmed
- `type` must be `"text"` or `"voice"`

**Response** `201 Created`:

```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "server_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "announcements",
  "type": "text",
  "position": 0,
  "created_at": "2026-02-20T12:00:00Z"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty name, invalid type |
| 403 | Insufficient permissions |

---

### `GET /api/v1/servers/{id}/channels`

Returns all channels for a server, ordered by position then creation date.

**Auth required:** Yes (Bearer token)

**Response** `200 OK`:

```json
[
  {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "server_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "general",
    "type": "text",
    "position": 0,
    "created_at": "2026-02-20T12:00:00Z"
  },
  {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "server_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "General",
    "type": "voice",
    "position": 1,
    "created_at": "2026-02-20T12:00:00Z"
  }
]
```

---

### `DELETE /api/v1/servers/{id}/channels/{channelId}`

Deletes a channel. Requires `PermManageChannels`.

**Auth required:** Yes (Bearer token)

**Error codes:**

| Status | Cause |
|---|---|
| 403 | Insufficient permissions |
| 404 | Channel not found |

---

## Members

### `GET /api/v1/servers/{id}/members`

Returns all members of a server with their user info. Members are ordered by role hierarchy (owner first) then join date.

**Auth required:** Yes (Bearer token)

**Response** `200 OK`:

```json
[
  {
    "server_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "gh_12345678",
    "username": "octocat",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
    "role": "owner",
    "joined_at": "2026-02-20T12:00:00Z"
  },
  {
    "server_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "gh_87654321",
    "username": "contributor",
    "avatar_url": "",
    "role": "member",
    "joined_at": "2026-02-20T13:00:00Z"
  }
]
```

**Roles and hierarchy** (highest to lowest):

| Role | Hierarchy | Permissions |
|---|---|---|
| `owner` | 4 | All permissions |
| `admin` | 3 | ManageChannels, ManageMembers, CreateInvite, SendMessages, ManageMessages |
| `moderator` | 2 | CreateInvite, SendMessages, ManageMessages |
| `member` | 1 | CreateInvite, SendMessages |

---

### `PUT /api/v1/servers/{id}/members/{userId}/role`

Changes a member's role. Requires `PermManageMembers`. Cannot promote above your own role or modify someone with equal/higher role.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "actor_id": "gh_12345678",
  "role": "moderator"
}
```

**Validation:**
- Cannot assign `owner` role directly
- Cannot promote to your own role level or above
- Cannot modify a member with equal or higher role

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Attempting to assign owner role |
| 403 | Insufficient permissions or hierarchy violation |
| 404 | Member not found |

---

### `DELETE /api/v1/servers/{id}/members/{userId}`

Kicks a member from the server. Requires `PermManageMembers`. Cannot kick someone with equal or higher role.

**Auth required:** Yes (Bearer token)

**Error codes:**

| Status | Cause |
|---|---|
| 403 | Insufficient permissions or hierarchy violation |
| 404 | Member not found |

---

## Invites

### `POST /api/v1/servers/{id}/invite`

Generates a new invite code for a server. Requires `PermCreateInvite` (all roles). The new code replaces the existing invite code.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "user_id": "gh_12345678"
}
```

**Response** `200 OK`:

```json
{
  "invite_code": "abcdefgh"
}
```

Invite codes are 8-character lowercase base32 strings generated from 5 random bytes.

**Error codes:**

| Status | Cause |
|---|---|
| 403 | Not a member |

---

### `GET /api/v1/invite/{code}`

Returns information about a server from its invite code, without joining.

**Auth required:** No

**Response** `200 OK`:

```json
{
  "server_id": "550e8400-e29b-41d4-a716-446655440000",
  "server_name": "My Gaming Server",
  "invite_code": "abcdefgh",
  "member_count": 42
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 404 | Invalid invite code |

---

### `POST /api/v1/invite/{code}/redeem`

Joins a server using an invite code. If the user is already a member, returns the server without changes.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "user_id": "gh_12345678"
}
```

**Response** `200 OK`:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "My Gaming Server",
  "icon_url": "",
  "owner_id": "gh_12345678",
  "invite_code": "abcdefgh",
  "created_at": "2026-02-20T12:00:00Z"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 404 | Invalid invite code |

---

## Messages

### `POST /api/v1/channels/{id}/messages`

Sends a text message to a channel.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "author_id": "gh_12345678",
  "content": "Hello, world!"
}
```

**Validation:**
- `content` is required, trimmed, max 4000 characters
- Empty content after trimming is rejected

**Response** `201 Created`:

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440003",
  "channel_id": "660e8400-e29b-41d4-a716-446655440001",
  "author_id": "gh_12345678",
  "content": "Hello, world!",
  "type": "text",
  "edited_at": null,
  "created_at": "2026-02-20T12:00:00Z",
  "author_name": "octocat",
  "author_avatar": "https://avatars.githubusercontent.com/u/12345678?v=4"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty content, content too long |
| 401 | Not authenticated |

---

### `GET /api/v1/channels/{id}/messages`

Retrieves messages for a channel with cursor-based pagination. Messages are returned in reverse chronological order (newest first) unless using `after`.

**Auth required:** Yes (Bearer token)

**Query parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `before` | string | (none) | Message ID -- load messages older than this message |
| `after` | string | (none) | Message ID -- load messages newer than this message |
| `limit` | int | 50 | Max messages to return (1-100) |

**Pagination behavior:**

- No `before`/`after`: returns the most recent messages (DESC order)
- `before={id}`: returns messages older than the given message (DESC order)
- `after={id}`: returns messages newer than the given message (ASC order)

**Response** `200 OK`:

```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440003",
    "channel_id": "660e8400-e29b-41d4-a716-446655440001",
    "author_id": "gh_12345678",
    "content": "Hello, world!",
    "type": "text",
    "edited_at": null,
    "created_at": "2026-02-20T12:00:00Z",
    "author_name": "octocat",
    "author_avatar": "https://avatars.githubusercontent.com/u/12345678?v=4"
  }
]
```

---

### `PUT /api/v1/channels/{id}/messages/{messageId}`

Edits a message. Only the original author can edit.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "author_id": "gh_12345678",
  "content": "Hello, updated world!"
}
```

**Response** `200 OK`:

```json
{
  "id": "770e8400-e29b-41d4-a716-446655440003",
  "channel_id": "660e8400-e29b-41d4-a716-446655440001",
  "author_id": "gh_12345678",
  "content": "Hello, updated world!",
  "type": "text",
  "edited_at": "2026-02-20T12:05:00Z",
  "created_at": "2026-02-20T12:00:00Z",
  "author_name": "octocat",
  "author_avatar": "https://avatars.githubusercontent.com/u/12345678?v=4"
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty content, content too long |
| 403 | Not the message author |
| 404 | Message not found |

---

### `DELETE /api/v1/channels/{id}/messages/{messageId}`

Deletes a message. The author or a user with `PermManageMessages` (moderator+) can delete.

**Auth required:** Yes (Bearer token)

**Request body:**

```json
{
  "actor_id": "gh_12345678",
  "is_manager": false
}
```

**Error codes:**

| Status | Cause |
|---|---|
| 403 | Not the author and not a manager |
| 404 | Message not found |

---

### `GET /api/v1/channels/{id}/messages/search`

Performs full-text search within a channel using SQLite FTS5.

**Auth required:** Yes (Bearer token)

**Query parameters:**

| Param | Type | Default | Description |
|---|---|---|---|
| `q` | string | (required) | Search query |
| `limit` | int | 20 | Max results (1-50) |

**Response** `200 OK`:

```json
[
  {
    "id": "770e8400-e29b-41d4-a716-446655440003",
    "channel_id": "660e8400-e29b-41d4-a716-446655440001",
    "author_id": "gh_12345678",
    "content": "Hello, world!",
    "type": "text",
    "edited_at": null,
    "created_at": "2026-02-20T12:00:00Z",
    "author_name": "octocat",
    "author_avatar": "",
    "snippet": "...the <mark>Hello</mark>, world! message..."
  }
]
```

The `snippet` field contains highlighted matches with `<mark>` tags.

**Error codes:**

| Status | Cause |
|---|---|
| 400 | Empty query |

---

## WebSocket

### `WS /api/v1/ws`

WebSocket endpoint for real-time signaling. Used for peer discovery, presence updates, and voice channel coordination.

**Auth required:** Yes (token in query string or header)

**Connection:** `ws://{host}:{port}/api/v1/ws`

All messages are JSON-encoded `Signal` envelopes:

```json
{
  "type": "join",
  "from": "peer_id_abc",
  "to": "",
  "server_id": "server_uuid",
  "channel_id": "channel_uuid",
  "payload": { ... }
}
```

### Signal Types

| Type | Direction | Description |
|---|---|---|
| `join` | Client -> Server | Peer joins a server/channel |
| `leave` | Client -> Server | Peer leaves |
| `offer` | Client -> Server -> Client | Connection offer (addresses + public key) |
| `answer` | Client -> Server -> Client | Connection answer |
| `peer_list` | Server -> Client | Current peers in the channel |
| `peer_joined` | Server -> Client | New peer notification (broadcast) |
| `peer_left` | Server -> Client | Peer departed notification (broadcast) |
| `error` | Server -> Client | Error message |

### Join Payload

```json
{
  "user_id": "gh_12345678",
  "peer_id": "12D3KooW...",
  "addresses": ["/ip4/192.168.1.5/tcp/4001/p2p/12D3KooW..."],
  "public_key": "<base64-encoded X25519 public key>"
}
```

### Offer / Answer Payload

```json
{
  "peer_id": "12D3KooW...",
  "addresses": ["/ip4/192.168.1.5/tcp/4001/p2p/12D3KooW..."],
  "public_key": "<base64-encoded X25519 public key>"
}
```

### Peer List Payload

```json
{
  "peers": [
    {
      "user_id": "gh_87654321",
      "peer_id": "12D3KooX...",
      "addresses": ["/ip4/10.0.0.2/tcp/4001/p2p/12D3KooX..."],
      "public_key": "<base64>"
    }
  ]
}
```

### Error Payload

```json
{
  "code": 400,
  "message": "invalid join payload"
}
```

### Connection Flow

1. Client opens WebSocket to `/api/v1/ws`
2. Client sends `join` signal with peer ID, addresses, and E2EE public key
3. Server responds with `peer_list` of existing peers
4. Server broadcasts `peer_joined` to other peers in the channel
5. Peers exchange `offer`/`answer` to establish direct P2P connections
6. On disconnect, server broadcasts `peer_left`
