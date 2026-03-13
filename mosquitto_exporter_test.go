package main

import (
	"testing"

	"github.com/qaoru/mosquitto_exporter/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMainFunctionality(t *testing.T) {
	// This is a basic test to ensure the main components can be initialized
	// without errors. More comprehensive integration tests would require
	// a running MQTT broker.

	// Test that we can create labels
	constLabels := make(prometheus.Labels, 4)
	constLabels["broker"] = "test-broker"
	constLabels["environment"] = "test"

	assert.Equal(t, "test-broker", constLabels["broker"])
	assert.Equal(t, "test", constLabels["environment"])

	// Test that collectors can be created
	defaultCollector := internal.NewDefaultCollector(constLabels)
	assert.NotNil(t, defaultCollector)

	clientsCollector := internal.NewClientsCollector(constLabels)
	assert.NotNil(t, clientsCollector)

	messagesCollector := internal.NewMessagesCollector(constLabels)
	assert.NotNil(t, messagesCollector)

	loadCollector := internal.NewLoadCollector(constLabels)
	assert.NotNil(t, loadCollector)
}