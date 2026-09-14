package internal

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

// mockMessage implements mqtt.Message for testing
type mockMessage struct {
	payload []byte
	topic   string
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

func TestNewDefaultCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 4, len(collector.descriptions))
}

func TestDefaultCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

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
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

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

func TestDefaultCollector_UptimeHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

	// Create mock message with uptime payload
	msg := &mockMessage{payload: []byte("12345 seconds")}

	// Call the handler directly
	collector.uptimeHandler(nil, msg)

	// Verify the metric was updated
	assert.Equal(t, float64(12345), collector.Metrics.uptime)
}

func TestDefaultCollector_VersionHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

	msg := &mockMessage{payload: []byte("mosquitto version 2.0.15")}
	collector.versionHandler(nil, msg)

	assert.Equal(t, "2.0.15", collector.Metrics.version)
}

func TestDefaultCollector_SubscriptionsHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

	msg := &mockMessage{payload: []byte("42")}
	collector.subscriptionsHandler(nil, msg)

	assert.Equal(t, float64(42), collector.Metrics.subscriptions)
}

func TestDefaultCollector_SharedSubscriptionsHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))

	msg := &mockMessage{payload: []byte("24")}
	collector.sharedSubscriptionsHandler(nil, msg)

	assert.Equal(t, float64(24), collector.Metrics.sharedSubscriptions)
}

func TestDefaultCollector_Subscribe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))
	client := newMockClient()

	collector.Subscribe(client)

	expected := []string{
		"$SYS/broker/uptime",
		"$SYS/broker/version",
		"$SYS/broker/subscriptions/count",
		"$SYS/broker/shared_subscriptions/count",
	}
	assert.ElementsMatch(t, expected, client.subscribedTopics())
	for _, topic := range expected {
		assert.NotNil(t, client.handlerFor(topic), "missing handler for %s", topic)
	}
}

func TestDefaultCollector_Subscribe_Error(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	subErr := newTestSubscriptionErrors(t)
	collector := NewDefaultCollector(labels, subErr)
	client := newMockClient().withSubscribeError(errors.New("boom"))

	topics := []string{
		"$SYS/broker/uptime",
		"$SYS/broker/version",
		"$SYS/broker/subscriptions/count",
		"$SYS/broker/shared_subscriptions/count",
	}
	before := map[string]float64{}
	for _, topic := range topics {
		before[topic] = testutil.ToFloat64(subErr.WithLabelValues(topic, "boom"))
	}

	collector.Subscribe(client)

	for _, topic := range topics {
		after := testutil.ToFloat64(subErr.WithLabelValues(topic, "boom"))
		assert.Equal(t, before[topic]+1, after, "subscription error not counted for %s", topic)
	}
}

func TestDefaultCollector_Handlers_ParseErrors(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewDefaultCollector(labels, newTestSubscriptionErrors(t))
	// Seed known values; a malformed payload must not overwrite them.
	collector.Metrics.uptime = 100
	collector.Metrics.version = "2.0.0"
	collector.Metrics.subscriptions = 5
	collector.Metrics.sharedSubscriptions = 3

	collector.uptimeHandler(nil, &mockMessage{payload: []byte("notanumber seconds")})
	collector.uptimeHandler(nil, &mockMessage{payload: []byte("")})
	collector.versionHandler(nil, &mockMessage{payload: []byte("mosquitto")})
	collector.subscriptionsHandler(nil, &mockMessage{payload: []byte("notanumber")})
	collector.sharedSubscriptionsHandler(nil, &mockMessage{payload: []byte("notanumber")})

	assert.Equal(t, float64(100), collector.Metrics.uptime)
	assert.Equal(t, "2.0.0", collector.Metrics.version)
	assert.Equal(t, float64(5), collector.Metrics.subscriptions)
	assert.Equal(t, float64(3), collector.Metrics.sharedSubscriptions)
}
