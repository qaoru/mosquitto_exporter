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

type defaultMetrics struct {
	uptime  float64
	version string
}

type DefaultCollector struct {
	descriptions map[string]metric
	mu           sync.RWMutex
	Metrics      *defaultMetrics
}

func NewDefaultCollector(labels prometheus.Labels) *DefaultCollector {
	return &DefaultCollector{
		mu:      sync.RWMutex{},
		Metrics: &defaultMetrics{},
		descriptions: map[string]metric{
			"uptime": {
				desc:      prometheus.NewDesc("mosquitto_uptime_seconds", "Seconds since the broker was started", nil, labels),
				valueType: prometheus.CounterValue,
			},
			"version": {
				desc:      prometheus.NewDesc("mosquitto_version_info", "Mosquitto version", []string{"version"}, labels),
				valueType: prometheus.GaugeValue,
			},
		},
	}
}

func (collector *DefaultCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range collector.descriptions {
		ch <- desc.desc
	}
}

func (collector *DefaultCollector) Collect(ch chan<- prometheus.Metric) {

	collector.mu.RLock()
	ch <- prometheus.MustNewConstMetric(collector.descriptions["uptime"].desc, collector.descriptions["uptime"].valueType, collector.Metrics.uptime)
	ch <- prometheus.MustNewConstMetric(collector.descriptions["version"].desc, collector.descriptions["version"].valueType, 1, collector.Metrics.version)
	collector.mu.RUnlock()
}

func (collector *DefaultCollector) Subscribe(client mqtt.Client) {
	if token := client.Subscribe("$SYS/broker/uptime", 0, collector.uptimeHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	if token := client.Subscribe("$SYS/broker/version", 0, collector.versionHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (collector *DefaultCollector) uptimeHandler(client mqtt.Client, message mqtt.Message) {
	// Payload is 'XXX seconds'
	uptime, _ := strconv.Atoi(strings.Split(string(message.Payload()), " ")[0])
	collector.mu.Lock()
	collector.Metrics.uptime = float64(uptime)
	collector.mu.Unlock()
}

func (collector *DefaultCollector) versionHandler(client mqtt.Client, message mqtt.Message) {
	// Payload is 'mosquitto version X.X.X'
	version := strings.Split(string(message.Payload()), " ")[2]
	collector.mu.Lock()
	collector.Metrics.version = version
	collector.mu.Unlock()
}
