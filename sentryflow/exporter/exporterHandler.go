// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/protobuf"
	"github.com/5GSEC/SentryFlow/types"

	"github.com/emicklei/go-restful/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// Exp global reference for Exporter Handler
var Exp *Handler

// init Function
func init() {
	Exp = NewExporterHandler()
}

// Handler structure
type Handler struct {
	baseExecutionID uint64
	currentLogCount uint64
	stopChan        chan struct{}
	lock            sync.Mutex
	exporters       []*Inform
	metricExporters []*metricStreamInform
	exporterLock    sync.Mutex
	exporterLogs    chan *protobuf.APILog
	exporterMetrics chan *protobuf.EnvoyMetric

	listener   net.Listener
	gRPCServer *grpc.Server
}

// Inform structure
type Inform struct {
	stream    protobuf.SentryFlow_GetLogServer
	error     chan error
	Hostname  string
	IPAddress string
}

type metricStreamInform struct {
	metricStream protobuf.SentryFlow_GetEnvoyMetricsServer
	error        chan error
	Hostname     string
	IPAddress    string
}

// NewExporterHandler Function
func NewExporterHandler() *Handler {
	exp := &Handler{
		baseExecutionID: uint64(time.Now().UnixMicro()),
		currentLogCount: 0,
		exporters:       make([]*Inform, 0),
		stopChan:        make(chan struct{}),
		lock:            sync.Mutex{},
		exporterLock:    sync.Mutex{},
		exporterLogs:    make(chan *protobuf.APILog),
		exporterMetrics: make(chan *protobuf.EnvoyMetric),
	}

	return exp
}

// InsertAccessLog Function
func InsertAccessLog(al *protobuf.APILog) {
	// Avoid race condition for currentLogCount, otherwise we might have duplicate IDs
	Exp.lock.Lock()
	al.Id = Exp.baseExecutionID + Exp.currentLogCount
	Exp.currentLogCount++
	Exp.lock.Unlock()

	saveAccessLog(al) // go routine??
	als, _ := MDB.AggregatedAccessLogSelect()

	for key, val := range als {
		for _, byte := range val {
			var eal protobuf.APILog
			if err := proto.Unmarshal(byte, &eal); err != nil {
				log.Printf("error unmarshaling AccessLog: %w", err)
			}
		}
	}

	Exp.exporterLogs <- al
}

func saveAccessLog(al *protobuf.APILog) {
	curLabels := al.SrcLabel
	curAnnotations := al.SrcAnnotation

	lbByteData, err := json.Marshal(curLabels)
	if err != nil {
		fmt.Println("Error encoding first map:", err)
		return
	}

	anByteData, err := json.Marshal(curAnnotations)
	if err != nil {
		fmt.Println("Error encoding second map:", err)
		return
	}

	alByteData, err := proto.Marshal(al)
	if err != nil {
		log.Printf("[Exporter]AccessLog marshaling error: ", err)
	}

	curData := types.DbAccessLogType{
		Labels:      lbByteData,
		Annotations: anByteData,
		AccessLog:   alByteData,
	}

	MDB.AccessLogInsert(curData)
}

// InsertEnvoyMetric Function
func InsertEnvoyMetric(em *protobuf.EnvoyMetric) {
	Exp.exporterMetrics <- em
}

// InitExporterServer Function
func (exp *Handler) InitExporterServer() error {
	listenAddr := fmt.Sprintf("%s:%s", cfg.GlobalCfg.CustomExportListenAddr, cfg.GlobalCfg.CustomExportListenPort)

	// Start listening
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		msg := fmt.Sprintf("unable to listen at %s: %v", listenAddr, err)
		return errors.New(msg)
	}

	// Create gRPC server
	server := grpc.NewServer()
	protobuf.RegisterSentryFlowServer(server, exs)

	exp.listener = lis
	exp.gRPCServer = server

	log.Printf("[Exporter] Exporter listening at %s", listenAddr)
	return nil
}

// StartExporterServer Function
func (exp *Handler) StartExporterServer(wg *sync.WaitGroup) error {
	log.Printf("[Exporter] Starting exporter server")
	var err error
	err = nil

	go exp.exportRoutine(wg)

	go func() {
		wg.Add(1)
		// Serve is blocking function
		err = exp.gRPCServer.Serve(exp.listener)
		if err != nil {
			wg.Done()
			return
		}

		wg.Done()
	}()

	return err
}

// exportRoutine Function
func (exp *Handler) exportRoutine(wg *sync.WaitGroup) {
	wg.Add(1)
	log.Printf("[Exporter] Starting export routine")

routineLoop:
	for {
		select {
		// @todo add more channels for this
		case al, ok := <-exp.exporterLogs:
			if !ok {
				log.Printf("[Exporter] Log exporter channel closed")
				break routineLoop
			}

			err := exp.sendLogs(al)
			if err != nil {
				log.Printf("[Exporter] Log exporting failed %v:", err)
			}

		case em, ok := <-exp.exporterMetrics:
			if !ok {
				log.Printf("[Exporter] EnvoyMetric exporter channel closed")
				break routineLoop
			}

			err := exp.sendMetrics(em)
			if err != nil {
				log.Printf("[Exporter] Metric exporting failed %v:", err)
			}

		case <-exp.stopChan:
			break routineLoop
		}
	}

	defer wg.Done()
	return
}

// sendLogs Function
func (exp *Handler) sendLogs(l *protobuf.APILog) error {
	exp.exporterLock.Lock()
	defer exp.exporterLock.Unlock()

	// iterate and send logs
	failed := 0
	total := len(exp.exporters)
	for _, exporter := range exp.exporters {
		curRetry := 0

		// @todo: make max retry count per logs using config
		// @todo: make max retry count per single exporter before removing the exporter using config
		var err error
		for curRetry < 3 {
			err = exporter.stream.Send(l)
			if err != nil {
				log.Printf("[Exporter] Unable to send log to %s(%s) retry=%d/%d: %v",
					exporter.Hostname, exporter.IPAddress, curRetry, 3, err)
				curRetry++
			} else {
				break
			}
		}

		// Count failed
		if err != nil {
			failed++
		}
	}

	// notify failed count
	if failed != 0 {
		msg := fmt.Sprintf("unable to send logs properly %d/%d failed", failed, total)
		return errors.New(msg)
	}

	return nil
}

// sendMetrics Function
func (exp *Handler) sendMetrics(l *protobuf.EnvoyMetric) error {
	exp.exporterLock.Lock()
	defer exp.exporterLock.Unlock()

	// iterate and send logs
	failed := 0
	total := len(exp.metricExporters)
	for _, exporter := range exp.metricExporters {
		curRetry := 0

		// @todo: make max retry count per logs using config
		// @todo: make max retry count per single exporter before removing the exporter using config
		var err error
		for curRetry < 3 {
			err = exporter.metricStream.Send(l)
			if err != nil {
				log.Printf("[Exporter] Unable to send metric to %s(%s) retry=%d/%d: %v",
					exporter.Hostname, exporter.IPAddress, curRetry, 3, err)
				curRetry++
			} else {
				break
			}
		}

		// Count failed
		if err != nil {
			failed++
		}
	}

	// notify failed count
	if failed != 0 {
		msg := fmt.Sprintf("unable to send metrics properly %d/%d failed", failed, total)
		return errors.New(msg)
	}

	return nil
}

// StopExporterServer Function
func (exp *Handler) StopExporterServer() {
	// Gracefully stop all client connections
	exp.stopChan <- struct{}{}

	// Gracefully stop gRPC Server
	exp.gRPCServer.GracefulStop()

	log.Printf("[Exporter] Stopped exporter server")
}
