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

func genLoadDescription(t prometheus.ValueType, fqName string, help string, variableLabels []string, constLabels prometheus.Labels) [3]metric {
	return [3]metric{
		{
			desc:      prometheus.NewDesc(fmt.Sprintf("%s_load1", fqName), help, nil, nil),
			valueType: t,
		},
		{
			desc:      prometheus.NewDesc(fmt.Sprintf("%s_load5", fqName), help, nil, nil),
			valueType: t,
		},
		{
			desc:      prometheus.NewDesc(fmt.Sprintf("%s_load15", fqName), help, nil, nil),
			valueType: t,
		},
	}
}

type LoadCollector struct {
	mu           sync.RWMutex
	Metrics      map[string]float64
	descriptions map[string][3]metric
}

func NewLoadCollector(labels prometheus.Labels) *LoadCollector {
	return &LoadCollector{
		mu:      sync.RWMutex{},
		Metrics: make(map[string]float64, 32),
		descriptions: map[string][3]metric{
			"connections":       genLoadDescription(prometheus.GaugeValue, "mosquitto_connections", "The moving average of the number of connections opened to the broker", nil, labels),
			"sockets":           genLoadDescription(prometheus.GaugeValue, "mosquitto_sockets", "The moving average of the number of socket connections opened to the broker", nil, labels),
			"bytes_received":    genLoadDescription(prometheus.GaugeValue, "mosquitto_bytes_received", "The moving average of the number of bytes received by the broker", nil, labels),
			"bytes_sent":        genLoadDescription(prometheus.GaugeValue, "mosquitto_bytes_sent", "The moving average of the number of bytes sent by the broker", nil, labels),
			"messages_received": genLoadDescription(prometheus.GaugeValue, "mosquitto_messages_received", "The moving average of the number of messages received by the broker", nil, labels),
			"messages_sent":     genLoadDescription(prometheus.GaugeValue, "mosquitto_messages_sent", "The moving average of the number of messages sent by the broker", nil, labels),
			"publish_received":  genLoadDescription(prometheus.GaugeValue, "mosquitto_publish_received", "The moving average of the number of publish messages received by the broker", nil, labels),
			"publish_sent":      genLoadDescription(prometheus.GaugeValue, "mosquitto_publish_sent", "The moving average of the number of publish messages sent by the broker", nil, labels),
			"publish_dropped":   genLoadDescription(prometheus.GaugeValue, "mosquitto_publish_dropped", "The moving average of the number of publish messages dropped by the broker", nil, labels),
		},
	}
}

func (collector *LoadCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range collector.descriptions {
		ch <- desc[0].desc
		ch <- desc[1].desc
		ch <- desc[2].desc
	}
}

func (collector *LoadCollector) Collect(ch chan<- prometheus.Metric) {

	for k, v := range collector.descriptions {
		k1 := fmt.Sprintf("%s_1min", k)
		k2 := fmt.Sprintf("%s_5min", k)
		k3 := fmt.Sprintf("%s_15min", k)
		collector.mu.RLock()
		ch <- prometheus.MustNewConstMetric(v[0].desc, v[0].valueType, collector.Metrics[k1])
		ch <- prometheus.MustNewConstMetric(v[1].desc, v[1].valueType, collector.Metrics[k2])
		ch <- prometheus.MustNewConstMetric(v[2].desc, v[2].valueType, collector.Metrics[k3])
		collector.mu.RUnlock()
	}
}

func (collector *LoadCollector) Subscribe(client mqtt.Client) {
	if token := client.Subscribe("$SYS/broker/load/#", 0, collector.loadHandler); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
}

func (collector *LoadCollector) loadHandler(client mqtt.Client, message mqtt.Message) {
	topic := strings.Split(message.Topic(), "/")
	var key string
	switch len(topic) {
	case 5:
		key = fmt.Sprintf("%s_%s", topic[3], topic[4])
	case 6:
		key = fmt.Sprintf("%s_%s_%s", topic[3], topic[4], topic[5])
	}
	num, _ := strconv.ParseFloat(string(message.Payload()), 64)
	collector.mu.Lock()
	collector.Metrics[key] = float64(num)
	collector.mu.Unlock()
}
