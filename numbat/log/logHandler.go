package log

import (
	"log"
	"numbat/exporter"
	"numbat/metrics"
	"numbat/protobuf"
)

// Lh global reference for LogHandler
var Lh *LogHandler

// init Function
func init() {
	Lh = NewLogHandler()
}

// LogHandler Structure
type LogHandler struct {
	stopChan chan struct{}
	logChan  chan interface{}
}

// NewLogHandler Structure
func NewLogHandler() *LogHandler {
	lh := &LogHandler{
		stopChan: make(chan struct{}),
		logChan:  make(chan interface{}),
	}

	return lh
}

// StartLogProcessor Function
func StartLogProcessor() {
	go Lh.logProcessingRoutine()
}

// StopLogProcessor Function
func StopLogProcessor() {
	Lh.stopChan <- struct{}{}
}

// InsertLog Function
func (lh *LogHandler) InsertLog(data interface{}) {
	lh.logChan <- data
}

// logProcessingRoutine Function
func (lh *LogHandler) logProcessingRoutine() {
	for {
		select {
		case l, ok := <-lh.logChan:
			if !ok {
				log.Printf("[Error] Unable to process log")
			}

			// Check new log's type
			switch l.(type) {
			case *protobuf.Log:
				processAccessLog(l.(*protobuf.Log))
			}

		case <-lh.stopChan:
			return
		}
	}
}

// processAccessLog Function
func processAccessLog(al *protobuf.Log) {
	// Send AccessLog to exporter first
	exporter.InsertAccessLog(al)

	// Then send AccessLog to metrics
	metrics.InsertAccessLog(al)
}
