package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
)

type config struct {
	listenAddr string
	listenPort int
}

// LogServer is for exporting log handler
type LogServer struct {
	logs.UnimplementedLogsServiceServer
}

// Export interface function for LogServiceServer
func (ls LogServer) Export(c context.Context, request *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	log.Printf("[LOG] Received:  %v", request.String())
	for _, rl := range request.GetResourceLogs() {
		rsc := rl.GetResource()
		log.Printf("[LOG] Attrs : %v", rsc.Attributes)
	}

	// Just assume this worked fine
	ret := logs.ExportLogsServiceResponse{PartialSuccess: nil}
	return &ret, nil
}

// loadEnvVars loads environment variables and stores them as global variable
func loadEnvVars() (config, error) {
	cfg := config{}
	var err error

	// load listen address and check if valid
	cfg.listenAddr = os.Getenv("LISTEN_ADDR")
	ip := net.ParseIP(cfg.listenAddr)
	if ip == nil {
		msg := fmt.Sprintf("invalid listen address %s", cfg.listenAddr)
		return cfg, errors.New(msg)
	}
	cfg.listenAddr = ip.String()

	// load listen port and check if valid
	cfg.listenPort, err = strconv.Atoi(os.Getenv("LISTEN_PORT"))
	if err != nil {
		msg := fmt.Sprintf("invalid listen port %s: %v", os.Getenv("LISTEN_PORT"), err)
		return cfg, errors.New(msg)
	}

	return cfg, nil
}

// main is the entrypoint of this program
func main() {
	// load environment variables
	cfg, err := loadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Start listening 4317
	addr := fmt.Sprintf("%s:%d", cfg.listenAddr, cfg.listenPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Could not start up server at %s: %v", lis, err)
	}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()

	// Register log collector server
	logServer := LogServer{}
	logs.RegisterLogsServiceServer(grpcServer, logServer)

	// Start serving gRPC requests
	log.Printf("Starting to serve on %s", addr)

	// Start listening gRPC Server for OTEL but with debugging
	log.Printf("Starting to serve...")
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatal("Could not start up gRPC server: ", err)
	}
}
