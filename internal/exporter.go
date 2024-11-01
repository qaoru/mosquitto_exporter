package internal

import (
	"github.com/prometheus/client_golang/prometheus"
)

type metric struct {
	desc      *prometheus.Desc
	valueType prometheus.ValueType
}
