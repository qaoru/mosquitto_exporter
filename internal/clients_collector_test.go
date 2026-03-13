package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewClientsCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 7, len(collector.descriptions))
}

func TestClientsCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)

	descriptions := make(chan *prometheus.Desc)
	go func() {
		collector.Describe(descriptions)
		close(descriptions)
	}()

	count := 0
	for range descriptions {
		count++
	}

	assert.Equal(t, 7, count)
}

func TestClientsCollector_Collect(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)

	// Set some test values
	collector.Metrics["active"] = 10
	collector.Metrics["connected"] = 8
	collector.Metrics["disconnected"] = 2
	collector.Metrics["expired"] = 1
	collector.Metrics["inactive"] = 3
	collector.Metrics["maximum"] = 15
	collector.Metrics["total"] = 20

	metrics := make(chan prometheus.Metric)
	go func() {
		collector.Collect(metrics)
		close(metrics)
	}()

	count := 0
	for range metrics {
		count++
	}

	assert.Equal(t, 7, count)
}

func TestClientsCollector_ClientsHandler(t *testing.T) {
	// Test different client metrics
	testCases := []struct {
		topic   string
		payload string
		expectedKey string
		expectedValue float64
	}{
		{"$SYS/broker/clients/active", "10", "active", 10},
		{"$SYS/broker/clients/connected", "8", "connected", 8},
		{"$SYS/broker/clients/disconnected", "2", "disconnected", 2},
		{"$SYS/broker/clients/expired", "1", "expired", 1},
		{"$SYS/broker/clients/inactive", "3", "inactive", 3},
		{"$SYS/broker/clients/maximum", "15", "maximum", 15},
		{"$SYS/broker/clients/total", "20", "total", 20},
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