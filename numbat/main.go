package main

import (
	"fmt"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	cfg "numbat/config"
	core "numbat/core"
	"numbat/exporter"
	"numbat/otel"
)

// main is the entrypoint of this program
func main() {
	err := cfg.LoadConfig()
	if err != nil {
		log.Fatalf("Unable to load config: %v", err)
	}

	core.Numbat()

	// Initialize exporter
	ex := exporter.NewHandler()
	if err != nil {
		log.Fatalf("Could not initialize exporter: %v", err)
	}
	ex.RunExporters(signalCh)

	// Initialize OTEL gRPC server
	addr := fmt.Sprintf("%s:%d", cfg.ListenAddr, cfg.ListenPort)
	oh := otel.NewHandler(addr)

	// Start serving OTEL
	err = oh.Serve()
	if err != nil {
		log.Fatalf("Could not serve OTEL gRPC server: %v", err)
	}
}
