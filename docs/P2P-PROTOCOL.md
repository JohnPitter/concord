# Concord P2P Protocol

This document describes the peer-to-peer networking protocol used by Concord for direct client communication, signaling, and NAT traversal.

---

## Table of Contents

- [Overview](#overview)
- [Signaling Layer](#signaling-layer)
- [P2P Transport](#p2p-transport)
- [NAT Traversal](#nat-traversal)
- [Wire Format](#wire-format)
- [Connection Lifecycle](#connection-lifecycle)
- [Security](#security)

---

## Overview

Concord uses a hybrid architecture:

1. **Central Server** — Handles authentication, server/channel metadata, message persistence, and signaling coordination.
2. **P2P Network** — Handles real-time voice streaming, direct file transfers, and presence updates between connected peers.

The P2P layer is built on **libp2p** for data transport and **Pion WebRTC v4** for voice media.

### Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Signaling | gorilla/websocket | Coordinate peer connections |
| Data Transport | libp2p | Stream multiplexing, peer discovery |
| Voice Media | Pion WebRTC | RTP/RTCP, DTLS-SRTP |
| NAT Traversal | libp2p relay + hole punching | Connect peers behind NAT |

---

## Signaling Layer

### WebSocket Connection

Clients connect to the signaling server at `ws://{host}:{port}/api/v1/ws`.

**Handshake:**
1. Client opens WebSocket with JWT in `Authorization` header
2. Server validates token, registers peer
3. Bidirectional message flow begins

### Message Types

All signaling messages are JSON:

```json
{
  "type": "offer|answer|ice-candidate|join|leave|ping|pong",
  "from": "peer-id",
  "to": "target-peer-id",
  "payload": { ... }
}
```

| Type | Direction | Description |
|------|-----------|-------------|
| `join` | Client → Server | Join a channel/room |
| `leave` | Client → Server | Leave a channel/room |
| `offer` | Client → Server → Client | WebRTC SDP offer |
| `answer` | Client → Server → Client | WebRTC SDP answer |
| `ice-candidate` | Client → Server → Client | ICE candidate exchange |
| `ping` / `pong` | Bidirectional | Keepalive (30s interval) |

### Reconnection

- Client implements exponential backoff: 1s, 2s, 4s, 8s, max 30s
- On reconnect, client re-sends `join` for active channels
- Server cleans up stale peers after 60s without `pong`

---

## P2P Transport

### libp2p Host

Each client runs a libp2p host (`internal/network/p2p/host.go`):

- **Identity**: Ed25519 keypair (generated on first launch, stored locally)
- **Listen addresses**: `/ip4/0.0.0.0/tcp/0`, `/ip4/0.0.0.0/udp/0/quic-v1`
- **Protocols**: `/concord/data/1.0.0` (custom stream protocol)
- **Peer discovery**: mDNS (LAN), DHT bootstrap nodes (WAN)

### Stream Protocol

The `/concord/data/1.0.0` protocol uses length-prefixed binary frames:

```
[4 bytes: frame length (big-endian uint32)]
[1 byte: message type]
[N bytes: protobuf or JSON payload]
```

Message types:

| Type | Value | Description |
|------|-------|-------------|
| `DATA` | 0x01 | Generic data payload |
| `FILE_CHUNK` | 0x02 | File transfer chunk |
| `FILE_META` | 0x03 | File metadata (name, size, hash) |
| `PRESENCE` | 0x04 | Online/offline status |
| `ACK` | 0x05 | Acknowledgment |

---

## NAT Traversal

### Strategy (in order)

1. **Direct connection** — If peers are on the same LAN (mDNS discovery)
2. **Hole punching** — libp2p's `holepunch` protocol for symmetric NAT
3. **Relay** — libp2p circuit relay v2 through a public relay node
4. **TURN** — Coturn server for WebRTC voice media

### TURN/STUN Configuration

Voice channels use Pion WebRTC with ICE servers:

```json
{
  "ice_servers": [
    { "urls": ["stun:stun.l.google.com:19302"] },
    { "urls": ["turn:{host}:3478"], "username": "...", "credential": "..." }
  ]
}
```

The relay server is deployed via `deployments/docker/Dockerfile.relay` (Coturn).

---

## Connection Lifecycle

```
Client A                    Signaling Server                    Client B
   |                              |                                |
   |--- join(channel) ----------->|                                |
   |                              |<---------- join(channel) ------|
   |                              |                                |
   |<-- peer-list ----------------|                                |
   |                              |                                |
   |--- offer(SDP) -------------->|--- offer(SDP) --------------->|
   |                              |                                |
   |                              |<---------- answer(SDP) --------|
   |<-- answer(SDP) --------------|                                |
   |                              |                                |
   |--- ice-candidate ----------->|--- ice-candidate ------------>|
   |<-- ice-candidate ------------|<---------- ice-candidate ------|
   |                              |                                |
   |================ P2P / WebRTC connection established ==========|
   |                              |                                |
   |<-------------- voice/data stream (direct P2P) -------------->|
```

---

## Security

### Transport Security

- **libp2p**: TLS 1.3 over TCP, Noise protocol over QUIC
- **WebRTC**: DTLS-SRTP (mandatory, negotiated during ICE)
- **Signaling**: WSS (TLS) in production

### E2E Encryption

For text messages and file transfers over the P2P data channel:

1. Each user generates an X25519 keypair on registration
2. Shared secret derived via ECDH
3. Messages encrypted with AES-256-GCM using derived key
4. Key rotation on each session reconnect

### Peer Authentication

- Peers are identified by their libp2p peer ID (derived from Ed25519 public key)
- The signaling server verifies JWT before forwarding offers
- Peer IDs are bound to user accounts in the server database
