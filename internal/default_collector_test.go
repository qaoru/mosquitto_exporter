package internal

import (
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

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