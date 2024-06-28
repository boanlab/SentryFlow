// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"SentryFlow/protobuf"
	"neo4j-client/client"
	"neo4j-client/config"

	"google.golang.org/grpc"
)

func main() {
	// Load environment variables
	cfg, err := config.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Get arguments
	nodeLevelPtr := flag.String("nodeLevel", "simple", "NodeLevel for storing API logs, {simple|detail}")
	edgeLevelPtr := flag.String("edgeLevel", "simple", "EdgeLevel for storing API logs, {simple|detail}")
	neo4jURIPtr := flag.String("neo4jHost", "", "Neo4j Host")
	neo4jUsernamePtr := flag.String("neo4jId", "", "Neo4j Id")
	neo4jPasswordPtr := flag.String("neo4jPassword", "", "Neo4j Password")
	flag.Parse()

	if cfg.NodeLevel != "" {
		*nodeLevelPtr = cfg.NodeLevel
	}
	if cfg.EdgeLevel != "" {
		*edgeLevelPtr = cfg.EdgeLevel
	}
	if cfg.Neo4jURI != "" {
		*neo4jURIPtr = cfg.Neo4jURI
	}
	if cfg.Neo4jUsername != "" {
		*neo4jUsernamePtr = cfg.Neo4jUsername
	}
	if cfg.Neo4jPassword != "" {
		*neo4jPasswordPtr = cfg.Neo4jPassword
	}

	if *nodeLevelPtr == "" && *edgeLevelPtr == "" {
		flag.PrintDefaults()
		return
	}

	// == //

	// Construct a string "ServerAddr:ServerPort"
	addr := fmt.Sprintf("%s:%d", cfg.ServerAddr, cfg.ServerPort)

	// Connect to the gRPC server of SentryFlow
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("[gRPC] Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	// Connected to the gRPC server
	log.Printf("[gRPC] Started to collect Logs from %s", addr)

	// Define clientInfo
	clientInfo := &protobuf.ClientInfo{
		HostName: cfg.Hostname,
	}

	// Create a gRPC client for the SentryFlow service
	sfClient := protobuf.NewSentryFlowClient(conn)

	// Create a log client with the gRPC client
	logClient := client.NewClient(sfClient, clientInfo, *nodeLevelPtr, *edgeLevelPtr, *neo4jURIPtr, *neo4jUsernamePtr, *neo4jPasswordPtr)
	if logClient == nil {
		log.Fatalf("[gRPC] Failed to create gRPC client")
		return
	}

	go logClient.APILogRoutine()
	fmt.Printf("[APILog] Started to watch API logs\n")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	logClient.DbHandler.Close()
	close(logClient.Done)
}
