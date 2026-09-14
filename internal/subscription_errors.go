package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

// NewSubscriptionErrors creates the subscription-errors counter, baking in the
// supplied const labels (e.g. `broker`) so errors can be attributed per broker.
//
// It is intended to be constructed once in main() and explicitly registered
// there, consistent with the other collectors, rather than via package-level
// init() — the broker URL (and therefore the const label value) is only known
// after flag parsing in main().
func NewSubscriptionErrors(constLabels prometheus.Labels) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "mosquitto_subscription_errors_total",
			Help:        "Total number of subscription errors",
			ConstLabels: constLabels,
		},
		[]string{"topic", "error"},
	)
}
