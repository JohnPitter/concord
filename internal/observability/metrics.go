package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// Voice metrics
	VoiceChannelUsers      *prometheus.GaugeVec
	VoiceConnectionsTotal  *prometheus.CounterVec
	VoiceLatency           *prometheus.HistogramVec
	VoicePacketsLost       *prometheus.CounterVec
	VoiceJitterBuffer      *prometheus.HistogramVec

	// Chat metrics
	MessagesSent           *prometheus.CounterVec
	MessagesReceived       *prometheus.CounterVec
	MessageLatency         *prometheus.HistogramVec

	// P2P metrics
	P2PConnectionType      *prometheus.CounterVec
	P2PConnectionDuration  *prometheus.HistogramVec
	P2PActiveConnections   *prometheus.GaugeVec
	P2PPeersDiscovered     *prometheus.CounterVec
	P2PRelayUsage          *prometheus.CounterVec

	// File metrics
	FilesUploaded          *prometheus.CounterVec
	FilesDownloaded        *prometheus.CounterVec
	FileTransferBytes      *prometheus.CounterVec
	FileTransferDuration   *prometheus.HistogramVec

	// Translation metrics
	TranslationRequests    *prometheus.CounterVec
	TranslationLatency     *prometheus.HistogramVec
	TranslationErrors      *prometheus.CounterVec
	TranslationCacheHits   *prometheus.CounterVec

	// Server metrics
	ServersCreated         *prometheus.CounterVec
	ServersActive          *prometheus.GaugeVec
	ServerMembers          *prometheus.GaugeVec

	// Auth metrics
	AuthAttempts           *prometheus.CounterVec
	AuthSuccessful         *prometheus.CounterVec
	AuthFailed             *prometheus.CounterVec
	ActiveSessions         *prometheus.GaugeVec

	// Database metrics
	DBQueryDuration        *prometheus.HistogramVec
	DBConnections          *prometheus.GaugeVec
	DBErrors               *prometheus.CounterVec

	// Cache metrics
	CacheHits              *prometheus.CounterVec
	CacheMisses            *prometheus.CounterVec
	CacheEvictions         *prometheus.CounterVec
	CacheSize              *prometheus.GaugeVec

	// HTTP metrics (for server mode)
	HTTPRequestsTotal      *prometheus.CounterVec
	HTTPRequestDuration    *prometheus.HistogramVec
	HTTPResponseSize       *prometheus.HistogramVec
}

// NewMetrics creates and registers all Prometheus metrics
// All metrics follow naming conventions: concord_<subsystem>_<metric>_<unit>
// Complexity: O(1)
func NewMetrics() *Metrics {
	m := &Metrics{
		// Voice metrics
		VoiceChannelUsers: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_voice_channel_users",
				Help: "Number of users currently in each voice channel",
			},
			[]string{"channel_id", "server_id"},
		),

		VoiceConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_voice_connections_total",
				Help: "Total number of voice connections established",
			},
			[]string{"server_id", "status"}, // status: success, failed
		),

		VoiceLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_voice_latency_milliseconds",
				Help:    "Voice connection latency in milliseconds",
				Buckets: []float64{10, 25, 50, 100, 200, 500, 1000},
			},
			[]string{"channel_id"},
		),

		VoicePacketsLost: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_voice_packets_lost_total",
				Help: "Total number of voice packets lost",
			},
			[]string{"channel_id", "peer_id"},
		),

		VoiceJitterBuffer: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_voice_jitter_buffer_milliseconds",
				Help:    "Jitter buffer size in milliseconds",
				Buckets: []float64{20, 50, 100, 150, 200},
			},
			[]string{"channel_id"},
		),

		// Chat metrics
		MessagesSent: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_messages_sent_total",
				Help: "Total number of messages sent",
			},
			[]string{"server_id", "channel_id", "type"}, // type: text, file, system
		),

		MessagesReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_messages_received_total",
				Help: "Total number of messages received",
			},
			[]string{"server_id", "channel_id", "type"},
		),

		MessageLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_message_latency_milliseconds",
				Help:    "Message delivery latency in milliseconds",
				Buckets: []float64{10, 50, 100, 250, 500, 1000},
			},
			[]string{"channel_id"},
		),

		// P2P metrics
		P2PConnectionType: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_p2p_connection_type_total",
				Help: "Total P2P connections by type",
			},
			[]string{"type"}, // type: direct, hole_punch, relay
		),

		P2PConnectionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_p2p_connection_duration_seconds",
				Help:    "Duration of P2P connections in seconds",
				Buckets: []float64{60, 300, 600, 1800, 3600, 7200},
			},
			[]string{"type"},
		),

		P2PActiveConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_p2p_active_connections",
				Help: "Number of active P2P connections",
			},
			[]string{"type"},
		),

		P2PPeersDiscovered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_p2p_peers_discovered_total",
				Help: "Total number of peers discovered",
			},
			[]string{"discovery_method"}, // mdns, dht, bootstrap
		),

		P2PRelayUsage: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_p2p_relay_usage_total",
				Help: "Total number of times relay was used",
			},
			[]string{"reason"}, // nat_traversal_failed, timeout, etc.
		),

		// File metrics
		FilesUploaded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_files_uploaded_total",
				Help: "Total number of files uploaded",
			},
			[]string{"server_id", "channel_id"},
		),

		FilesDownloaded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_files_downloaded_total",
				Help: "Total number of files downloaded",
			},
			[]string{"server_id", "channel_id"},
		),

		FileTransferBytes: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_file_transfer_bytes_total",
				Help: "Total bytes transferred for files",
			},
			[]string{"direction"}, // upload, download
		),

		FileTransferDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_file_transfer_duration_seconds",
				Help:    "File transfer duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
			},
			[]string{"direction"},
		),

		// Translation metrics
		TranslationRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_translation_requests_total",
				Help: "Total number of translation requests",
			},
			[]string{"lang_pair", "status"}, // status: success, failed, cached
		),

		TranslationLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_translation_latency_milliseconds",
				Help:    "Translation request latency in milliseconds",
				Buckets: []float64{50, 100, 170, 250, 500, 1000, 2000},
			},
			[]string{"lang_pair"},
		),

		TranslationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_translation_errors_total",
				Help: "Total number of translation errors",
			},
			[]string{"error_type"},
		),

		TranslationCacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_translation_cache_hits_total",
				Help: "Total number of translation cache hits",
			},
			[]string{"lang_pair"},
		),

		// Server metrics
		ServersCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_servers_created_total",
				Help: "Total number of servers created",
			},
			[]string{"user_id"},
		),

		ServersActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_servers_active",
				Help: "Number of active servers",
			},
			[]string{},
		),

		ServerMembers: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_server_members",
				Help: "Number of members in each server",
			},
			[]string{"server_id"},
		),

		// Auth metrics
		AuthAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_auth_attempts_total",
				Help: "Total number of authentication attempts",
			},
			[]string{"method"}, // github, token_refresh
		),

		AuthSuccessful: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_auth_successful_total",
				Help: "Total number of successful authentications",
			},
			[]string{"method"},
		),

		AuthFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_auth_failed_total",
				Help: "Total number of failed authentications",
			},
			[]string{"method", "reason"},
		),

		ActiveSessions: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_active_sessions",
				Help: "Number of active user sessions",
			},
			[]string{},
		),

		// Database metrics
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_db_query_duration_milliseconds",
				Help:    "Database query duration in milliseconds",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
			},
			[]string{"operation", "table"},
		),

		DBConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_db_connections",
				Help: "Number of database connections",
			},
			[]string{"state"}, // idle, in_use, open
		),

		DBErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"operation", "error_type"},
		),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type"}, // lru, redis
		),

		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type"},
		),

		CacheEvictions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"cache_type", "reason"}, // reason: size, ttl
		),

		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "concord_cache_size_entries",
				Help: "Current number of entries in cache",
			},
			[]string{"cache_type"},
		),

		// HTTP metrics (server mode)
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "concord_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_http_request_duration_milliseconds",
				Help:    "HTTP request duration in milliseconds",
				Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000},
			},
			[]string{"method", "path"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "concord_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path"},
		),
	}

	return m
}
