// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"log"
	"sync"

	"github.com/5gsec/SentryFlow/config"
)

// == //

// APIA Local reference for API Analyzer
var APIA *Analyzer

// init function
func init() {
	APIA = NewAPIAnalyzer()
}

// Analyzer Structure
type Analyzer struct {
	stopChan chan struct{}

	apiLog      chan string
	apiLogs     []string
	apiLogsLock sync.Mutex
}

// NewAPIAnalyzer Function
func NewAPIAnalyzer() *Analyzer {
	ret := &Analyzer{
		apiLog:      make(chan string),
		apiLogs:     []string{},
		apiLogsLock: sync.Mutex{},
	}
	return ret
}

// StartAPIAnalyzer Function
func StartAPIAnalyzer(wg *sync.WaitGroup) bool {
	// keep analyzing given APIs
	go analyzeAPIs(wg)

	log.Print("[APIAnalyzer] Started API Analyzer")

	return true
}

// AnalyzeAPI Function
func AnalyzeAPI(api string) {
	APIA.apiLog <- api
}

// StopAPIAnalyzer Function
func StopAPIAnalyzer() bool {
	APIA.stopChan <- struct{}{}

	log.Print("[APIAnalyzer] Stopped API Analyzer")

	return true
}

// == //

// analyzeAPIs Function
func analyzeAPIs(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case api, ok := <-APIA.apiLog:
			if !ok {
				continue
			}

			APIA.apiLogsLock.Lock()

			APIA.apiLogs = append(APIA.apiLogs, api)

			if len(APIA.apiLogs) > config.GlobalConfig.AIEngineBatchSize {
				ClassifyAPIs(APIA.apiLogs)
				APIA.apiLogs = []string{}
			}

			APIA.apiLogsLock.Unlock()
		case <-APIA.stopChan:
			wg.Done()
			return
		}
	}
}

// == //
