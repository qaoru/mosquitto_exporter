package internal

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewLoadCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))

	assert.NotNil(t, collector)
	assert.NotNil(t, collector.Metrics)
	assert.NotNil(t, collector.descriptions)
	assert.Equal(t, 9, len(collector.descriptions))
}

func TestLoadCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))

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
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))

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

func TestLoadCollector_LoadHandler_Integration(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))

	testCases := []struct {
		topic         string
		payload       string
		expectedKey   string
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

func TestLoadCollector_Subscribe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))
	client := newMockClient()

	collector.Subscribe(client)

	assert.ElementsMatch(t, []string{"$SYS/broker/load/#"}, client.subscribedTopics())
	assert.NotNil(t, client.handlerFor("$SYS/broker/load/#"))
}

func TestLoadCollector_Subscribe_Error(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	subErr := newTestSubscriptionErrors(t)
	collector := NewLoadCollector(labels, subErr)
	client := newMockClient().withSubscribeError(errors.New("boom"))

	topic := "$SYS/broker/load/#"
	before := testutil.ToFloat64(subErr.WithLabelValues(topic, "boom"))
	collector.Subscribe(client)
	after := testutil.ToFloat64(subErr.WithLabelValues(topic, "boom"))
	assert.Equal(t, before+1, after)
}

func TestLoadCollector_LoadHandler_ParseError(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))
	collector.Metrics["connections_1min"] = 9.9

	collector.loadHandler(nil, &mockMessage{payload: []byte("notanumber"), topic: "$SYS/broker/load/connections/1min"})

	assert.Equal(t, 9.9, collector.Metrics["connections_1min"])
}

func TestLoadCollector_LoadHandler_RejectsNonFinite(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewLoadCollector(labels, newTestSubscriptionErrors(t))
	collector.Metrics["connections_1min"] = 1.5

	for _, payload := range []string{"NaN", "nan", "+Inf", "-Inf", "Infinity"} {
		collector.loadHandler(nil, &mockMessage{payload: []byte(payload), topic: "$SYS/broker/load/connections/1min"})
		assert.Equal(t, 1.5, collector.Metrics["connections_1min"], "non-finite payload %q overwrote stored value", payload)
	}
}
