// SPDX-License-Identifier: Apache-2.0

package core

import (
	"github.com/5GSEC/sentryflow/exporter"
	"github.com/5GSEC/sentryflow/metrics"
	"github.com/5GSEC/sentryflow/protobuf"
	"github.com/5GSEC/sentryflow/types"
	"log"
	"strconv"
	"strings"
	"sync"
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
func StartLogProcessor(wg *sync.WaitGroup) {
	go Lh.logProcessingRoutine(wg)
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
func (lh *LogHandler) logProcessingRoutine(wg *sync.WaitGroup) {
	wg.Add(1)
	for {
		select {
		case l, ok := <-lh.logChan:
			if !ok {
				log.Printf("[Error] Unable to process log")
			}

			// Check new log's type
			switch l.(type) {
			case *protobuf.APILog:
				go processAccessLog(l.(*protobuf.APILog))
			}

		case <-lh.stopChan:
			wg.Done()
			return
		}
	}
}

// processAccessLog Function
func processAccessLog(al *protobuf.APILog) {
	// Send AccessLog to exporter first
	exporter.InsertAccessLog(al)

	// Then send AccessLog to metrics
	metrics.InsertAccessLog(al)
}

// GenerateAccessLogs Function
func GenerateAccessLogs(logText string) []*protobuf.APILog {
	// @todo this needs more optimization, this code is kind of messy
	// Create an array of AccessLogs for returning gRPC comm
	var index int
	ret := make([]*protobuf.APILog, 0)

	// Preprocess redundant chars
	logText = strings.ReplaceAll(logText, `\"`, "")
	logText = strings.ReplaceAll(logText, `}`, "")

	// Split logs by log_records, this is single access log instance
	parts := strings.Split(logText, "log_records")
	if len(parts) == 0 {
		return nil
	}

	// Ignore the first entry, this was the metadata "resource_logs:{resource:{ scope_logs:{" part.
	for _, al := range parts[0:] {
		if len(al) == 0 {
			continue
		}

		index = strings.Index(al, "string_value:\"")
		if index == -1 {
			continue
		}

		result := al[index+len("string_value:\""):]
		words := strings.Fields(result)

		method := words[1]
		path := words[2]
		protocolName := words[3]
		timeStamp := words[0]
		resCode, _ := strconv.ParseInt(words[4], 10, 64)

		srcInform := words[21]
		dstInform := words[20]

		var srcIP string
		var dstIP string
		var srcPort string
		var dstPort string
		var colonIndex int

		// Extract the left and right words based on the colon delimiter (ADDR:PORT)
		colonIndex = strings.LastIndex(srcInform, ":")
		if colonIndex > 0 && colonIndex < len(srcInform)-1 {
			srcIP = strings.TrimSpace(srcInform[:colonIndex])
			srcPort = strings.TrimSpace(srcInform[colonIndex+1:])
		}

		colonIndex = strings.LastIndex(dstInform, ":")
		if colonIndex > 0 && colonIndex < len(dstInform)-1 {
			dstIP = strings.TrimSpace(dstInform[:colonIndex])
			dstPort = strings.TrimSpace(dstInform[colonIndex+1:])
		}

		// Lookup using K8s API
		src := LookupNetworkedResource(srcIP)
		dst := LookupNetworkedResource(dstIP)

		// Create AccessLog in our gRPC format
		cur := protobuf.APILog{
			TimeStamp:    timeStamp,
			Id:           0, //  do 0 for now, we are going to write it later
			SrcNamespace: src.Namespace,
			SrcName:      src.Name,
			SrcLabel:     src.Labels,
			SrcIP:        srcIP,
			SrcPort:      srcPort,
			SrcType:      types.K8sResourceTypeToString(src.Type),
			DstNamespace: dst.Namespace,
			DstName:      dst.Name,
			DstLabel:     dst.Labels,
			DstIP:        dstIP,
			DstPort:      dstPort,
			DstType:      types.K8sResourceTypeToString(dst.Type),
			Protocol:     protocolName,
			Method:       method,
			Path:         path,
			ResponseCode: int32(resCode),
		}

		ret = append(ret, &cur)
	}

	return ret
}
