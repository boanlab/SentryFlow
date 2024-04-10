// SPDX-License-Identifier: Apache-2.0

package main

import (
	"SentryFlow/protobuf"
	"flag"
	"fmt"
	"log"
	"log-client/client"
	"log-client/common"
	"os"
	"os/signal"
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

	// get arguments
	logCfgPtr := flag.String("logCfg", "stdout", "Output location for logs, {path|stdout|none}")
	metricCfgPtr := flag.String("metricCfg", "stdout", "Output location for envoy metrics and api metrics, {path|stdout|none}")
	metricFilterPtr := flag.String("metricFilter", "envoy", "Filter for what kinds of envoy and api metric to receive, {policy|envoy|api}")
	flag.Parse()

	if *logCfgPtr == "none" && *metricCfgPtr == "none" {
		flag.PrintDefaults()
		return
	}

	if cfg.LogCfg != "" {
		*logCfgPtr = cfg.LogCfg
	}
	if cfg.MetricCfg != "" {
		*metricCfgPtr = cfg.MetricCfg
	}
	if cfg.MetricFilter != "" {
		*metricFilterPtr = cfg.MetricFilter
	}

	if *metricFilterPtr != "all" && *metricFilterPtr != "envoy" && *metricFilterPtr != "api" {
		flag.PrintDefaults()
		return
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
	log.Printf("[gRPC] Successfully connected to %s for AccessLog", addr)

	// Create a client for the SentryFlow service
	sfClient := protobuf.NewSentryFlowClient(conn)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not find hostname: %v", err)
	}

	// Define the client information
	clientInfo := &protobuf.ClientInfo{
		HostName: hostname,
	}

	logClient := client.NewClient(sfClient, *logCfgPtr, *metricCfgPtr, *metricFilterPtr, clientInfo)

	if *logCfgPtr != "none" {
		go logClient.LogRoutine(*logCfgPtr)
		fmt.Printf("Started to watch logs\n")
	}

	if *metricCfgPtr != "none" {
		if *metricFilterPtr == "all" || *metricFilterPtr == "envoy" {
			go logClient.EnvoyMetricRoutine(*metricCfgPtr)
			fmt.Printf("Started to watch envoy metrics\n")
		}

		if *metricFilterPtr == "all" || *metricFilterPtr == "api" {
			go logClient.APIMetricRoutine(*metricCfgPtr)
			fmt.Printf("Started to watch api metrics\n")
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	close(logClient.Done)
}
