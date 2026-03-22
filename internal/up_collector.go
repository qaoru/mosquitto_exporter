package internal

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type UpCollector struct {
	mu          sync.RWMutex
	up          float64
	description *prometheus.Desc
}

func NewUpCollector(labels prometheus.Labels) *UpCollector {
	return &UpCollector{
		up:          0,
		description: prometheus.NewDesc("mosquitto_up", "Whether the exporter is connected to the broker (1 = up, 0 = down)", nil, labels),
	}
}

func (c *UpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.description
}

func (c *UpCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	ch <- prometheus.MustNewConstMetric(c.description, prometheus.GaugeValue, c.up)
	c.mu.RUnlock()
}

func (c *UpCollector) SetUp(up bool) {
	c.mu.Lock()
	if up {
		c.up = 1
	} else {
		c.up = 0
	}
	c.mu.Unlock()
}