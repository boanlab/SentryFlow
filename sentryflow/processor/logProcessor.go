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

	// handle Envoy metrics
	go ProcessEnvoyMetrics(wg)

	log.Print("[LogProcessor] Started Log Processors")

	return true
}

// StopLogProcessor Function
func StopLogProcessor() bool {
	// One for ProcessAPILogs
	LogH.stopChan <- struct{}{}

	// One for ProcessMetrics
	LogH.stopChan <- struct{}{}

	log.Print("[LogProcessor] Stopped Log Processors")

	return true
}

// == //

// ProcessAPILogs Function
func ProcessAPILogs(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case logType, ok := <-LogH.apiLogChan:
			if !ok {
				log.Print("[LogProcessor] Failed to process an API log")
			}

			go AnalyzeAPI(logType.(*protobuf.APILog).Path)
			go exporter.InsertAPILog(logType.(*protobuf.APILog))

		case <-LogH.stopChan:
			wg.Done()
			return
		}
	}
}

// InsertAPILog Function
func InsertAPILog(data interface{}) {
	LogH.apiLogChan <- data
}

// ProcessEnvoyMetrics Function
func ProcessEnvoyMetrics(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case logType, ok := <-LogH.metricsChan:
			if !ok {
				log.Print("[LogProcessor] Failed to process Envoy metrics")
			}

			go exporter.InsertEnvoyMetrics(logType.(*protobuf.EnvoyMetrics))

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

// == //
