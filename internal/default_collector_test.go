package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// mockMessage implements mqtt.Message for testing
type mockMessage struct {
	payload []byte
	topic   string
}

func (m *mockMessage) Duplicate() bool { return false }
func (m *mockMessage) Qos() byte       { return 0 }
func (m *mockMessage) Retained() bool  { return false }
func (m *mockMessage) Topic() string   { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte { return m.payload }
func (m *mockMessage) Ack()            {}

func TestNewDefaultCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 4, len(collector.descriptions))
}

func TestDefaultCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)

	descriptions := make(chan *prometheus.Desc)
	go func() {
		collector.Describe(descriptions)
		close(descriptions)
	}()

	count := 0
	for range descriptions {
		count++
	}

	assert.Equal(t, 4, count)
}

func TestDefaultCollector_Collect(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)

	// Set some test values
	collector.Metrics.uptime = 123.45
	collector.Metrics.version = "2.0.15"
	collector.Metrics.subscriptions = 10
	collector.Metrics.sharedSubscriptions = 5

	metrics := make(chan prometheus.Metric)
	go func() {
		collector.Collect(metrics)
		close(metrics)
	}()

	count := 0
	for range metrics {
		count++
	}

	assert.Equal(t, 4, count)
}

func TestDefaultCollector_UptimeHandler(t *testing.T) {
	// Test the parsing logic directly
	payload := []byte("12345 seconds")
	uptime, _ := strconv.Atoi(strings.Split(string(payload), " ")[0])
	assert.Equal(t, 12345, uptime)
}

func TestDefaultCollector_VersionHandler(t *testing.T) {
	// Test the parsing logic directly
	payload := []byte("mosquitto version 2.0.15")
	version := strings.Split(string(payload), " ")[2]
	assert.Equal(t, "2.0.15", version)
}

func TestDefaultCollector_SubscriptionsHandler(t *testing.T) {
	// Test the parsing logic directly
	payload := []byte("42")
	num, _ := strconv.Atoi(string(payload))
	assert.Equal(t, 42, num)
}

func TestDefaultCollector_SharedSubscriptionsHandler(t *testing.T) {
	// Test the parsing logic directly
	payload := []byte("24")
	num, _ := strconv.Atoi(string(payload))
	assert.Equal(t, 24, num)
}

func TestDefaultCollector_UptimeHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)
	
	// Create mock message with uptime payload
	msg := &mockMessage{payload: []byte("12345 seconds")}
	
	// Call the handler directly
	collector.uptimeHandler(nil, msg)
	
	// Verify the metric was updated
	assert.Equal(t, float64(12345), collector.Metrics.uptime)
}

func TestDefaultCollector_VersionHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)
	
	msg := &mockMessage{payload: []byte("mosquitto version 2.0.15")}
	collector.versionHandler(nil, msg)
	
	assert.Equal(t, "2.0.15", collector.Metrics.version)
}

func TestDefaultCollector_SubscriptionsHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)
	
	msg := &mockMessage{payload: []byte("42")}
	collector.subscriptionsHandler(nil, msg)
	
	assert.Equal(t, float64(42), collector.Metrics.subscriptions)
}

func TestDefaultCollector_SharedSubscriptionsHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels)
	
	msg := &mockMessage{payload: []byte("24")}
	collector.sharedSubscriptionsHandler(nil, msg)
	
	assert.Equal(t, float64(24), collector.Metrics.sharedSubscriptions)
}