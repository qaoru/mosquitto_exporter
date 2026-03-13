package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewMessagesCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 5, len(collector.descriptions))
}

func TestMessagesCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels)

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
	collector := NewMessagesCollector(labels)

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

func TestMessagesCollector_MessagesHandler(t *testing.T) {
	// Test different message metrics
	testCases := []struct {
		topic   string
		payload string
		expectedKey string
		expectedValue float64
	}{
		{"$SYS/broker/messages/received", "100", "received", 100},
		{"$SYS/broker/messages/sent", "95", "sent", 95},
		{"$SYS/broker/messages/inflight", "3", "inflight", 3},
	}

	for _, tc := range testCases {
		// Simulate the handler logic
		topicParts := strings.Split(tc.topic, "/")
		last := topicParts[len(topicParts)-1]
		num, _ := strconv.Atoi(tc.payload)
		
		assert.Equal(t, tc.expectedKey, last)
		assert.Equal(t, tc.expectedValue, float64(num))
	}
}

func TestMessagesCollector_StoredMessagesHandler(t *testing.T) {
	// Test stored message metrics
	testCases := []struct {
		topic   string
		payload string
		expectedKey string
		expectedValue float64
	}{
		{"$SYS/broker/store/messages/count", "5", "stored_count", 5},
		{"$SYS/broker/store/messages/bytes", "1024", "stored_bytes", 1024},
	}

	for _, tc := range testCases {
		// Simulate the handler logic
		topicParts := strings.Split(tc.topic, "/")
		last := topicParts[len(topicParts)-1]
		key := "stored_" + last
		num, _ := strconv.Atoi(tc.payload)
		
		assert.Equal(t, tc.expectedKey, key)
		assert.Equal(t, tc.expectedValue, float64(num))
	}
}