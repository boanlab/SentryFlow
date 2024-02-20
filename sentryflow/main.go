// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/5GSEC/sentryflow/collector"
	"github.com/5GSEC/sentryflow/core"
	_ "google.golang.org/grpc/encoding/gzip" // If not set, encoding problem occurs https://stackoverflow.com/questions/74062727
	"log"
)

// main is the entrypoint of this program
func main() {
	go func() {
		core.SentryFlow()
	}()

	err := collector.Ch.InitGRPCServer()
	if err != nil {
		log.Fatalf("[Error] Unable to start collector gRPC Server: %v", err)
	}

	err = collector.Ch.Serve()
	if err != nil {
		log.Fatalf("[Error] Unable to serve gRPC Server: %v", err)
	}
}
