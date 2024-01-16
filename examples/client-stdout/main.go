package main

import (
	"client-stdout/common"
	protobuf "client-stdout/protobuf"
	context "context"
	"fmt"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	"net"
)

// AccessLogServer implements our gRPC server for stdout
type AccessLogServer struct {
	protobuf.UnimplementedLogsServer
}

// Send implements Send service for AccessLogServer
func (a AccessLogServer) Send(_ context.Context, accessLog *protobuf.AccessLog) (*protobuf.LogResponse, error) {
	log.Println(accessLog)
	return &protobuf.LogResponse{}, nil
}

// main is the entrypoint of this program
func main() {
	// Load environment variables
	cfg, err := common.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Construct address and start listening
	addr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[gRPC] Could not start up server at %s: %v", lis, err)
	}

	// Create a new gRPC server
	server := grpc.NewServer()
	als := AccessLogServer{}
	protobuf.RegisterLogsServer(server, als)

	// Start serving gRPC server
	log.Printf("[gRPC] Started to serve on %s", addr)
	err = server.Serve(lis)
	if err != nil {
		log.Fatalf("[gRPC] Could not start up gRPC server at %s: %v", addr, err)
	}
}
