package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// SubscriptionErrors counts subscription failures per topic and error.
	SubscriptionErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mosquitto_subscription_errors_total",
			Help: "Total number of subscription errors",
		},
		[]string{"topic", "error"},
	)
)

func init() {
	prometheus.MustRegister(SubscriptionErrors)
}