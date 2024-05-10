// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"fmt"
	"net"
	"sync"

	"github.com/5gsec/SentryFlow/config"
	"github.com/5gsec/SentryFlow/protobuf"

	"log"

	"google.golang.org/grpc"
)

// == //

// ExpH global reference for Exporter Handler
var ExpH *ExpHandler

// init Function
func init() {
	ExpH = NewExporterHandler()
}

// ExpHandler structure
type ExpHandler struct {
	exporterService net.Listener
	grpcServer      *grpc.Server
	grpcService     *ExpService

	apiLogExporters       []*apiLogStreamInform
	apiMetricsExporters   []*apiMetricStreamInform
	envoyMetricsExporters []*envoyMetricsStreamInform

	exporterLock sync.Mutex

	exporterAPILogs    chan *protobuf.APILog
	exporterAPIMetrics chan *protobuf.APIMetrics
	exporterMetrics    chan *protobuf.EnvoyMetrics

	statsPerLabel     map[string]StatsPerLabel
	statsPerLabelLock sync.RWMutex

	stopChan chan struct{}
}

// ExpService Structure
type ExpService struct {
	protobuf.UnimplementedSentryFlowServer
}

// == //

// NewExporterHandler Function
func NewExporterHandler() *ExpHandler {
	exp := &ExpHandler{
		grpcService: new(ExpService),

		apiLogExporters:       make([]*apiLogStreamInform, 0),
		apiMetricsExporters:   make([]*apiMetricStreamInform, 0),
		envoyMetricsExporters: make([]*envoyMetricsStreamInform, 0),

		exporterLock: sync.Mutex{},

		exporterAPILogs:    make(chan *protobuf.APILog),
		exporterAPIMetrics: make(chan *protobuf.APIMetrics),
		exporterMetrics:    make(chan *protobuf.EnvoyMetrics),

		statsPerLabel:     make(map[string]StatsPerLabel),
		statsPerLabelLock: sync.RWMutex{},

		stopChan: make(chan struct{}),
	}

	return exp
}

// == //

// StartExporter Function
func StartExporter(wg *sync.WaitGroup) bool {
	// Make a string with the given exporter address and port
	exporterService := fmt.Sprintf("%s:%s", config.GlobalConfig.ExporterAddr, config.GlobalConfig.ExporterPort)

	// Start listening gRPC port
	expService, err := net.Listen("tcp", exporterService)
	if err != nil {
		log.Printf("[Exporter] Failed to listen at %s: %v", exporterService, err)
		return false
	}
	ExpH.exporterService = expService

	log.Printf("[Exporter] Listening Exporter gRPC services (%s)", exporterService)

	// Create gRPC server
	gRPCServer := grpc.NewServer()
	ExpH.grpcServer = gRPCServer

	protobuf.RegisterSentryFlowServer(gRPCServer, ExpH.grpcService)

	log.Printf("[Exporter] Initialized Exporter gRPC services")

	// Serve gRPC Service
	go ExpH.grpcServer.Serve(ExpH.exporterService)

	log.Printf("[Exporter] Serving Exporter gRPC services (%s)", exporterService)

	// Export APILogs
	go ExpH.exportAPILogs(wg)

	log.Printf("[Exporter] Exporting API logs through gRPC services")

	// Export APIMetrics
	go ExpH.exportAPIMetrics(wg)

	log.Printf("[Exporter] Exporting API metrics through gRPC services")

	// Export EnvoyMetrics
	go ExpH.exportEnvoyMetrics(wg)

	log.Printf("[Exporter] Exporting Envoy metrics through gRPC services")

	// Start Export Time Ticker Routine
	go AggregateAPIMetrics()
	go CleanUpOutdatedStats()

	return true
}

// StopExporter Function
func StopExporter() bool {
	// One for exportAPILogs
	ExpH.stopChan <- struct{}{}

	// One for exportAPIMetrics
	ExpH.stopChan <- struct{}{}

	// One for exportEnvoyMetrics
	ExpH.stopChan <- struct{}{}

	// Stop gRPC server
	ExpH.grpcServer.GracefulStop()

	log.Printf("[Exporter] Gracefully stopped Exporter gRPC services")

	return true
}

// == //

// exportAPILogs Function
func (exp *ExpHandler) exportAPILogs(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case apiLog, ok := <-exp.exporterAPILogs:
			if !ok {
				log.Printf("[Exporter] Failed to fetch APIs from APIs channel")
				wg.Done()
				return
			}

			if err := exp.SendAPILogs(apiLog); err != nil {
				log.Printf("[Exporter] Failed to export API Logs: %v", err)
			}

		case <-exp.stopChan:
			wg.Done()
			return
		}
	}
}

// exportAPIMetrics Function
func (exp *ExpHandler) exportAPIMetrics(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case apiMetrics, ok := <-exp.exporterAPIMetrics:
			if !ok {
				log.Printf("[Exporter] Failed to fetch metrics from API Metrics channel")
				wg.Done()
				return
			}
			if err := exp.SendAPIMetrics(apiMetrics); err != nil {
				log.Printf("[Exporter] Failed to export API metrics: %v", err)
			}

		case <-exp.stopChan:
			wg.Done()
			return
		}
	}
}

// exportEnvoyMetrics Function
func (exp *ExpHandler) exportEnvoyMetrics(wg *sync.WaitGroup) {
	wg.Add(1)

	for {
		select {
		case evyMetrics, ok := <-exp.exporterMetrics:
			if !ok {
				log.Printf("[Exporter] Failed to fetch metrics from Envoy Metrics channel")
				wg.Done()
				return
			}

			if err := exp.SendEnvoyMetrics(evyMetrics); err != nil {
				log.Printf("[Exporter] Failed to export Envoy metrics: %v", err)
			}

		case <-exp.stopChan:
			wg.Done()
			return
		}
	}
}

// == //
