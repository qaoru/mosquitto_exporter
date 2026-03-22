package internal

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestNewUpCollector(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewUpCollector(labels)

	assert.NotNil(t, collector)
	assert.Equal(t, float64(0), collector.up)
	assert.NotNil(t, collector.description)
}

func TestUpCollector_Describe(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewUpCollector(labels)

	ch := make(chan *prometheus.Desc, 1)
	collector.Describe(ch)
	close(ch)

	desc := <-ch
	assert.Contains(t, desc.String(), "mosquitto_up")
}

func TestUpCollector_Collect(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewUpCollector(labels)

	ch := make(chan prometheus.Metric)
	go func() {
		collector.Collect(ch)
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}
	assert.Equal(t, 1, count)
}

func TestUpCollector_SetUp(t *testing.T) {
	labels := prometheus.Labels{"broker": "test-broker"}
	collector := NewUpCollector(labels)

	collector.SetUp(true)
	assert.Equal(t, float64(1), collector.up)

	collector.SetUp(false)
	assert.Equal(t, float64(0), collector.up)
}