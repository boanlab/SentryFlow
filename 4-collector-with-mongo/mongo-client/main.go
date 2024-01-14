package main

import (
	"log"
	"mongo-client/db"
)

// main is the entrypoint of this program
func main() {
	// Init DB
	_, err := db.New()
	if err != nil {
		log.Fatalf("Unable to intialize DB: %v", err)
	}
}
