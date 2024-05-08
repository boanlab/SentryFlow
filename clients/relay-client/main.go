// SPDX-License-Identifier: Apache-2.0

package main

import (
	pb "SentryFlow/protobuf"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"relay-client/common"
	"syscall"

	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
)

func main() {
	// Load environment variables
	cfg, err := common.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Construct address and start listening
	addr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)
	relayAddr := fmt.Sprintf("%s:%d", "10.10.0.167", 50051)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	relayConn, err := grpc.Dial(relayAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer relayConn.Close()

	// Start serving gRPC server
	log.Printf("[gRPC] Successfully connected to %s for AccessLog", addr)
	log.Printf("[gRPC] Successfully connected to %s for AccessLog", relayAddr)

	// Create a client for the SentryFlow service
	client := pb.NewSentryFlowClient(conn)
	relayClient := pb.NewSentryFlowClient(relayConn)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not find hostname: %v", err)
	}

	// Define the client information
	clientInfo := &pb.ClientInfo{
		HostName: hostname,
	}

	// Contact the server and print out its response
	accessLogStream, err := client.GetAPILog(context.Background(), clientInfo)
	if err != nil {
		log.Fatalf("could not get log: %v", err)
	}

	relayStream, err := relayClient.GetRelayLog(context.Background())
	if err != nil {
		log.Fatalf("could not get log: %v", err)
	}

	done := make(chan struct{})

	go accessLogRoutine(accessLogStream, relayStream, done)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	close(done)
}

func accessLogRoutine(stream pb.SentryFlow_GetAPILogClient, relayStream pb.SentryFlow_GetRelayLogClient, done chan struct{}) {
	for {
		select {
		default:
			data, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed to receive log: %v", err)
			}
			log.Printf("[Client] Received log: %v", data)
			relayStream.Send(data)

		case <-done:
			return
		}
	}
}
