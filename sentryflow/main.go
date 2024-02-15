// SPDX-License-Identifier: Apache-2.0

package main

import (
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
	cfg "sentryflow/config"
	core "sentryflow/core"
)

// main is the entrypoint of this program
func main() {
	err := cfg.LoadConfig()
	if err != nil {
		log.Fatalf("[SentryFlow] Unable to load config: %v", err)
	}

	core.SentryFlow()
}
