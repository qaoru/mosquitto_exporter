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

var ClientMetrics = make(map[string]float64, 16)

type ClientsCollector struct {
	mu           sync.RWMutex
	Metrics      map[string]float64
	descriptions map[string]metric
}

func NewClientsCollector(labels prometheus.Labels) *ClientsCollector {
	return &ClientsCollector{
		mu:      sync.RWMutex{},
		Metrics: make(map[string]float64, 8),
		descriptions: map[string]metric{
			"active": {
				desc:      prometheus.NewDesc("mosquitto_active_clients_count", "Number of active clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"connected": {
				desc:      prometheus.NewDesc("mosquitto_connected_clients_count", "Number of connected clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"disconnected": {
				desc:      prometheus.NewDesc("mosquitto_disconnected_clients_count", "Number of disconnected clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"expired": {
				desc:      prometheus.NewDesc("mosquitto_expired_clients_count", "Number of expired clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"inactive": {
				desc:      prometheus.NewDesc("mosquitto_inactive_clients_count", "Number of inactive clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"maximum": {
				desc:      prometheus.NewDesc("mosquitto_maximum_clients_count", "Maximum number of simultaneously connected clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
			"total": {
				desc:      prometheus.NewDesc("mosquitto_total_clients_count", "Total number of clients", nil, labels),
				valueType: prometheus.GaugeValue,
			},
		},
	}
}

func (collector *ClientsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range collector.descriptions {
		ch <- desc.desc
	}
}

func (collector *ClientsCollector) Collect(ch chan<- prometheus.Metric) {

	for k, v := range collector.descriptions {
		collector.mu.RLock()
		ch <- prometheus.MustNewConstMetric(v.desc, v.valueType, collector.Metrics[k])
		collector.mu.RUnlock()
	}
}

func (collector *ClientsCollector) Subscribe(client mqtt.Client) {
	if token := client.Subscribe("$SYS/broker/clients/#", 0, collector.clientsHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (collector *ClientsCollector) clientsHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	last := topic[len(topic)-1]
	num, _ := strconv.Atoi(string(message.Payload()))
	collector.mu.Lock()
	collector.Metrics[last] = float64(num)
	collector.mu.Unlock()
}
