// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"io"
	"log"
	"log-client/common"
	"os"
	"sentryflow/protobuf"
)

func main() {
	// Load environment variables
	cfg, err := common.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Construct address and start listening
	addr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Start serving gRPC server
	log.Printf("[gRPC] Successfully connected to %s", addr)

	// Create a client for the SentryFlow service
	client := protobuf.NewSentryFlowClient(conn)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not find hostname: %v", err)
	}

	// Define the client information
	clientInfo := &protobuf.ClientInfo{
		HostName: hostname,
	}

	// Contact the server and print out its response
	stream, err := client.GetLog(context.Background(), clientInfo)
	if err != nil {
		log.Fatalf("could not get log: %v", err)
	}

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to receive log: %v", err)
		}
		log.Printf("[Client] Received log: %v", data)
	}
}
