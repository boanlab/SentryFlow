// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"sync"

	"github.com/5GSEC/SentryFlow/metrics/api"
	"github.com/5GSEC/SentryFlow/protobuf"
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
func StartMetricsAnalyzer(wg *sync.WaitGroup) {
	api.StartAPIAnalyzer(wg)
}

// StopMetricsAnalyzer Function
func StopMetricsAnalyzer() {
	api.StopAPIAnalyzer()
}

// InsertAccessLog Function
func InsertAccessLog(al *protobuf.APILog) {
	// @todo: make this fixed, for now will just send path from AccessLog
	api.InsertAnalyzeJob(al.Path)
}
