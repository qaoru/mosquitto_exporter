package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewLoadCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 9, len(collector.descriptions))
}

func TestLoadCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels)

	descriptions := make(chan *prometheus.Desc)
	go func() {
		collector.Describe(descriptions)
		close(descriptions)
	}()

	count := 0
	for range descriptions {
		count++
	}

	// 9 metrics * 3 load averages each = 27 descriptions
	assert.Equal(t, 27, count)
}

func TestLoadCollector_Collect(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels)

	// Set some test values
	collector.Metrics["connections_1min"] = 1.5
	collector.Metrics["connections_5min"] = 2.1
	collector.Metrics["connections_15min"] = 3.7
	collector.Metrics["bytes_received_1min"] = 1024.0
	collector.Metrics["bytes_received_5min"] = 2048.0
	collector.Metrics["bytes_received_15min"] = 4096.0

	metrics := make(chan prometheus.Metric)
	go func() {
		collector.Collect(metrics)
		close(metrics)
	}()

	count := 0
	for range metrics {
		count++
	}

	// 9 metrics * 3 load averages each = 27 metrics
	assert.Equal(t, 27, count)
}

func TestLoadCollector_LoadHandler(t *testing.T) {
	// Test different load metric topics
	testCases := []struct {
		topic   string
		payload string
		expectedKey string
		expectedValue float64
	}{
		{"$SYS/broker/load/connections/1min", "1.5", "connections_1min", 1.5},
		{"$SYS/broker/load/bytes/received/5min", "2048.0", "bytes_received_5min", 2048.0},
		{"$SYS/broker/load/messages/sent/15min", "128.0", "messages_sent_15min", 128.0},
	}

	for _, tc := range testCases {
		// Simulate the handler logic
		topicParts := strings.Split(tc.topic, "/")
		var key string
		switch len(topicParts) {
		case 5:
			key = topicParts[3] + "_" + topicParts[4]
		case 6:
			key = topicParts[3] + "_" + topicParts[4] + "_" + topicParts[5]
		}
		num, _ := strconv.ParseFloat(tc.payload, 64)
		
		assert.Equal(t, tc.expectedKey, key)
		assert.Equal(t, tc.expectedValue, num)
	}
}

func TestLoadCollector_LoadHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels)

	testCases := []struct {
		topic   string
		payload string
		expectedKey string
		expectedValue float64
	}{
		{"$SYS/broker/load/connections/1min", "1.5", "connections_1min", 1.5},
		{"$SYS/broker/load/bytes/received/5min", "2048.0", "bytes_received_5min", 2048.0},
		{"$SYS/broker/load/messages/sent/15min", "128.0", "messages_sent_15min", 128.0},
	}

	for _, tc := range testCases {
		msg := &mockMessage{
			payload: []byte(tc.payload),
			topic:   tc.topic,
		}
		collector.loadHandler(nil, msg)
		assert.Equal(t, tc.expectedValue, collector.Metrics[tc.expectedKey])
	}
}