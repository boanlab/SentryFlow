package main

import (
	"custom-collector/common"
	"custom-collector/exporter"
	"custom-collector/k8s"
	"custom-collector/otel"
	"fmt"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	"os"
	"os/signal"
	"syscall"
)

// main is the entrypoint of this program
func main() {
	// Signal handling for SIGTERM
	signalCh := make(chan os.Signal, 1)

	// Load environment variables
	cfg, err := common.LoadEnvVars()
	if err != nil {
		log.Fatalf("Could not load environment variables: %v", err)
	}

	// Initialize exporter
	ex := exporter.NewHandler()
	if err != nil {
		log.Fatalf("Could not initialize exporter: %v", err)
	}
	ex.RunExporters(signalCh)

	// Initialize Kubernetes handler
	kh, err := k8s.NewHandler()
	if err != nil {
		log.Fatalf("Could not initialize Kubernetes handler: %v", err)
	}
	kh.RunInformers(signalCh)

	// Initialize OTEL gRPC server
	addr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	oh := otel.NewHandler(addr)

	// Start serving OTEL
	err = oh.Serve()
	if err != nil {
		log.Fatalf("Could not serve OTEL gRPC server: %v", err)
	}

	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	<-signalCh
}
