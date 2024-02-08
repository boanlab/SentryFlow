package main

import (
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	cfg "numbat/config"
	core "numbat/core"
)

// main is the entrypoint of this program
func main() {
	err := cfg.LoadConfig()
	if err != nil {
		log.Fatalf("[Numbat] Unable to load config: %v", err)
	}

	core.Numbat()
}
