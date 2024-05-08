// SPDX-License-Identifier: Apache-2.0

package processor

import (
	"sync"

	"github.com/5gsec/SentryFlow/config"
)

// == //

// APIA Local reference for API analyzer
var APIA *Analyzer

// init function
func init() {
	APIA = NewAPIAnalyzer()
}

// Analyzer Structure
type Analyzer struct {
	apiLog  chan string
	apiLogs []string

	analyzerLock sync.Mutex

	stopChan chan struct{}
}

// NewAPIAnalyzer Function
func NewAPIAnalyzer() *Analyzer {
	ret := &Analyzer{
		apiLogs:      []string{},
		analyzerLock: sync.Mutex{},
	}
	return ret
}

// StartAPIAnalyzer Function
func StartAPIAnalyzer(wg *sync.WaitGroup) bool {
	// keep analyzing given APIs
	go analyzeAPIs(wg)

	return true
}

// AnalyzeAPI Function
func AnalyzeAPI(api string) {
	APIA.apiLog <- api
}

// StopAPIAnalyzer Function
func StopAPIAnalyzer() bool {
	APIA.stopChan <- struct{}{}

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

			APIA.analyzerLock.Lock()

			APIA.apiLogs = append(APIA.apiLogs, api)

			if len(APIA.apiLogs) > config.GlobalConfig.AIEngineBatchSize {
				InsertAPILogsAI(APIA.apiLogs)
				APIA.apiLogs = []string{}
			}

			APIA.analyzerLock.Unlock()
		case <-APIA.stopChan:
			wg.Done()
			return
		}
	}
}

// == //

// == //
