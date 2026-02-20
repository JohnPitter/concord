package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetrics(t *testing.T) {
	metrics := NewMetrics()
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.MessagesSent)
	assert.NotNil(t, metrics.MessagesReceived)
	assert.NotNil(t, metrics.VoiceLatency)
	assert.NotNil(t, metrics.MessageLatency)
	assert.NotNil(t, metrics.HTTPRequestsTotal)
	assert.NotNil(t, metrics.HTTPRequestDuration)
	assert.NotNil(t, metrics.P2PActiveConnections)
	assert.NotNil(t, metrics.FilesUploaded)
	assert.NotNil(t, metrics.FilesDownloaded)
}

func TestMetrics_IncrementMessagesSent(t *testing.T) {
	metrics := NewMetrics()

	metrics.MessagesSent.WithLabelValues("server-1", "channel-1", "text").Inc()
	metrics.MessagesSent.WithLabelValues("server-1", "channel-2", "text").Inc()
}

func TestMetrics_RecordVoiceLatency(t *testing.T) {
	metrics := NewMetrics()

	metrics.VoiceLatency.WithLabelValues("channel-1").Observe(50.0)
	metrics.VoiceLatency.WithLabelValues("channel-2").Observe(25.0)
}

func TestMetrics_SetActiveP2PConnections(t *testing.T) {
	metrics := NewMetrics()

	metrics.P2PActiveConnections.WithLabelValues("direct").Set(42)
	metrics.P2PActiveConnections.WithLabelValues("relay").Set(15)
}

func TestMetrics_RecordHTTPRequest(t *testing.T) {
	metrics := NewMetrics()

	metrics.HTTPRequestsTotal.WithLabelValues("POST", "/api/messages", "200").Inc()
	metrics.HTTPRequestDuration.WithLabelValues("POST", "/api/messages").Observe(100.0)
}
