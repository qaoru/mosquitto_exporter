package internal

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
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

func TestClientsCollector_ClientsHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)

	testCases := []struct {
		topic         string
		payload       string
		expectedKey   string
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
		msg := &mockMessage{
			payload: []byte(tc.payload),
			topic:   tc.topic,
		}
		collector.clientsHandler(nil, msg)
		assert.Equal(t, tc.expectedValue, collector.Metrics[tc.expectedKey])
	}
}

func TestClientsCollector_Subscribe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)
	client := newMockClient()

	collector.Subscribe(client)

	assert.ElementsMatch(t, []string{"$SYS/broker/clients/#"}, client.subscribedTopics())
	assert.NotNil(t, client.handlerFor("$SYS/broker/clients/#"))
}

func TestClientsCollector_Subscribe_Error(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)
	client := newMockClient().withSubscribeError(errors.New("boom"))

	topic := "$SYS/broker/clients/#"
	before := testutil.ToFloat64(SubscriptionErrors.WithLabelValues(topic, "boom"))
	collector.Subscribe(client)
	after := testutil.ToFloat64(SubscriptionErrors.WithLabelValues(topic, "boom"))
	assert.Equal(t, before+1, after)
}

func TestClientsCollector_ClientsHandler_ParseError(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewClientsCollector(labels)
	collector.Metrics["active"] = 42

	collector.clientsHandler(nil, &mockMessage{payload: []byte("notanumber"), topic: "$SYS/broker/clients/active"})

	assert.Equal(t, float64(42), collector.Metrics["active"])
}
