package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

var logger = log.WithFields(log.Fields{"component": "metrics"})

var (
	namespace = ""
	subsystem = "service"
)

var (
	reqLabels = []string{"requestType"}

	reqCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_type_total",
		Help:      "Counter of requests received into the system.",
	}, reqLabels)
)

func registerMetrics() {
	reqCnt = register(reqCnt, "requests_type_total").(*prometheus.CounterVec)
}

func register(c prometheus.Collector, name string) prometheus.Collector {
	err := prometheus.Register(c)
	if err != nil {
		if are, ok := err.(prometheus.AlreadyRegisteredError); ok {
			return are.ExistingCollector
		}
		logger.
			WithField("err", err).
			WithField("metric_name", prometheus.BuildFQName(namespace, subsystem, name)).
			Panic("Failed to register the prometheus metric")
	}
	logger.
		WithField("metric_name", prometheus.BuildFQName(namespace, subsystem, name)).
		Debug("metric registered successfully")
	return c
}

func reportRequestsTotal(requestType string) {
	if requestType != "" {
		reqCnt.WithLabelValues(requestType).Inc()
	}
}
