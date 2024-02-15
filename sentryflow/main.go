// SPDX-License-Identifier: Apache-2.0

package main

import (
	cfg "github.com/5GSEC/sentryflow/config"
	core "github.com/5GSEC/sentryflow/core"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
)

// main is the entrypoint of this program
func main() {
	err := cfg.LoadConfig()
	if err != nil {
		log.Fatalf("[SentryFlow] Unable to load config: %v", err)
	}

	core.SentryFlow()
}
