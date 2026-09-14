package main

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/qaoru/mosquitto_exporter/internal"
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
	subErrors := internal.NewSubscriptionErrors(constLabels)
	defaultCollector := internal.NewDefaultCollector(constLabels, subErrors)
	assert.NotNil(t, defaultCollector)

	clientsCollector := internal.NewClientsCollector(constLabels, subErrors)
	assert.NotNil(t, clientsCollector)

	messagesCollector := internal.NewMessagesCollector(constLabels, subErrors)
	assert.NotNil(t, messagesCollector)

	loadCollector := internal.NewLoadCollector(constLabels, subErrors)
	assert.NotNil(t, loadCollector)
}
