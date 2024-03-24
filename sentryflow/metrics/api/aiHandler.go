// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"log"
	"os"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/protobuf"
	"google.golang.org/grpc"
)

// ah Local reference for AI handler server
var Ah *aiHandler

// init Function
func init() {
	Ah := newAIHandler(cfg.AIEngineService, cfg.AIEngineServicePort)

	// Construct address and start listening
	addr := fmt.Sprintf("%s:%d", Ah.aiHost, Ah.aiPort)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer conn.Close()

	// Start serving gRPC server
	log.Printf("[gRPC] Successfully connected to %s for APIMetric", addr)

	client := protobuf.NewSentryFlowMetricsClient(conn)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not find hostname: %v", err)
	}

	// Define the client information
	clientInfo := &protobuf.ClientInfo{
		HostName: hostname,
	}

}

// aiHandler Structure
type aiHandler struct {
	aiHost string
	aiPort string

	// @todo: add gRPC stream here for bidirectional connection
}

// newAIHandler Function
func newAIHandler(host string, port string) *aiHandler {
	ah := &aiHandler{
		aiHost: host,
		aiPort: port,
	}

	return ah
}

// initHandler Function
func (ah *aiHandler) initHandler() error {

	return nil
}

// callAI Function
func (ah *aiHandler) callAI(api string) error {
	// @todo: add gRPC send request
	return nil
}

// processBatch Function
func processBatch(batch []string, update bool) error {
	for _, _ = range batch {

	}

	return nil
}

// performHealthCheck Function
func (ah *aiHandler) performHealthCheck() error {
	return nil
}

// disconnect Function
func (ah *aiHandler) disconnect() {
	return
}
