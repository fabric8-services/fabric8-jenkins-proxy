package metric

import (
	"strings"
)

// Recorder interface that encapsulates all logic of metrics
type Recorder interface {
	Initialize()
	RecordReqByTypeTotal(requestType string)
}

// PrometheusRecorder struct used to record metrics to be consumed by Prometheus
type PrometheusRecorder struct {
}

// Initialize all metrics
func (pr PrometheusRecorder) Initialize() {
	registerMetrics()
}

// RecordReqByTypeTotal records a request type
func (pr PrometheusRecorder) RecordReqByTypeTotal(requestType string) {
	// adapt label to better name
	reportRequestsTotal(convertLabel(requestType))
}

func convertLabel(label string) string {
	newLabel := strings.ToLower(label)
	return strings.Replace(newLabel, " ", "", -1)
}
