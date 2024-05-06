// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"log"
	"sync"

	"github.com/5gsec/SentryFlow/exporter"
	"github.com/5gsec/SentryFlow/protobuf"
)

// == //

// LogH global reference for Log Handler
var LogH *LogHandler

// init Function
func init() {
	LogH = NewLogHandler()
}

// LogHandler Structure
type LogHandler struct {
	stopChan chan struct{}

	apiLogChan  chan interface{}
	metricsChan chan interface{}
}

// aggregationLog Structure
type aggregationLog struct {
	Labels      map[string]string
	Annotations map[string]string

	Logs []*protobuf.APILog
}

// NewLogHandler Structure
func NewLogHandler() *LogHandler {
	lh := &LogHandler{
		stopChan: make(chan struct{}),

		apiLogChan:  make(chan interface{}),
		metricsChan: make(chan interface{}),
	}

	return lh
}

// == //

// StartLogProcessor Function
func StartLogProcessor(wg *sync.WaitGroup) bool {
	// handle API logs
	go ProcessAPILogs(wg)

	// handle metrics
	go ProcessMetrics(wg)

	log.Print("[LogProcessor] Started Log Processor")

	return true
}

// StopLogProcessor Function
func StopLogProcessor() bool {
	// One for ProcessAPILogs
	LogH.stopChan <- struct{}{}

	// One for ProcessMetrics
	LogH.stopChan <- struct{}{}

	log.Print("[LogProcessor] Stopped Log Processor")

	return true
}

// == //

// InsertAPILog Function
func InsertAPILog(data interface{}) {
	LogH.apiLogChan <- data
}

// ProcessLogs Function
func ProcessAPILogs(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case logType, ok := <-LogH.apiLogChan:
			if !ok {
				log.Print("[LogProcessor] Unable to process an API log")
			}

			switch logType.(type) {
			case *protobuf.APILog:
				go exporter.InsertAPILog(logType.(*protobuf.APILog))

				// Send API for Further Analysis
				go AnalyzeAPI(logType.(*protobuf.APILog).Path)
			}

		case <-LogH.stopChan:
			wg.Done()
			return
		}
	}
}

// InsertMetrics Function
func InsertMetrics(data interface{}) {
	LogH.metricsChan <- data
}

// ProcessMetrics Function
func ProcessMetrics(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case logType, ok := <-LogH.metricsChan:
			if !ok {
				log.Print("[LogProcessor] Unable to process metrics")
			}

			switch logType.(type) {
			case *protobuf.EnvoyMetrics:
				go exporter.InsertEnvoyMetrics(logType.(*protobuf.EnvoyMetrics))
			}

		case <-LogH.stopChan:
			wg.Done()
			return
		}
	}
}

// == //
