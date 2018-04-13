package metric

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
	reportRequestsTotal(requestType)
}
