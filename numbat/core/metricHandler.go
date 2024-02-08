package core

import "sync"

// Mh Global reference for metric handler
var Mh *MetricHandler

// init Function
func init() {
	Mh = NewMetricHandler()
}

// MetricHandler Structure
type MetricHandler struct {
	perAPICount     map[string]uint64
	perAPICountLock sync.Mutex // @todo perhaps combine those two?
}

// NewMetricHandler Function
func NewMetricHandler() *MetricHandler {
	mh := &MetricHandler{
		perAPICount: make(map[string]uint64),
	}

	return mh
}

// GetPerAPICount Function
func (mh *MetricHandler) GetPerAPICount() map[string]uint64 {
	return mh.perAPICount
}

// UpdatePerAPICount Function
func (mh *MetricHandler) UpdatePerAPICount(nm map[string]uint64) {
	mh.perAPICountLock.Lock()
	mh.perAPICount = nm
	mh.perAPICountLock.Unlock()
}
