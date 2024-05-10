// SPDX-License-Identifier: Apache-2.0

package main

import (
	"SentryFlow/protobuf"
	"flag"
	"fmt"
	"log"
	"log-client/client"
	"log-client/config"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

// ========== //
// == Main == //
// ========== //

func main() {
	// Load environment variables
	cfg, err := config.LoadEnvVars()
	if err != nil {
		log.Fatalf("[Config] Could not load environment variables: %v", err)
	}

	// Get arguments
	logCfgPtr := flag.String("logCfg", "stdout", "Output location for API logs, {stdout|file|none}")
	metricCfgPtr := flag.String("metricCfg", "stdout", "Output location for API and Envoy metrics, {stdout|file|none}")
	metricFilterPtr := flag.String("metricFilter", "envoy", "Filter to select specific API or Envoy metrics to receive, {api|envoy}")
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

	if *metricFilterPtr != "all" && *metricFilterPtr != "api" && *metricFilterPtr != "envoy" {
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
	logClient := client.NewClient(sfClient, clientInfo, *logCfgPtr, *metricCfgPtr, *metricFilterPtr)

	if *logCfgPtr != "none" {
		go logClient.APILogRoutine(*logCfgPtr)
		fmt.Printf("[APILog] Started to watch API logs\n")
	}

	if *metricCfgPtr != "none" {
		if *metricFilterPtr == "all" || *metricFilterPtr == "api" {
			go logClient.APIMetricsRoutine(*metricCfgPtr)
			fmt.Printf("[Metric] Started to watch API Metrics\n")
		}

		if *metricFilterPtr == "all" || *metricFilterPtr == "envoy" {
			go logClient.EnvoyMetricsRoutine(*metricCfgPtr)
			fmt.Printf("[Metric] Started to watch Envoy Metrics\n")
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	close(logClient.Done)
}
