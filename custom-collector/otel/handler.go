package otel

import (
	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	"net"
)

// Handler is the handler for OTEL collector
type Handler struct {
	addr      string
	lis       net.Listener
	server    *grpc.Server
	logServer LogServer
}

// NewHandler creates a handler for OTEL collector
func NewHandler(addr string) *Handler {
	// Create a handler object
	h := Handler{}
	var err error

	// Dump address
	h.addr = addr

	// Start listening 4317
	h.lis, err = net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[OTEL] Could not start up server at %s: %v", h.lis, err)
	}

	// Create a new gRPC server
	h.server = grpc.NewServer()

	// Register log collector server for Logging
	h.logServer = LogServer{}
	logs.RegisterLogsServiceServer(h.server, h.logServer)

	return &h
}

// Serve starts the gRPC server for OTEL collection
func (h *Handler) Serve() error {
	log.Printf("[OTEL] Started to serve on %s", h.addr)
	err := h.server.Serve(h.lis)
	if err != nil {
		log.Fatalf("[OTEL] Could not start up gRPC server at %s: %v", h.addr, err)
		return err
	}

	return nil
}
