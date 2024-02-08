package metrics

import (
	"numbat/metrics/api"
	"numbat/protobuf"
)

// Mh Global reference for metric handler
var Mh *MetricHandler

// init Function
func init() {
	Mh = NewMetricHandler()
}

// MetricHandler Structure
type MetricHandler struct {
}

// NewMetricHandler Function
func NewMetricHandler() *MetricHandler {
	mh := &MetricHandler{}

	return mh
}

// StartMetricsAnalyzer Function
func StartMetricsAnalyzer() {
	api.StartAPIAnalyzer()
}

// StopMetricsAnalyzer Function
func StopMetricsAnalyzer() {
	api.StopAPIAnalyzer()
}

// InsertAccessLog Function
func InsertAccessLog(al *protobuf.Log) {
	// @todo: make this fixed, for now will just send path from AccessLog
	api.InsertAnalyzeJob(al.Path)
}
