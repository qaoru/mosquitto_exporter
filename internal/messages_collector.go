package internal

import (
	"log"
	"strconv"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

type MessagesCollector struct {
	mu                 sync.RWMutex
	Metrics            map[string]float64
	descriptions       map[string]metric
	subscriptionErrors *prometheus.CounterVec
}

func NewMessagesCollector(labels prometheus.Labels, subErrors *prometheus.CounterVec) *MessagesCollector {
	return &MessagesCollector{
		mu:                 sync.RWMutex{},
		Metrics:            make(map[string]float64, 4),
		subscriptionErrors: subErrors,
		descriptions: map[string]metric{
			"received": {
				desc:      prometheus.NewDesc("mosquitto_received_messages_count", "Number of received messages", nil, labels),
				valueType: prometheus.CounterValue,
			},
			"sent": {
				desc:      prometheus.NewDesc("mosquitto_sent_messages_count", "Number of sent messages", nil, labels),
				valueType: prometheus.CounterValue,
			},
			"stored_count": {
				desc:      prometheus.NewDesc("mosquitto_stored_messages_count", "Number of stored messages", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"stored_bytes": {
				desc:      prometheus.NewDesc("mosquitto_stored_messages_bytes", "Stored messages size in bytes", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"inflight": {
				desc:      prometheus.NewDesc("mosquitto_inflight_messages_gauge", "Number of inflight messages", nil, labels),
				valueType: prometheus.GaugeValue,
			},
		},
	}
}

func (collector *MessagesCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range collector.descriptions {
		ch <- desc.desc
	}
}

func (collector *MessagesCollector) Collect(ch chan<- prometheus.Metric) {
	collector.mu.RLock()
	defer collector.mu.RUnlock()
	for k, v := range collector.descriptions {
		ch <- prometheus.MustNewConstMetric(v.desc, v.valueType, collector.Metrics[k])
	}
}

func (collector *MessagesCollector) Subscribe(client mqtt.Client) {
	if token := client.Subscribe("$SYS/broker/messages/#", 0, collector.messagesHandler); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to $SYS/broker/messages/#: %v", token.Error())
		collector.subscriptionErrors.WithLabelValues("$SYS/broker/messages/#", token.Error().Error()).Inc()
	}
	if token := client.Subscribe("$SYS/broker/store/messages/#", 0, collector.storedMessagesHandler); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to $SYS/broker/store/messages/#: %v", token.Error())
		collector.subscriptionErrors.WithLabelValues("$SYS/broker/store/messages/#", token.Error().Error()).Inc()
	}
}

func (collector *MessagesCollector) messagesHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	last := topic[len(topic)-1]
	num, err := strconv.Atoi(string(message.Payload()))
	if err != nil {
		log.Printf("Failed to parse messages metric %q from %q: %v", last, message.Payload(), err)
		return
	}
	collector.mu.Lock()
	collector.Metrics[last] = float64(num)
	collector.mu.Unlock()
}

func (collector *MessagesCollector) storedMessagesHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	last := topic[len(topic)-1]
	num, err := strconv.Atoi(string(message.Payload()))
	if err != nil {
		log.Printf("Failed to parse stored messages metric %q from %q: %v", last, message.Payload(), err)
		return
	}
	key := "stored_" + last
	collector.mu.Lock()
	collector.Metrics[key] = float64(num)
	collector.mu.Unlock()
}
