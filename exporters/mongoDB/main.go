package main

import (
	"client-mongo/common"
	"client-mongo/db"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	protobuf "numbat/protobuf"
)

// AccessLogServer implements our gRPC server for mongoDB
type AccessLogServer struct {
	protobuf.UnimplementedLogsServer
}

// Send implements Send service for AccessLogServer
func (a AccessLogServer) Send(_ context.Context, accessLog *protobuf.AccessLog) (*protobuf.LogResponse, error) {
	err := db.Manager.InsertData(accessLog)
	if err != nil {
		log.Printf("[DB] Unable to insert data into DB: %v", err)
	}

	return &protobuf.LogResponse{}, nil
}

// main is the entrypoint of this program
func main() {
	// Init DB
	_, err := db.New()
	if err != nil {
		log.Fatalf("Unable to intialize DB: %v", err)
	}

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
