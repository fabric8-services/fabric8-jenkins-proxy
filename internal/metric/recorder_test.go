package metric

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

var (
	github  = "GitHub"
	jenkins = "Jenkins UI"
)

func TestReqsByTypeTotalMetric(t *testing.T) {

	recorder := PrometheusRecorder{}

	recorder.RecordReqByTypeTotal(github)
	recorder.RecordReqByTypeTotal(jenkins)

	recorder.RecordReqByTypeTotal(github)

	checkCounter(t, github, 2)
	checkCounter(t, jenkins, 1)
}

func checkCounter(t *testing.T, reportType string, expected int64) {
	reqMetric, _ := reqCnt.GetMetricWithLabelValues(reportType)
	m := &dto.Metric{}
	reqMetric.Write(m)
	actual := int64(m.Counter.GetValue())
	if actual != expected {
		t.Errorf("metric(\"%s\"), want: %d, got: %d", reportType, expected, actual)
	}
}
