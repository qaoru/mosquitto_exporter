package internal

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewMessagesCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 5, len(collector.descriptions))
}

func TestMessagesCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))

	descriptions := make(chan *prometheus.Desc)
	go func() {
		collector.Describe(descriptions)
		close(descriptions)
	}()

	count := 0
	for range descriptions {
		count++
	}

	assert.Equal(t, 5, count)
}

func TestMessagesCollector_Collect(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))

	// Set some test values
	collector.Metrics["received"] = 100
	collector.Metrics["sent"] = 95
	collector.Metrics["stored_count"] = 5
	collector.Metrics["stored_bytes"] = 1024
	collector.Metrics["inflight"] = 3

	metrics := make(chan prometheus.Metric)
	go func() {
		collector.Collect(metrics)
		close(metrics)
	}()

	count := 0
	for range metrics {
		count++
	}

	assert.Equal(t, 5, count)
}

func TestMessagesCollector_MessagesHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))

	testCases := []struct {
		topic         string
		payload       string
		expectedKey   string
		expectedValue float64
	}{
		{"$SYS/broker/messages/received", "100", "received", 100},
		{"$SYS/broker/messages/sent", "95", "sent", 95},
		{"$SYS/broker/messages/inflight", "3", "inflight", 3},
	}

	for _, tc := range testCases {
		msg := &mockMessage{
			payload: []byte(tc.payload),
			topic:   tc.topic,
		}
		collector.messagesHandler(nil, msg)
		assert.Equal(t, tc.expectedValue, collector.Metrics[tc.expectedKey])
	}
}

func TestMessagesCollector_StoredMessagesHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))

	testCases := []struct {
		topic         string
		payload       string
		expectedKey   string
		expectedValue float64
	}{
		{"$SYS/broker/store/messages/count", "5", "stored_count", 5},
		{"$SYS/broker/store/messages/bytes", "1024", "stored_bytes", 1024},
	}

	for _, tc := range testCases {
		msg := &mockMessage{
			payload: []byte(tc.payload),
			topic:   tc.topic,
		}
		collector.storedMessagesHandler(nil, msg)
		assert.Equal(t, tc.expectedValue, collector.Metrics[tc.expectedKey])
	}
}

func TestMessagesCollector_Subscribe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))
	client := newMockClient()

	collector.Subscribe(client)

	expected := []string{
		"$SYS/broker/messages/#",
		"$SYS/broker/store/messages/#",
	}
	assert.ElementsMatch(t, expected, client.subscribedTopics())
	for _, topic := range expected {
		assert.NotNil(t, client.handlerFor(topic), "missing handler for %s", topic)
	}
}

func TestMessagesCollector_Subscribe_Error(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	subErr := newTestSubscriptionErrors(t)
	collector := NewMessagesCollector(labels, subErr)
	client := newMockClient().withSubscribeError(errors.New("boom"))

	topics := []string{
		"$SYS/broker/messages/#",
		"$SYS/broker/store/messages/#",
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

func TestMessagesCollector_Handlers_ParseError(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels, newTestSubscriptionErrors(t))
	collector.Metrics["received"] = 50
	collector.Metrics["stored_count"] = 7

	collector.messagesHandler(nil, &mockMessage{payload: []byte("notanumber"), topic: "$SYS/broker/messages/received"})
	collector.storedMessagesHandler(nil, &mockMessage{payload: []byte("notanumber"), topic: "$SYS/broker/store/messages/count"})

	assert.Equal(t, float64(50), collector.Metrics["received"])
	assert.Equal(t, float64(7), collector.Metrics["stored_count"])
}
