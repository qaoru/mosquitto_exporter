package internal

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

type MessagesCollector struct {
	mu           sync.RWMutex
	Metrics      map[string]float64
	descriptions map[string]metric
}

func NewMessagesCollector(labels prometheus.Labels) *MessagesCollector {
	return &MessagesCollector{
		mu:      sync.RWMutex{},
		Metrics: make(map[string]float64, 4),
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
				desc:      prometheus.NewDesc("mosquitto_stored_messages_bytes", "Stored messages size in bytse", nil, labels),
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

	for k, v := range collector.descriptions {
		collector.mu.RLock()
		ch <- prometheus.MustNewConstMetric(v.desc, v.valueType, collector.Metrics[k])
		collector.mu.RUnlock()
	}
}

func (collector *MessagesCollector) Subscribe(client mqtt.Client) {
	if token := client.Subscribe("$SYS/broker/messages/#", 0, collector.messagesHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	if token := client.Subscribe("$SYS/broker/store/messages/#", 0, collector.storedMessagesHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (collector *MessagesCollector) messagesHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	last := topic[len(topic)-1]
	num, _ := strconv.Atoi(string(message.Payload()))
	collector.mu.Lock()
	collector.Metrics[last] = float64(num)
	collector.mu.Unlock()
}

func (collector *MessagesCollector) storedMessagesHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	last := topic[len(topic)-1]
	num, _ := strconv.Atoi(string(message.Payload()))
	key := "stored_" + last
	collector.mu.Lock()
	collector.Metrics[key] = float64(num)
	collector.mu.Unlock()
}
