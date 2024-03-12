// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"errors"
	"fmt"
	cfg "github.com/5GSEC/sentryflow/config"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

// Ch global reference for Collector Handler
var Ch *Handler

// init Function
func init() {
	Ch = NewCollectorHandler()
}

// Handler Structure
type Handler struct {
	collectors []collectorInterface

	listener   net.Listener
	grpcServer *grpc.Server

	wg sync.WaitGroup
}

// NewCollectorHandler Function
func NewCollectorHandler() *Handler {
	ch := &Handler{
		collectors: make([]collectorInterface, 0),
	}

	return ch
}

// InitGRPCServer Function
func (h *Handler) InitGRPCServer() error {
	listenAddr := fmt.Sprintf("%s:%s", cfg.GlobalCfg.OtelGRPCListenAddr, cfg.GlobalCfg.OtelGRPCListenPort)

	// Start listening
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		msg := fmt.Sprintf("unable to listen at %s: %v", listenAddr, err)
		return errors.New(msg)
	}

	// Create gRPC Server, register services
	server := grpc.NewServer()

	h.listener = lis
	h.grpcServer = server

	// initialize collectors
	err = h.initCollectors()
	if err != nil {
		log.Printf("[Collector] Unable to initialize collector: %v", err)
	}

	// register services
	h.registerServices()

	log.Printf("[Collector] Server listening at %s", listenAddr)
	return nil
}

// initCollectors Function
func (h *Handler) initCollectors() error {
	// @todo make configuration determine which collector to start or not
	h.collectors = append(h.collectors, newOtelLogServer())
	h.collectors = append(h.collectors, newEnvoyMetricsServer())
	h.collectors = append(h.collectors, newEnvoyAccessLogsServer())

	return nil
}

// registerServices Function
func (h *Handler) registerServices() {
	for _, col := range h.collectors {
		col.registerService(h.grpcServer)
		log.Printf("[Collector] Successfully registered services")
	}
}

// Serve Function
func (h *Handler) Serve() error {
	log.Printf("[Collector] Starting gRPC server")
	return h.grpcServer.Serve(h.listener)
}

// Stop Function
func (h *Handler) Stop() {
	log.Printf("[Collector] Stopped gRPC server")
	h.grpcServer.GracefulStop()
}
