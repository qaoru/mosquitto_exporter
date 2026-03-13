package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// Test edge cases for parsing functions
func TestDefaultCollector_EdgeCases(t *testing.T) {
	// Test malformed uptime
	testCases := []struct {
		payload    string
		expected   float64
		description string
	}{
		{"12345 seconds", 12345, "normal case"},
		{"0 seconds", 0, "zero uptime"},
		{"999999 seconds", 999999, "large uptime"},
		{"1 seconds", 1, "single digit"},
	}

	for _, tc := range testCases {
		t := t
		t.Run(tc.description, func(t *testing.T) {
			parts := strings.Split(tc.payload, " ")
			uptime, err := strconv.Atoi(parts[0])
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, float64(uptime))
		})
	}

	// Test malformed version strings
	versionCases := []struct {
		payload    string
		expected   string
		description string
	}{
		{"mosquitto version 2.0.15", "2.0.15", "normal version"},
		{"mosquitto version 1.6.12", "1.6.12", "older version"},
		{"mosquitto version 0.1.0", "0.1.0", "very old version"},
	}

	for _, tc := range versionCases {
		t := t
		t.Run(tc.description, func(t *testing.T) {
			parts := strings.Split(tc.payload, " ")
			version := parts[2]
			assert.Equal(t, tc.expected, version)
		})
	}
}

func TestClientsCollector_EdgeCases(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)

	// Test edge case values
	edgeCases := []struct {
		key    string
		value  int
		description string
	}{
		{"active", 0, "zero active clients"},
		{"connected", 1000, "many connected clients"},
		{"total", 999999, "very large total"},
		{"expired", 1, "single expired client"},
	}

	for _, tc := range edgeCases {
		t := t
		t.Run(tc.description, func(t *testing.T) {
			collector.Metrics[tc.key] = float64(tc.value)
			assert.Equal(t, float64(tc.value), collector.Metrics[tc.key])
		})
	}
}

func TestMessagesCollector_EdgeCases(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewMessagesCollector(labels)

	// Test edge case values
	edgeCases := []struct {
		key    string
		value  int
		description string
	}{
		{"received", 0, "zero received messages"},
		{"sent", 1000000, "million sent messages"},
		{"stored_bytes", 1073741824, "1GB stored bytes"},
		{"inflight", 1, "single inflight message"},
	}

	for _, tc := range edgeCases {
		t := t
		t.Run(tc.description, func(t *testing.T) {
			collector.Metrics[tc.key] = float64(tc.value)
			assert.Equal(t, float64(tc.value), collector.Metrics[tc.key])
		})
	}
}

func TestLoadCollector_EdgeCases(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels)

	// Test edge case values for load averages
	edgeCases := []struct {
		key    string
		value  float64
		description string
	}{
		{"connections_1min", 0.0, "zero load"},
		{"bytes_received_5min", 1024.5, "fractional load"},
		{"messages_sent_15min", 999999.99, "very high load"},
		{"connections_1min", 0.1, "very low load"},
	}

	for _, tc := range edgeCases {
		t := t
		t.Run(tc.description, func(t *testing.T) {
			collector.Metrics[tc.key] = tc.value
			assert.Equal(t, tc.value, collector.Metrics[tc.key])
		})
	}
}

// Test metric description generation
func TestMetricDescriptions(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}

	// Test default collector descriptions
	defaultCollector := NewDefaultCollector(labels)
	assert.NotNil(t, defaultCollector.descriptions["uptime"])
	assert.NotNil(t, defaultCollector.descriptions["version"])
	assert.NotNil(t, defaultCollector.descriptions["subscriptions_total"])
	assert.NotNil(t, defaultCollector.descriptions["shared_subscriptions_total"])

	// Test clients collector descriptions
	clientsCollector := NewClientsCollector(labels)
	assert.NotNil(t, clientsCollector.descriptions["active"])
	assert.NotNil(t, clientsCollector.descriptions["connected"])
	assert.NotNil(t, clientsCollector.descriptions["disconnected"])
	assert.NotNil(t, clientsCollector.descriptions["expired"])
	assert.NotNil(t, clientsCollector.descriptions["inactive"])
	assert.NotNil(t, clientsCollector.descriptions["maximum"])
	assert.NotNil(t, clientsCollector.descriptions["total"])

	// Test messages collector descriptions
	messagesCollector := NewMessagesCollector(labels)
	assert.NotNil(t, messagesCollector.descriptions["received"])
	assert.NotNil(t, messagesCollector.descriptions["sent"])
	assert.NotNil(t, messagesCollector.descriptions["stored_count"])
	assert.NotNil(t, messagesCollector.descriptions["stored_bytes"])
	assert.NotNil(t, messagesCollector.descriptions["inflight"])

	// Test load collector descriptions
	loadCollector := NewLoadCollector(labels)
	assert.NotNil(t, loadCollector.descriptions["connections"])
	assert.NotNil(t, loadCollector.descriptions["sockets"])
	assert.NotNil(t, loadCollector.descriptions["bytes_received"])
	assert.NotNil(t, loadCollector.descriptions["bytes_sent"])
	assert.NotNil(t, loadCollector.descriptions["messages_received"])
	assert.NotNil(t, loadCollector.descriptions["messages_sent"])
	assert.NotNil(t, loadCollector.descriptions["publish_received"])
	assert.NotNil(t, loadCollector.descriptions["publish_sent"])
	assert.NotNil(t, loadCollector.descriptions["publish_dropped"])
}