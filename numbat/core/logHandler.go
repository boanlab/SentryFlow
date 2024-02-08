package core

import (
	otel "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"log"
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
			case *otel.ExportLogsServiceRequest:
				processAccessLog(l.(*otel.ExportLogsServiceRequest))
			}

		case <-lh.stopChan:
		}
	}
}

// processAccessLog Function
func processAccessLog(al *otel.ExportLogsServiceRequest) {

}
