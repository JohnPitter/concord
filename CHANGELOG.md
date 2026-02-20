# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

#### Phase 1: Foundation (2026-02-20)

- Project scaffolding with Wails v2 + Go 1.23
- Configuration system with environment variable support
  - JSON-based configuration with sensible defaults
  - Per-OS default paths (AppData/Library/XDG)
  - Full configuration validation
- Logging infrastructure with zerolog
  - Structured JSON logging
  - Performance logging utilities
  - Context-aware logger middleware
  - Automatic sensitive data sanitization
- Observability infrastructure
  - Prometheus metrics for all major subsystems (voice, chat, P2P, files, translation)
  - Health check system with component-level monitoring
  - Automatic health caching with configurable TTL
- SQLite database layer
  - Pure Go SQLite driver (modernc.org/sqlite, no CGo)
  - WAL mode for improved concurrency
  - Automatic migration system with embedded SQL files
  - Database utilities (vacuum, optimize, backup, integrity check)
  - Transaction helpers with automatic rollback
- Initial database schema (migration 001_init)
  - Users, servers, channels, messages tables
  - Full-text search for messages (FTS5)
  - File attachments support
  - Direct messages
  - Server member roles and permissions
  - Voice channel tracking
  - Authentication sessions
  - Server invites
  - User settings and preferences
  - P2P peer caching
  - Translation result caching
- Desktop application entry point (cmd/concord/main.go)
  - Wails integration
  - Application lifecycle management
  - Health check registration
  - Database initialization and migration
- Central server entry point stub (cmd/server/main.go)
  - Graceful shutdown handling
  - Signal handling (SIGINT, SIGTERM)
- Version management system
- Makefile with comprehensive build, test, and development commands
- Project documentation
  - README.md
  - ARCHITECTURE.md (comprehensive technical specification)
  - LICENSE (MIT)

### Security

- JWT secret validation (minimum 32 chars in production)
- SQLite database encryption support
- Rate limiting configuration
- File upload validation and size limits
- Sensitive data sanitization in logs
- Foreign key constraints enforcement
- Prepared statements for SQL injection prevention

### Performance

- O(1) cache operations with LRU cache
- O(log n) database queries with proper indexing
- Connection pooling for database connections
- WAL mode for SQLite (improved concurrency)
- Memory-mapped I/O for SQLite (30GB mmap_size)
- 64MB SQLite cache
- Optimized pragmas for performance

## [0.1.0] - TBD

Initial development release - in progress.

### Planned Features

- Phase 2: GitHub OAuth authentication
- Phase 3: Server management (CRUD, channels, members, invites)
- Phase 4: Real-time text chat with WebSocket
- Phase 5: P2P networking with libp2p
- Phase 6: Voice chat with WebRTC
- Phase 7: File sharing
- Phase 8: Voice translation with NVIDIA PersonaPlex
- Phase 9: Central server (PostgreSQL, Redis, REST API)
- Phase 10: Production hardening and release

---

## Version History

- **0.1.0-dev** - Current development version
