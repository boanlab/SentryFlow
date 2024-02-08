package core

import (
	"context"
	"errors"
	"fmt"
	otelLogs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"log"
	"net"
	cfg "numbat/config"
)

// Oh Global reference for OtelHandler
var Oh *OtelHandler
var olh *OtelLogServer

// init Function
func init() {
	Oh = NewOtelHandler()
	olh = NewOtelLogServer()
}

// OtelHandler Structure
type OtelHandler struct {
	stopChan chan struct{}

	listener   net.Listener
	gRPCServer *grpc.Server
}

// NewOtelHandler Function
func NewOtelHandler() *OtelHandler {
	oh := &OtelHandler{
		stopChan: make(chan struct{}),
	}

	return oh
}

// InitOtelServer Function
func (oh *OtelHandler) InitOtelServer() error {
	listenAddr := fmt.Sprintf("%s:%s", cfg.GlobalCfg.OtelGRPCListenAddr, cfg.GlobalCfg.OtelGRPCListenPort)

	// Start listening
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		msg := fmt.Sprintf("unable to listen at %s: %v", listenAddr, err)
		return errors.New(msg)
	}

	// Create gRPC Server, register services
	server := grpc.NewServer()
	otelLogs.RegisterLogsServiceServer(server, olh)

	oh.listener = lis
	oh.gRPCServer = server

	log.Printf("[OpenTelemetry] Server Listening at %s", listenAddr)
	return nil
}

// StartOtelServer Function
func (oh *OtelHandler) StartOtelServer() error {
	log.Printf("[OpenTelemetry] Starting server")
	var err error
	err = nil

	// Serve is blocking function
	go func() {
		err = oh.gRPCServer.Serve(oh.listener)
		if err != nil {
			return
		}
	}()

	return err
}

// StopOtelServer Function
func (oh *OtelHandler) StopOtelServer() {
	// Gracefully cleanup
	oh.stopChan <- struct{}{}

	// Gracefully stop gRPC Server
	oh.gRPCServer.GracefulStop()

	log.Printf("[OpenTelemetry] Stopped server")
}

// OtelLogServer structure
type OtelLogServer struct {
	otelLogs.UnimplementedLogsServiceServer
}

// NewOtelLogServer Function
func NewOtelLogServer() *OtelLogServer {
	return new(OtelLogServer)
}

// Export Function
func (ols *OtelLogServer) Export(_ context.Context, req *otelLogs.ExportLogsServiceRequest) (*otelLogs.ExportLogsServiceResponse, error) {
	// This is for Log.Export in OpenTelemetry format
	als := GenerateAccessLogs(req.String())

	for _, al := range als {
		Lh.InsertLog(al)
	}

	// For now, we will not consider partial success
	ret := otelLogs.ExportLogsServiceResponse{
		PartialSuccess: nil,
	}

	return &ret, nil
}
