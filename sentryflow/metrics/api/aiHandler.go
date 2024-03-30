// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	cfg "github.com/5GSEC/SentryFlow/config"
	"github.com/5GSEC/SentryFlow/protobuf"
	"google.golang.org/grpc"
)

// ah Local reference for AI handler server
var AH *aiHandler

// aiHandler Structure
type aiHandler struct {
	aiHost string
	aiPort string

	stopChan       chan struct{}
	aggregatedLogs chan []*protobuf.APILog
	apis           chan []string

	aiStream protobuf.SentryFlowMetrics_GetAPIClassificationClient

	// @todo: add gRPC stream here for bidirectional connection
}

// init Function
func init() {
	// Construct address and start listening
	AH = NewAIHandler(cfg.AIEngineService, cfg.AIEngineServicePort)
}

// newAIHandler Function
func NewAIHandler(host string, port string) *aiHandler {
	ah := &aiHandler{
		aiHost: host,
		aiPort: port,

		stopChan:       make(chan struct{}),
		aggregatedLogs: make(chan []*protobuf.APILog),
		apis:           make(chan []string),
	}

	return ah
}

// initHandler Function
func (ah *aiHandler) InitAIHandler() bool {
	addr := fmt.Sprintf("%s:%s", cfg.AIEngineService, cfg.AIEngineServicePort)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not connect: %v | %v", err, addr)
		return false
	}
	defer conn.Close()

	// Start serving gRPC server
	log.Printf("[gRPC] Successfully connected to %s for APIMetric", addr)

	client := protobuf.NewSentryFlowMetricsClient(conn)

	log.Printf("[whywhywhy]%v", client)
	aiStream, err := client.GetAPIClassification(context.Background())
	log.Printf("[why2why2why2]%v", aiStream)
	AH.aiStream = aiStream

	done := make(chan struct{})

	go sendAPIRoutine()
	go recvAPIRoutine(done)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	close(done)

	return true
}

func InsertAccessLog(APIs []string) {
	AH.apis <- APIs
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

func sendAPIRoutine() {
routineLoop:
	for {
		select {
		case aal, ok := <-AH.aggregatedLogs:
			if !ok {
				log.Printf("[Exporter] EnvoyMetric exporter channel closed")
				break routineLoop
			}
			for _, al := range aal {
				curAPIRequest := &protobuf.APIClassificationRequest{
					Path: al.Path,
				}
				err := AH.aiStream.Send(curAPIRequest)
				if err != nil {
					log.Printf("[Exporter] Metric exporting failed %v:", err)
				}
			}

		case <-AH.stopChan:
			break routineLoop
		}
	}

	return
}

func recvAPIRoutine(done chan struct{}) error {
	for {
		select {
		default:
			event, err := AH.aiStream.Recv()
			if err == io.EOF {
				return nil
			}

			if err != nil {
				log.Printf("[Envoy] Something went on wrong when receiving event: %v", err)
				return err
			}

			log.Printf("[AIHANDLER] Receive API: %v", event)

		case <-done:
			return nil
		}
	}
}
